package configuration

import (
    "fmt"
    "time"
    "sync"
    "errors"
    "strconv"
    "strings"
    "context"
    "encoding/json"
    "database/sql"
    
    "s7server/api"
    "s7server/dblayer"
)

const (
    authSalt = "iATdT7R4JKGg1h1YeDPp:Zl6fyUw10sgh1EGxnyKQ"
    qTimeout = 1500 // db query default timeout, msec // TODO: atomic increase during backup?
    dbOpenAttempts = 2 // 2 => original + 2 prev backups
    connParams = "?_synchronous=NORMAL&_journal_mode=WAL" // &_locking_mode=EXCLUSIVE //&_busy_timeout=10000
    maxDBBackups = 8
    maxDBEvents = 200e3 // max events count to store in database
    minBackupsInterval = 5 * time.Minute
    dbBackupInterval = 12 * time.Hour
)

var (
    db dblayer.DBLayer
    tooFrequentBackups = errors.New("Too frequent backups")
)


func init() {
    //dblayer.LogTables = []string{}
    dblayer.LogTables = []string {
        //"algorithms",
        /*"events",
        "external_links",
        "services"*/}
}

// >>>>>>>>>>>>>>> HELPER FOR CORE EXPORT >>>>>>>>>>>>>>>>>>>
var core ConfigAPI
var coreLock sync.RWMutex

func ExportCore(c *ConfigAPI) {
    coreLock.Lock()
    defer coreLock.Unlock()
    if nil == *c {
        *c = core
    }
}
// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

func (cfg *Configuration) Run(c ConfigAPI) (err error) {
    var ctx context.Context
    core = c
    ctx, cfg.Cancel = context.WithCancel(context.Background())

    if err = cfg.openDatabase(dbOpenAttempts); nil != err {
        err = fmt.Errorf("Database problem: %w", err)
        return
    }
    cfg.Log("Database ready")

    //err = cfg.backupDatabase()
    
    //lastTime, err := cfg.lastBackupTime()
    //cfg.Log("LT:", err, time.Now().Sub(lastTime))
    
    //return fmt.Errorf("Stopped.")

    //cfg.DB.SetMaxOpenConns(1)
    
    /*if err := cfg.cacheRelations(); nil != err {
        cfg.lastError = fmt.Errorf("Database problem: %w", err)
        return
    }*/
    
    cfg.setupApi()
    cfg.complaints = make(chan error, 100)

    go cfg.ErrChecker(ctx, cfg.complaints, api.EC_SERVICE_READY, api.EC_SERVICE_FAILURE)
    go cfg.forbiddenVisitorsDetector(ctx)
    go cfg.dbBackupSheduler(ctx)
    cfg.Log("CORE STARTED")
    return
}

func (cfg *Configuration) Shutdown() {
    cfg.Log("Shutting down...")
    cfg.Cancel()
    // wait before closing db
    time.Sleep(1 * time.Second) // TODO: upgrade to wait group?
    db.Close()
    
    // lock already is not required, all is stopped
    if "" != cfg.nextDatabase {
        cfg.applyBackup()
    }
}

// for interface compatibility
func (cfg *Configuration) GetList() []int64 {
    return nil
}

func (cfg *Configuration) Get() ([]*api.Settings, error) {
    return cfg.loadServices()
}

func (cfg *Configuration) replyLoop(c chan interface{}) {
    for msg := range c {
        c <- msg
    }
}

///////////////////// API INTERFACE //////////////////////
func (cfg *Configuration) StartNewShift(userId int64) (err error) {
    defer func () {cfg.complaints <- de(err, "StartNewShift")}()
    shiftId, err := cfg.currentShiftId(userId)
    if shiftId == 0 && nil == err {
        events := api.EventsList{api.Event{
            Class: api.EC_USER_SHIFT_STARTED,
            UserId: userId}}
        cfg.Broadcast("Events", events)
    }
    return
}

func (cfg *Configuration) CompleteShift(userId int64) (err error) {
    defer func () {cfg.complaints <- de(err, "CompleteShift")}()
    shiftId, err := cfg.currentShiftId(userId)
    if shiftId > 0 && nil == err {
        events := api.EventsList{api.Event{
            Class: api.EC_USER_SHIFT_COMPLETED,
            UserId: userId}}
        cfg.Broadcast("Events", events)
    }
    return
}


func (cfg *Configuration) ProcessEvents(events api.EventsList) (err error) {
    defer func () {cfg.complaints <- de(err, "ProcessEvents")}()
    for i := 0; i < len(events) && nil == err; i++ {
        err = cfg.processEvent(&events[i])
    }
    if nil == err {
        // store events
        err = cfg.dbLogEvents(events)
    }

    return
}

// add extra info to event: algos, text/names etc.
func (cfg *Configuration) processEvent(e *api.Event) (err error) {
    //TODO: maybe implementing scanner & valuer is a better choice?
    var commands [][4]int64 // [serviceId, deviceId, commandCode, argument]
    if 0 == e.Time {
        e.Time = time.Now().Unix()
    }
    if "" == e.Text {
        if 0 == e.Event && e.Class > 0 {
            e.Text = api.DescribeClass(e.Class)
        } else {
            e.Text = api.DescribeEvent(e.Event)
        }
    }

    e.Algorithms, err = cfg.findDevAlgorithms(e)
    if nil != err {
        return
    }
    for i := range e.Algorithms {
        commands = append(commands, [4]int64{
            e.Algorithms[i].TargetServiceId,
            e.Algorithms[i].TargetDeviceId,
            e.Algorithms[i].Command,
            e.Algorithms[i].Argument})
    }
    if len(commands) > 0 {
        cmds, _ := json.Marshal(commands)
        //check(err)
        e.Commands = string(cmds)
    }
    
    // prepare event for broadcasting
    
    if "" == e.ServiceName {
        if 0 == e.ServiceId {
            e.ServiceName = "Система"
        } 
    }
    if e.UserId > 0 && "" == e.UserName {
        var user *User
        user, err = cfg.GetUser(e.UserId)
        if nil != err {
            return
        }
        if nil != user { // user was found
            e.UserName = user.Name + " " + user.Surename
        }
    }
    if 0 == e.ZoneId && e.DeviceId > 0 {
        // check that device in zone
        e.ZoneId, err = cfg.deviceZone(e.ServiceId, e.DeviceId)
        if nil != err {
            return
        }
    }
    if e.ZoneId > 0 && "" == e.ZoneName {
        var zone *Zone
        zone, err = cfg.getZone(e.ZoneId)
        if nil != err {
            return
        }
        e.ZoneName = zone.Name
    }
    if e.DeviceId > 0 && "" == e.DeviceName {
        e.DeviceName, err = cfg.getDeviceName(e.DeviceId)
    }
    
    return
}

/*func (cfg *Configuration) CheckEvent(e *api.Event) (algos []Algorithm) {
    return cfg.findAlgorithms(e.ServiceId, e.DeviceId, e.FromState, e.Event)
}*/

func (cfg *Configuration) ZoneDevices(zoneId, userId int64, devList []int64) (devMap map[int64][]int64) {
    var err error
    defer func () {cfg.complaints <- de(err, "ZoneDevices")}()
    
    list, err := core.LoadLinks(zoneId, "zone-device")
    if nil != err {return}

    ds := make(map[int64] int64) // device-service
    for i := range list {
        ds[list[i][1]] = list[i][0]
    }
    
    var devices []int64
    for _, d := range devList {
        if _, ok := ds[d]; ok {
            devices = append(devices, d)
        }
    }

    if 0 == len(devices) {
        // Zone is empty or all devices are "offline"
        return // return empty list, nothing to control
    }
    
    filter, err := core.Authorize(userId, devices)
    if nil != err {return}
    
    devMap = map[int64] []int64{0: []int64{}} // 0 - forbidden devices
    for _, id := range devices {
        // filter[0] > 0 => all id are acceptable
        if 0 == filter[0] && 0 == filter[id] & api.AM_CONTROL {
             devMap[0] = append(devMap[0], id)
        }
        sid := ds[id]
        if 0 == len(devMap[sid]) {
            devMap[sid] = []int64{id}
        } else {
            devMap[sid] = append(devMap[sid], id)
        }
    }
    return
}

// get userId by login and password
func (cfg *Configuration) Authenticate(login, token string) (id, role int64, err error) {
    defer func () {cfg.complaints <- de(err, "Authenticate")}()
    var userToken string
    fields := dblayer.Fields {
        "id": &id,
        "role": &role,
        "token": &userToken}

    rows, values, err := db.Table("users").
        Seek("login = ? AND role > ? AND archived = ?", login, 0, 0).
        Get(nil, fields)

    if nil != err {
        return
    }
    defer rows.Close()

    if rows.Next() {
        err = rows.Scan(values...)
        if nil == err && userToken != token {
            id = 0
        }
    }
    if nil == err {
        err = rows.Err()
    }
    return
}

// not all devices are really "deleted", so don't use serviceId and discard "suspended"
// devices == nil => check services
func (cfg *Configuration) Authorize(userId int64, devices []int64) (list map[int64]int64, err error) {
    defer func () {cfg.complaints <- de(err, "Authorize")}()
    list = make(map[int64]int64) // [deviceId] => flags
    
    if 0 == userId {
        list[0] = api.AM_CONTROL
        return
    }

    user, err := cfg.GetUser(userId)
    if nil == user /*|| nil != err*/ {return} // not found or db error

    switch user.Role {
        case api.ARM_ADMIN: list[0] = api.AM_CONTROL
        case api.ARM_SECRET: list[0] = api.AM_WATCH
    }

    if len(list) > 0 || len(devices) == 0 {
        return // in any case, if list[i] == 0, then user can't do anything
    }
    /*cfg.cache.RLock()
    users := append(cfg.cache.parents[user.ParentId], user.Id, user.ParentId)
    cfg.cache.RUnlock()*/
    users, err := cfg.cache.expandParents(user.Id, user.ParentId)
    //cfg.Log("EXPAND", users, err)
    if nil != err {
        return
    }

    var deviceId, flags int64
    flags = 1 // for services list
    var params []interface{}
    fields := dblayer.Fields{"target_id": &deviceId, "flags": &flags}
    cond := "target_id IN(?" +
        strings.Repeat(", ?", len(devices)-1) +
        ") AND link = 'user-device' AND source_id"
    params = append(params, cond)

    for _, v := range devices {
        params = append(params, v)
    }
    params = append(params, users)

    rows, values, err := db.Table("external_links").
        Seek(params...).
        Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()

    for rows.Next() {
        err = rows.Scan(values...)
        if nil != err {
            break
        }
        val, _ := list[deviceId]
        list[deviceId] = val | flags
    }
    if nil == err {
        err = rows.Err()
    }
    //cfg.Log("### AUTHORIZED:", userId, devices, list)
    return
}

/*
func (cfg *Configuration) Subscribe() chan interface{} {
    cfg.Lock()
    defer cfg.Unlock()
    c := make(chan interface{}, 10) // WARN: number of initial packets to avoid deadlock
    cfg.subscribers = append(cfg.subscribers, c)
    return c
}

func (cfg *Configuration) Unsubscribe(c chan interface{}) {
    cfg.Lock()
    defer cfg.Unlock()
    for i, subs := range cfg.subscribers {
        if subs == c {
            last := len(cfg.subscribers) - 1
            cfg.subscribers[i] = cfg.subscribers[last]
            cfg.subscribers = cfg.subscribers[:last]
        }
    }
}
*/
// card - ruleId - deviceId
/*func (cfg *Configuration) GetActiveCards(serviceId int64) (cards map[int64][]string) {
    cards = make(map[int64][]string)
    return
}*/


////////////////////////////////// LEGACY /////////////////////////////////

// returns []*UserSummary: {UserId: int64, Cards: []string, Rules: []int64, Devices: []int64}
/*func (cfg *Configuration) DescribeUsers(serviceId int64) (summaries map[int64]*UserSummary) {
    // TODO: maybe lock on a caller side?
    cfg.RLock()
    defer cfg.RUnlock()
    
    // 1. get cards
    var userId int64
    var card string
    var summary *UserSummary
    //var summary = new(UserSummary)
    summaries = make(map[int64]*UserSummary)


    // 2. get links for user-devices and user-rules
    rules := cfg.getTargetsByScope("user-rule", 0)
    devices := cfg.getTargetsByScope("user-device", serviceId)
    
    // 3. get cards
    // TODO: seek cards only for active users
    rows, values := db.Table("cards").Get(nil, dblayer.Fields{"id": &userId, "card": &card})
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        if summary = summaries[userId]; summary == nil {
            r, okD := devices[userId]
            d, okR := rules[userId]
            if okD && okR {
                summary = new(UserSummary)
                summary.Devices = d
                summary.Rules = r
                summary.Cards = append(summary.Cards, card)
                // store
                summaries[userId] = summary
            }
        } else {
            summary.Cards = append(summary.Cards, card)
        }
    }
    
    return
}*/




/*
func (cfg *Configuration) getTargetsByScope(target string, scopeId int64) []UserLink {
    list := make([]UserLink, 0)
    ids := make([]int64[], 0)
    groups := make(map[int64] []int64)
    var userId, targetId, parentId int64
    

    table := db.Table("user_links ul LEFT JOIN users u ON ul.user_id = u.id")
    fields := dblayer.Fields{
        "u.id": &userId,
        "u.parent_id": &parentId,
        "ul.target_id": &targetId}

    // ### 1. find all groups with linked targets    
    
    rows, values := table.
        Order("u.id"). // children_id can't be greater than parent_id
        Seek(
            `u.archived = ? AND u.type = ? AND ul.target = ? AND ul.scope_id = ?`,
            0, 1, target, scopeId).
        Get(nil, fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        if _, ok := groups[userId]; ok {
            groups[userId] = append(groups[userId], targetId)
        } else {
            groups[userId] = []int64{targetId}
        }
        if _, ok := groups[parentId]; ok {
            // inheritance
            groups[userId] = append(groups[userId], groups[parentId]...)
        }
        ids = append(ids, userId)
    }
    
    // ### 2 find groups with no links for inheriance
    
    fld := dblayer.Fields{
        "id": &userId,
        "parent_id": &parentId}
    rows, values = db.Table("users").
        Order("id"). // children_id can't be greater than parent_id
        Seek("archived = ? AND type = ?", 0, 1).
        Get(fld)
    
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        if _, ok := groups[userId]; !ok {
            if _, ok = groups[parentId]; ok {
                groups[userId] = groups[parentId] // inheritance
                ids = append(ids, userId)
            }
        }    
    }
    log.Println("### G2:", groups)
    
    // ### 3. find users with personal links and children of groups from #1
    
    rows, values = table.
        Seek(
            `u.archived = ? AND u.type <> ? AND ul.target = ? AND ul.scope_id = ? OR parent`,
            0, 1, target, scopeId, ids).
        Get(nil, fields)
    defer rows.Close()
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, UserLink{userId, targetId})
        if _, ok := groups[parentId]; ok {
            for _, targetId = range groups[parentId] {
                list = append(list, UserLink{userId, targetId})
            }
        }
    }
    return list
}
*/



func (cfg *Configuration) setupApi() {
    cfg.Api(map[string] api.Action {
        "UpdateSettings": cfg.updateSettings,
        "ListSettings": cfg.listSettings,
        
        "ListBackups": cfg.listBackups,
        "MakeBackup": cfg.makeBackup,
        "RestoreBackup": cfg.restoreBackup,
        
        //"ListLocations": cfg.listLocations,
        "ResetAlarm": cfg.resetAlarm,
        "RunAlarm": cfg.runAlarm,
        
        "CompleteShift": cfg.completeShift,
        
        "LoadJournal": cfg.loadJournal,
        "ListEvents": cfg.listEvents,
        "DescribeEvent": cfg.describeEvent,
        
        "ListAlgorithms": cfg.listAlgorithms,
        "UpdateAlgorithm": cfg.updateAlgorithm,
        "DeleteAlgorithm": cfg.deleteAlgorithm,

        "ListZones": cfg.listZones,
        "UpdateZone": cfg.updateZone,
        "DeleteZone": cfg.deleteZone,

        "ListMaps": cfg.listMaps,
        "UpdateMap": cfg.updateMap,
        "DeleteMap": cfg.deleteMap,
        //"UpdateMapPosition": cfg.updateMapPosition,

        
        "ListRules": cfg.listRules,
        "UpdateRule": cfg.updateRule,
        "DeleteRule": cfg.deleteRule,
        
        "UpdateService": cfg.updateService,
        "DeleteService": cfg.deleteService,
        
        "UserInfo": cfg.userInfo,
        "ListUsers": cfg.listUsers,
        "UpdateUser": cfg.updateUser,
        "DeleteUser": cfg.deleteUser})
}

// check database error
/*func (cfg *Configuration) cdbe(err error) {
    if nil != err {
        cfg.Log("Database problem:", err)
    }
}*/

func (cfg *Configuration) Encrypt(text string) (string, error) {
    return encrypt(text)
}
    
func (cfg *Configuration) Decrypt(cipher string) (string, error) {
    return decrypt(cipher)
}

//////////////////////////////////////////////////////////////////////

// extract last 3 bytes from card number (card format: 000,00000 or 1234567890)
func keyFromCard(card string) (key int64) {
    if strings.HasPrefix(card, "#") {
        key, _ = strconv.ParseInt(card[1:], 16, 64)
    } else if strings.ContainsRune(card, ',') {
        pair := strings.Split(card, ",")
        n, _ := strconv.ParseInt(pair[0], 10, 64)
        key, _ = strconv.ParseInt(pair[1], 10, 64)
        key = 65536 * n + key
    } else {
        key, _ = strconv.ParseInt(card, 10, 64)
    }
    
    key = key & 0xFFFFFF
    return
}

// describe error
func de(err error, desc string) error {
    if nil != err {
        return fmt.Errorf("%s: %w", desc, err)
    } else {
        return nil
    }
}

func completeTx(tx *sql.Tx, err error) {
    if nil != err {
        tx.Rollback()
    } else {
        tx.Commit()
    }
}

func findString(s string, list []string) int {
    for i := range list {
        if list[i] == s {
            return i
        }
    }
    return -1
}

func findNumber(n int64, list []int64) int {
    for i := range list {
        if list[i] == n {
            return i
        }
    }
    return -1
}
