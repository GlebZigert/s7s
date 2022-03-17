package configuration

import(
    "time"
    "context"
    "strconv"
    "strings"
    "database/sql"

    "s7server/api"
    "s7server/dblayer"
)


const passtroughScanDeep = 30 // days
// event codes to seek passtrough in combination with zone_id != 0
// TODO: add unified code to events EC_ENTER_ZONE && EC_EXIT_ZONE
var passtroughEvents = []int64{16, 17}


/* CASES:
* 1. Update card - ONE user affected
* 2. Update rule - ONE rule affected
* 3. Update user links (devs or rule) - ONE user affected
* 4. Update group links (devs or rules) - MULTIPLE users affected
*/


func (cfg *Configuration) forbiddenVisitorsDetector(ctx context.Context) {
    forbiddenVisitors := make(map[int64] int64)
    timer := time.NewTimer(5 * time.Second) // initial delay
    for {
        select {
            case <-ctx.Done():
            return // TODO: return -> break?
            case <-timer.C:
                // TODO: more precise error reporting?
                cfg.complaints <- cfg.detectForbiddenVisitors(forbiddenVisitors)
        }

        now := time.Now()
        next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 1 + now.Minute(), 3, 0, time.Local)
        timer.Reset(next.Sub(now))
    }
    
    cfg.Log("detectForbiddenVisitors() stopped") // TODO: unreachable
}

func (cfg *Configuration) detectForbiddenVisitors(forbiddenVisitors map[int64] int64) (err error) {
    // TODO: optimize performance?
    var events api.EventsList
    forbidden := make(map[int64] int64)
    loc, parents, err := cfg.visitorsLocation() // [userId] => zoneId
    if nil != err {return}
    zones, err := cfg.allowedZones() // [zoneId] = > [user, user, ...]
    if nil != err {return}

    for userId, zoneId := range loc {
        if 1 == zoneId {
            continue // skip "Внешняя территория"
        }
        allowed := false
        for _, uid := range zones[zoneId] {
            if uid == userId { // user is in place
                allowed = true
                break
            }
        }
        
        if !allowed {
            /*cfg.cache.RLock()
            users := append(cfg.cache.parents[parentId], parentId, userId)
            cfg.cache.RUnlock()*/
            parentId := parents[userId]
            users, err := cfg.cache.expandParents(userId, parentId)
            if nil != err {
                return err // shadowed
            }
            allowed = intersected(zones[zoneId], users)
        }

        if allowed {
            continue // user or it's parents are allowed
        }

        // forbidden user detected!
        forbidden[userId] = zoneId
        if _, ok := forbiddenVisitors[userId]; !ok { // new violation
            forbiddenVisitors[userId] = zoneId
            // new violation events
            events = append(events, api.Event{
                Class: api.EC_ACCESS_VIOLATION,
                UserId: userId,
                ZoneId: zoneId,
                Text: api.DescribeClass(api.EC_ACCESS_VIOLATION)})
        }
    }
    // check violation is ended
    for userId := range forbiddenVisitors {
        if _, ok := forbidden[userId]; !ok {
            // violation id ended
            events = append(events, api.Event{
                Class: api.EC_ACCESS_VIOLATION_ENDED,
                UserId: userId,
                ZoneId: forbiddenVisitors[userId],
                Text: api.DescribeClass(api.EC_ACCESS_VIOLATION_ENDED)})
            delete(forbiddenVisitors, userId)        
        }
    }
    //cfg.Log("FORBIDDEN:", forbidden)
    //cfg.Log("EVENTS:", events, "is nil:", nil == events)
    if nil != events {
        cfg.Broadcast("Events", events)
    }
    return
}

func intersected(a, b []int64) bool {
    for _, x := range a {
        for _, y := range b {
            if x == y {
                return true
            }
        }
    }
    return false
}


func (cfg *Configuration) EnterZone(event api.Event) {
    cfg.Broadcast("EnterZone", &event)
}

func (cfg *Configuration) UserByCard(card string) (userId int64, err error) {
    fields := dblayer.Fields{"user_id": &userId}
    emCard, _ := encodeCard(card)
    err = db.Table("cards").Seek("card = ?", emCard).First(nil, fields)
    if sql.ErrNoRows == err {
        err = nil // NoRows is not really an error
    }
    return
}


func encodeCard(card string) (emCard, pin string) {
    p1, _ := strconv.ParseInt(card[len(card)-6:len(card)-4], 16, 32)
    p2, _ := strconv.ParseInt(card[len(card)-4:], 16, 32)
    emCard = strconv.FormatInt(p1, 10) + "," + strconv.FormatInt(p2, 10)
    pin = strings.Replace(strings.TrimLeft(card, "0"), "A", "0", -1)
    return 
}
// returns 0 if forbidden
// or user_id > 0 if confirmed
// return zone_id ?
// TODO: return errors: user not found, AP not allowed, timerange is incorrect
func (cfg *Configuration) RequestPassage(zoneId int64, card, pin string) (userId, errCode int64, err error) {
    defer func () {cfg.complaints <- err}()
    
    // encode to EM
    emCard, _ := encodeCard(card)
    //cfg.Log("EM", emCard)
    /*tabName := "cards"
    if "64,48690" == emCard { // buggy card emulation
        tabName += "$"
    }*/
    // 1. find card
    var dbPin string
    fields := dblayer.Fields{"user_id": &userId, "pin": &dbPin}
    err = db.Table("cards").Seek("card = ?", emCard).First(nil, fields)
    if sql.ErrNoRows == err {
        err = nil // NoRows is not really an error
        errCode = api.ACS_UNKNOWN_CARD // user (card) not found
        return
    }
    if nil != err {return}
    
    // pin requred
    if "" != dbPin {
        if "" == pin {
            errCode = api.ACS_PIN_REQUIRED // pin required
        } else {
            emCard, pin = encodeCard(pin)
            if dbPin != emCard && dbPin != pin {
                errCode = api.ACS_WRONG_PIN // wrong pin
            }
        }
    }
    
    if 0 != errCode {
        return
    }
    
    // 2. get user info and and it's parent_id
    user, err := cfg.GetUser(userId) // TODO: handle err
    if nil != err {return}
    if nil == user {
        errCode = api.ACS_UNKNOWN_CARD // card found, but unknown or deleted user - WTF?
        return
    }

    users, err := cfg.cache.expandParents(userId, user.ParentId) // TODO: handle err
    if nil != err {return}
    
    
    // 3. Find applicable zone, rules & timeranges
    allowedZone, err := cfg.checkZone(zoneId, users)
    if nil != err {return}
    if !allowedZone {
        errCode = api.ACS_ACCESS_DENIED
        return
    }

    // 4. check for anti-passback
    locId, err := cfg.visitorLocation(userId) // TODO: handle err
    if nil != err {return}
    if locId == zoneId {
        errCode = api.ACS_ANTIPASSBACK // already in zone
        return
    }
    
    // 5. check visitors limit
    overflow, err := cfg.checkMaxVisitors(zoneId)
    if nil != err {return}
    if overflow {
        errCode = api.ACS_MAX_VISITORS
        return
    }

    return
}

func (cfg *Configuration) checkZone(zoneId int64, users []int64) (pass bool, err error) {
    wallTime := wallTime()
    monthdayTime := time.Date(
        1970, 1, wallTime.Day(),
        wallTime.Hour(), wallTime.Minute(), wallTime.Second(), 0, time.UTC)
    weekday := int(wallTime.Weekday())
    if 0 == weekday {
        weekday = 7
    }
    weekdayTime := time.Date(
        1970, 2, weekday,
        wallTime.Hour(), wallTime.Minute(), wallTime.Second(), 0, time.UTC)
    //cfg.Log(">", wallTime)
    //cfg.Log(">", monthdayTime)
    //cfg.Log(">", weekdayTime)
    
    var ruleId int64
    fields := dblayer.Fields{
        "ar.id": &ruleId}
    source := `
        timeranges tr
        JOIN accessrules ar ON tr.rule_id = ar.id
        JOIN external_links el ON el.flags = ar.id`
    cond := `
        el.link = "user-zone"
        AND el.target_id = ?
        AND ? BETWEEN ar.start_date AND ar.end_date
        AND (
               ? BETWEEN tr.'from' AND tr.'to'
            OR ? BETWEEN tr.'from' AND tr.'to'
            OR ? BETWEEN tr.'from' AND tr.'to'
        )
        AND el.source_id`
    err = db.Table(source).
        Seek(cond, zoneId, wallTime, wallTime, monthdayTime, weekdayTime, users).
        First(nil, fields)

    pass = nil == err
    if sql.ErrNoRows == err {
        err = nil // NoRows is not really an error
    }

    return
}

// location (zone) for specified user
func (cfg *Configuration) visitorLocation(userId int64) (zoneId int64, err error) {
    var timestamp int64
    fields := dblayer.Fields{
        "zone_id": &zoneId,
        "MAX(time)": &timestamp}

    startTime := time.Now().AddDate(0, 0, -passtroughScanDeep).Unix()
    err = db.Table("events").
        Seek("zone_id > 0 AND user_id = ? AND time > ? AND event", userId, startTime, passtroughEvents).
        Group("user_id").
        First(nil, fields)
    
    if sql.ErrNoRows == err {
        err = nil // NoRows is not really an error
    }

    return
}


// current location (zone) per each user ([userId] => zoneId)
// and parents[userId] => parentId
func (cfg *Configuration) visitorsLocation() (locations, parents map[int64] int64, err error) {
    var parentId, userId, zoneId, timestamp int64
    locations = make(map[int64]int64)
    parents = make(map[int64]int64)
    fields := dblayer.Fields{
        "u.parent_id": &parentId,
        "e.user_id": &userId,
        "e.zone_id": &zoneId,
        "MAX(e.time)": &timestamp}

    startTime := time.Now().AddDate(0, 0, -passtroughScanDeep).Unix()
    err = db.Table("events e JOIN users u ON e.user_id = u.id").
        Seek("u.archived = false AND e.zone_id > 0 AND e.time > ? AND e.event", startTime, passtroughEvents).
        Group("e.user_id").
        Rows(nil, fields).
        Each(func() {
            locations[userId] = zoneId
            parents[userId] = parentId
        })

    return
}

// returns true if max visitors reached
func (cfg *Configuration) checkMaxVisitors(zoneId int64) (max bool, err error) {
    var zid, timestamp, count, maxVisitors, realMax int64
    fields := dblayer.Fields{
        "e.zone_id": &zid,
        "z.max_visitors": &maxVisitors,
        "MAX(e.time)": &timestamp}
    // TODO: optimize query - COUNT(...)
    table := db.Table(`events e
                        JOIN zones z ON e.zone_id = z.id
                        JOIN users u ON e.user_id = u.id`)
    
    startTime := time.Now().AddDate(0, 0, -passtroughScanDeep).Unix()
    err = table.
        Seek("u.archived = false AND e.zone_id > 0 AND e.time > ? AND e.event",
             startTime, passtroughEvents).
        Group("e.user_id").
        Rows(nil, fields).
        Each(func() { // TODO: implement return falue to stop when count >= realMax?
            if zoneId == zid {
                count += 1
                realMax = maxVisitors
            }
        })

    //cfg.Log("MAX", realMax, "COUNT", count, "ZONE", zoneId)
    return 0 != realMax && count >= realMax, err
}

func (cfg *Configuration) entranceEvents() (list []api.Event, err error) {
    var timestamp int64
    event := new(api.Event)
    fields := eventFieldsEx(event)
    fields["MAX(e.time)"] = &timestamp
    table := db.Table(`events e
                        LEFT JOIN services s ON e.service_id = s.id
                        LEFT JOIN devices d ON e.device_id = d.id
                        LEFT JOIN zones z ON e.zone_id = z.id
                        LEFT JOIN users u ON e.user_id = u.id`)
    
    startTime := time.Now().AddDate(0, 0, -passtroughScanDeep).Unix()
    err = table.
        Seek("u.archived = false AND e.zone_id > 0 AND e.time > ? AND e.event", startTime, passtroughEvents).
        Group("e.user_id").
        Rows(nil, fields).
        Each(func () {
            list = append(list, *event)
        })

    // filter 
    return
}

// list of allowed zones for each user ([userId] => [zoneId, zoneId, ...])
func (cfg *Configuration) allowedZones() (zones map[int64] []int64, err error) {
    zones = make(map[int64] []int64)
    wallTime := wallTime()
    monthdayTime := time.Date(
        1970, 1, wallTime.Day(),
        wallTime.Hour(), wallTime.Minute(), wallTime.Second(), 0, time.UTC)
    weekday := int(wallTime.Weekday())
    if 0 == weekday {
        weekday = 7
    }
    weekdayTime := time.Date(
        1970, 2, weekday,
        wallTime.Hour(), wallTime.Minute(), wallTime.Second(), 0, time.UTC)
//    cfg.Log(">", wallTime)
//    cfg.Log(">", monthdayTime)
//    cfg.Log(">", weekdayTime)
    
    var userId, zoneId int64
    fields := dblayer.Fields{
        "el.source_id": &userId,
        "el.target_id": &zoneId}
    source := `
        external_links el
        JOIN accessrules ar ON ar.id = el.flags
        JOIN timeranges tr ON tr.rule_id = ar.id`
    cond := `
        el.link = "user-zone"
        AND ? BETWEEN ar.start_date AND ar.end_date
        AND (
               ? BETWEEN tr.'from' AND tr.'to'
            OR ? BETWEEN tr.'from' AND tr.'to'
            OR ? BETWEEN tr.'from' AND tr.'to'
        )`
    err = db.Table(source).
        Seek(cond, wallTime, wallTime, monthdayTime, weekdayTime).
        DistinctRows(nil, fields).
        Each(func() {
            zones[zoneId] = append(zones[zoneId], userId)
        })
    return
}

// find related access points (devices), linked by zone
/*func (cfg *Configuration) SameZoneDevices(deviceId int64) (list []int64) {
    var id int64
    fields := dblayer.Fields{"l1.source_id": &id}
    source := "external_links l1 LEFT JOIN external_links l2 ON l1.target_id = l2.target_id"
    cond := "l1.link = 'device-zone' AND l2.link = 'device-zone' AND l2.source_id = ? AND l1.source_id <> ?"
    rows, values := db.Table(source).
        Seek(cond, deviceId, deviceId).
        GetDistinct(fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, id)
    }
    return
}*/


/*func (cfg *Configuration) checkDevice(deviceId int64, users []int64) (pass bool) {
    fields := dblayer.Fields{"target_id": &deviceId}
    cond := `link = "user-device" AND flags & ? > 0 AND target_id = ? AND source_id`
    rows, _ := db.Table("external_links").
        Seek(cond, api.AM_PASSTROUGH, deviceId, users).
        Get(nil, fields, 1)
    defer rows.Close()

    if rows.Next() {
        pass = true
    }
    return
}*/

/*func (cfg *Configuration) checkRules(reader int, ids []int64) (pass bool) {
    wallTime := wallTime()
    monthdayTime := time.Date(
        1970, 1, wallTime.Day(),
        wallTime.Hour(), wallTime.Minute(), wallTime.Second(), 0, time.UTC)
    weekdayTime := time.Date(
        1970, 2, int(wallTime.Weekday()),
        wallTime.Hour(), wallTime.Minute(), wallTime.Second(), 0, time.UTC)
//    cfg.Log(">", wallTime)
//    cfg.Log(">", monthdayTime)
//    cfg.Log(">", weekdayTime)
    
    var ruleId int64
    fields := dblayer.Fields{
        "ar.id": &ruleId}
    source := `
        timeranges tr
        LEFT JOIN accessrules ar ON tr.rule_id = ar.id
        LEFT JOIN external_links el ON el.target_id = ar.id`
    cond := `
        el.link = "user-rule"
        AND ? & tr.direction > 0
        AND ? BETWEEN ar.start_date AND ar.end_date
        AND (
               ? BETWEEN tr.'from' AND tr.'to'
            OR ? BETWEEN tr.'from' AND tr.'to'
            OR ? BETWEEN tr.'from' AND tr.'to'
        )
        AND el.source_id`
    rows, _ := db.Table(source).
        Seek(cond, reader, wallTime, wallTime, monthdayTime, weekdayTime, ids).
        Get(nil, fields, 1)
    
    defer rows.Close()

    if rows.Next() {
        pass = true
    }
    return
}*/


// 
/*func (cfg *Configuration) GetAccessRules(serviceId int64) (rules []*Rule) {
    allRules := cfg.loadRules()
    for _, rule := range allRules {
        if rule.Priority < 100 { // basic rules, stored on-board
            rules = append(rules, rule)
        }
    }
    return
}*/

/*func (cfg *Configuration) GroupsWithLinks(serviceId int64) []*User {
    // 1.   Load [user, device||rule] pairs for serviceId (by scope_id)
    // 1.1. load devices by IDs
    // 2.   Merge group devices and user devices
    // 2.   load users
    // 3.   load rules and timeranges

    // rules per each user
    rules := cfg.getTargetsByScope("user-rule", 0)
    // devices per each user
    devices := cfg.getTargetsByScope("user-device", serviceId)

    ids := []int64{}
    // iterate trough devices: rules without devices are unusible
    for id, _ := range devices {
        ids = append(ids, id)
    }
    
    //log.Println("### DEVS:", devices)
    //log.Println("### RULES:", rules)
    //log.Println("### IDS:", ids)

    users := cfg.getUsersById(ids)
    for _, user := range users {
        for _, id := range devices[user.Id] {
            user.Devices = append(user.Devices, UserLink{serviceId, id})
        }
        for _, id := range rules[user.Id] {
            user.AccessRules = append(user.AccessRules, UserLink{0, id})
        }
        //log.Println("###", user)
    }
    return users
}*/

/*
cache {
    devLinks[groupId] = [UserLink, UserLink, ...]
    ruleLinks[groupId] = [ruleId, ruleId, ...]
}
*/

// returns array of targets per each GROUP with inheritance
// map[groupId] => [[target_id, flags], [target_id, flags]...]
/*func (cfg *Configuration) getGroupLinks(linkType string) map[int64] []UserLink {
    //list := make(map[int64] []int64) // [user_id] => devices[5, 6, 7, 8]
    groups := make(map[int64] []UserLink)
    var groupId, parentId int64
    var link UserLink
    
    fields := dblayer.Fields{
        "u.id": &groupId,
        "u.parent_id": &parentId,
        "el.scope_id": &link[0],
        "el.target_id": &link[1],
        "el.flags": &link[2]}

    // ### 1. get all users & groups with linked targets
    
    rows, values := db.Table("external_links el LEFT JOIN users u ON el.source_id = u.id").
        // TODO: children_id is always greater than parent_id, but until transfer between groups happens (or use timestamp for group change?)
        Order("u.id"). 
        Seek(`u.type = 1 AND u.archived = false AND el.link = ?`, linkType).
        Get(nil, fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        if _, ok := groups[groupId]; ok {
            groups[groupId] = append(groups[groupId], link)
        } else {
            groups[groupId] = []UserLink{link}
        }
        groups[groupId] = append(groups[groupId], groups[parentId]...)
    }
    return groups
}

func (cfg *Configuration) getUsersById(ids []int64) []*User{
    var list []*User
    user := new(User)
    //cfg.tables["users"].query("fields").where("cond")
    //db.Table("users").Find("cond").Get(nil, "list")
    fields := dblayer.Fields {
        "id":           &user.Id,
        "archived":     &user.Archived,
        "parent_id":    &user.ParentId,
        "type":         &user.Type,
        "name":         &user.Name,
        "surename":     &user.Surename,
        "login":        &user.Login}

    rows, values := db.Table("users").Seek(ids).Get(nil, fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        tmp := *user
        list = append(list, &tmp)
    }

    return list
}*/

/*
func unique(values []int64) (list []int64) {
    keys := make(map[int64]struct{})
    for _, item := range values {
        if _, ok := keys[item]; !ok {
            keys[item] = struct{}{}
            list = append(list, item)
        }
    }    
    return list
}
*/

/*
// users and groups allowed to pass
func (cfg *Configuration) usersAllowedToPass(deviceId int64) (users []int64) {
    var userId int64
    rows, values := db.Table("external_links").
        Seek(`link = "user-device" AND flags & ? > 0 AND target_id = ?`, api.AM_PASSTROUGH, deviceId).
        Get(dblayer.Fields{"source_id": &userId})
    defer rows.Close()

    var list []int64
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, userId)
    }

    cfg.cache.RLock()
    for _, userId = range list {
        users = append(users, userId)
        users = append(users, cfg.cache.children[userId]...)
    }
    cfg.cache.RUnlock()
    //users = unique(users)
    return
}

// map[card] => [*rule1, *rule2, ...]
func (cfg *Configuration) GetCards(deviceId int64) map[string][]*Rule{
    // 1. get users and groups associated with device as AM_PASSTROUGH
    // 1.1. extend them with parents
    users := cfg.usersAllowedToPass(deviceId)
    //cfg.Log("U>>>", users)
    
    // 2. load rules
    rules := make(map[int64]*Rule)
    for _, rule := range cfg.loadHWRules() {
        rules[rule.Id] = rule
    }
    //cfg.Log("R>>>", rules)
    
    // 3. get cards by user_id or parent_id where rule.priority = 0
    var card string
    var ruleId int64
    fields := dblayer.Fields{
        "c.card": &card,
        "r.id": &ruleId}
    
    source := `
        cards c
        LEFT JOIN users u ON c.user_id = u.id
        LEFT JOIN external_links l ON c.user_id = l.source_id
        LEFT JOIN accessrules r ON l.target_id = r.id`
    list := "?" + strings.Repeat(", ?", len(users)-1)
    cond := `
        l.link = "user-rule"
        AND r.priority = 0
        AND (
            u.id IN(` + list + `)
            OR u.parent_id IN(` + list + `)
        )`
    args := make([]interface{}, 0, 2 * len(users) + 1)
    args = append(args, cond)
    for _, userId := range users {args = append(args, userId)}
    for _, userId := range users {args = append(args, userId)}
    //cfg.Log("!!!!!!!!!!!!!!!", args)
    
    params := []interface{}{cond}
    params = append(params, users...)
    params = append(params, users...)
    rows, values := db.Table(source).
        Seek(args...).
        Get(nil, fields)
    defer rows.Close()

    cards := make(map[string][]*Rule)
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        cards[card] = append(cards[card], rules[ruleId])
    }
    //cfg.Log("C>>>", cards)
    return cards
}

*/


////////////////////////////////////////////////////////////////////////

func wallTime() time.Time {
    now := time.Now().Truncate(time.Second)
    _, utcOffset := now.Zone()
    return now.Add(time.Second * time.Duration(utcOffset)).UTC()
}