package configuration

import (
    "fmt"
    "time"
    "strings"
    "context"
    "encoding/json"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"

    "../../api"
    "../../dblayer"
)

const (
    authSalt = "iATdT7R4JKGg1h1YeDPp:Zl6fyUw10sgh1EGxnyKQ"
    qTimeout = 500 // db query default timeout, msec
)

var db dblayer.DBLayer

func init() {
    //dblayer.LogTables = []string{}
    dblayer.LogTables = []string {
        //"algorithms",
        /*"events",
        "external_links",
        "services"*/}
}

func (cfg *Configuration) Run(_ ConfigAPI) (err error) {
    var ctx context.Context
    ctx, cfg.Cancel = context.WithCancel(context.Background())

    dbFilename := cfg.GetStorage() + ".db?_synchronous=NORMAL&_journal_mode=WAL" // _busy_timeout=10000
    
    if err = cfg.openDB(dbFilename); nil != err {
        err = fmt.Errorf("Database problem: %w", err)
        return
    }
    
    //cfg.DB.SetMaxOpenConns(1)
    
    /*if err := cfg.cacheRelations(); nil != err {
        cfg.lastError = fmt.Errorf("Database problem: %w", err)
        return
    }*/
    
    cfg.setupApi()

    go cfg.forbiddenVisitorsDetector(ctx)
    return
}

func (cfg *Configuration) Shutdown() {
    cfg.Log("Shutting down...")
    cfg.Cancel()
    db.Close()
}

// for interface compatibility
func (cfg *Configuration) GetList() []int64 {
    return nil
}

func (cfg *Configuration) Get() []*api.Settings {
    return cfg.loadServices()
}

func (cfg *Configuration) replyLoop(c chan interface{}) {
    for msg := range c {
        c <- msg
    }
}

func (cfg *Configuration) notifySubscribers(msg interface{}) {
    for _, subscriber := range cfg.subscribers {
        subscriber <- msg
    }
}

///////////////////// API INTERFACE //////////////////////
func (cfg *Configuration) StartNewShift(userId int64) (err error) {
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
        e.UserName = user.Name + " " + user.Surename
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
        e.DeviceName, err = cfg.getDeviceName(e.ServiceId, e.DeviceId)
    }
    
    return
}

/*func (cfg *Configuration) CheckEvent(e *api.Event) (algos []Algorithm) {
    return cfg.findAlgorithms(e.ServiceId, e.DeviceId, e.FromState, e.Event)
}*/


// get userId by login and password
func (cfg *Configuration) Authenticate(login, token string) (id, role int64) {
    var userToken string
    fields := dblayer.Fields {
        "id": &id,
        "role": &role,
        "token": &userToken}

    rows, values, _ := db.Table("users").
        Seek("login = ? AND role > ? AND archived = ?", login, 0, false).
        Get(nil, fields)
    defer rows.Close()

    if rows.Next() {
        err := rows.Scan(values...)
        if nil == err {
            if userToken != token {
                id = 0
            }
        }
    } else if "buro" == login {
        user := User{
            Type: api.UT_PERSONAL,
            Role: api.ARM_BUREAU,
            Name: "Бюро пропусков",
            Login: "buro",
            Password: "Start7"}
        cfg.dbUpdateUser(&user, nil)
        id = user.Id
    }
    return
}


// not all devices are really "deleted", so don't use serviceId
// devices == nil => check services
func (cfg *Configuration) Authorize(userId int64, devices []int64) (list map[int64]int64, err error) {
    list = make(map[int64]int64) // [deviceId] => flags
    
    user, _ := cfg.GetUser(userId) // TODO: handle err
    //cfg.Log("AUTHORIZING:", userId, user)
    if user != nil {
        switch user.Role {
            case api.ARM_ADMIN: list[0] = api.AM_CONTROL
            case api.ARM_SECRET: list[0] = api.AM_WATCH
        }
    }
    if nil == user || len(list) > 0 || len(devices) == 0 {
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
        err := rows.Scan(values...)
        catch(err)
        val, _ := list[deviceId]
        list[deviceId] = val | flags
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
            false, 1, target, scopeId).
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
        Seek("archived = ? AND type = ?", false, 1).
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
            false, 1, target, scopeId, ids).
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

func (cfg *Configuration) openDB(fn string) (err error) {
    //TODO: https://github.com/mattn/go-sqlite3#user-authentication
    //var db interface{}
    database, err := sql.Open("sqlite3", fn)
    if nil != err {
        return
    }
    db.Bind(database, qTimeout)
    err = db.MakeTables(tables)
    if nil != err {
        db.Close()
        return
    }
    db.MakeTables(tableUpdates) // ignore errors
    return
}

func (cfg *Configuration) setupApi() {
    cfg.Api(map[string] api.Action {
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
        
        "ListServices": cfg.listServices,  // TODO: this cmd is unneded? may be handled by dispatcher
        "UpdateService": cfg.updateService,
        "DeleteService": cfg.deleteService,
        
        "UserInfo": cfg.userInfo,
        "ListUsers": cfg.listUsers,
        "UpdateUser": cfg.updateUser,
        "DeleteUser": cfg.deleteUser})
}

// check database error
func (cfg *Configuration) cdbe(err error) {
    if nil != err {
        cfg.Log("Database problem:", err)
    }
}

//////////////////////////////////////////////////////////////////////
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

func catch(err error) {
    if nil != err {
        panic(err)
    }
}
