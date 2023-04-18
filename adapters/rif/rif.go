package rif

import (
    "os"
    "log"
    "fmt"
    "time"
    "context"
    "s7server/api"
    "s7server/adapters/configuration"
//	"strings"
)

const (
    LogExchange = false
    daysDeep = 3 // archive deep
    eventsPollInterval = 30 // seconds
    eventsPacketSize = 100
    dtLayout = "2006-01-02 15:04:05"
)


// responses to await per each command
var responses = map[int64] []int64 {
    133: []int64{1133},
    136: []int64{1136},
    137: []int64{1137},
    100: []int64{100, 110, 151, 1001, 1003},
    101: []int64{101, 111, 150, 1000, 1004}}

var core configuration.ConfigAPI

func (svc *Rif) Run(_ configuration.ConfigAPI) (err error) {
    configuration.ExportCore(&core)
    var ctx context.Context
    ctx, svc.Cancel = context.WithCancel(context.Background())
    defer svc.Cancel()
    svc.Stopped = make(chan struct{})
    defer close(svc.Stopped)

    // log
    if LogExchange {
        svc.xmlLog, err = os.OpenFile(svc.GetStorage() + ".xml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            return
        }
        defer svc.xmlLog.Close()
    }
    
    svc.waitReply = make(map[string]int64)
    svc.queryEventsChan = make(chan int64, 1)
    //defer close(svc.queryEventsChan)
    svc.complaints = make(chan error, 100)
    //defer close(svc.queryEventsChan)
    
    go svc.ErrChecker(ctx, svc.complaints, api.EC_SERVICE_READY, api.EC_SERVICE_FAILURE)
    go svc.connect(ctx)
    
    svc.setupApi()

    <-ctx.Done()
    ////////////////////////////////////////////////////////
    
    log.Println(svc.GetName(), "Shutting down...")
    svc.closeConnection()
    svc.SetServiceStatus(api.EC_SERVICE_SHUTDOWN)
    
    return
}

func (svc *Rif) Shutdown() {
    svc.RLock()
    ret := nil == svc.Cancel || nil == svc.Stopped
    svc.RUnlock()
    if ret {
        return // shutdown called before Run
    }

    svc.Cancel()
    <-svc.Stopped
}

// Return all devices IDs for user filtering
func (svc *Rif) GetList() []int64 {
    svc.RLock()
    defer svc.RUnlock()

    list := make([]int64, 0, len(svc.devices))
    
    for id := range svc.devices {
        list = append(list, id)
    }

    return list
}

func (svc *Rif) ZoneCommand(userId, zoneCommand int64, devList []int64) {
    var xml string
    svc.Log("ZONE-COMMAND", devList, zoneCommand)
    for _, devId := range devList {
        var cmd int64
        switch zoneCommand {
            case api.EC_ARMED: cmd = 137
            case api.EC_DISARMED: cmd = 136
        }
        if cmd > 0 {
            xml = svc.commandXML(devId, cmd)
        }
        if "" != xml {
            svc.queueWait(devId, cmd, userId)
            svc.SendCommand(xml)
        }
    }
}


func (svc *Rif) scanJourEvents(events []_Event) (err error){
    defer func () {svc.complaints <- de(err, "ScanJourEvents")}()
    //svc.Log(">>> EA:", len(events))
    var devId int64
    ee := make([]api.Event, 0, len(events))
    for i := range events {
        if 0 == events[i].Type {
            continue // skip groups
        }
        
        handle := svc.makeHandle(&events[i])
        // TODO: don't use GlobalDeviceId()
        devId, err = core.GlobalDeviceId(svc.Settings.Id, handle, events[i].DeviceName)
        if nil != err {
            break
        }
        ee = append(ee, api.Event {
            ExternalId: events[i].Id,
            Event:      events[i].Event,
            Class:      getClassCode(events[i].Event, events[i].Type),
            Text:       events[i].Text,
            ServiceId:  svc.Settings.Id,
            DeviceId:   devId,
            Reason:     events[i].Reason,
            Reaction:   events[i].Reaction,
            Time:       parseTime(events[i].DateTime).Unix()})
    }

    if nil != err {
        return
    }

    //svc.Log(ee)
    if len(events) > 0 {
        svc.Log("Import", len(events), "events, range", events[0].DateTime, "..", events[len(events)-1].DateTime)
    } else {
        //svc.Log("Empty events list")
    }
    err = core.ImportEvents(ee)
    if nil != err {
        // TODO: set service status "internal error"?
        return // core db problems?
    }
    if eventsPacketSize == len(events) {
        // read more events
        //time.Sleep(200 * time.Millisecond)
        svc.queryEventsChan <-events[len(events)-1].Id
    } else { // all events are pumped
        //svc.SetDBStatus("online")
        svc.SetServiceStatus(api.EC_DATABASE_READY)
    }
    return
}

func (svc *Rif) pollEventLog(ctx context.Context) {
    defer svc.Log("Events polling stopped")
    timer := time.NewTimer(eventsPollInterval * time.Second)
    for nil == ctx.Err() {
        svc.Sleep(ctx, 1 * time.Second)
        select {
            case <-ctx.Done():
                return

            case n := <-svc.queryEventsChan:
                if !timer.Stop() {
                    <-timer.C // drain the channel for reuse: https://pkg.go.dev/time#Timer.Stop
                }
                //svc.Log("Events request", n)
                svc.getEventLog(n)
            
            case <-timer.C:
                //svc.Log("Events request timer")
                svc.getEventLog(0)
        }
        timer.Reset(eventsPollInterval * time.Second)
    }
}


func (svc *Rif) getEventLog(nextId int64) (err error){
    defer func () {svc.complaints <- de(err, "GetEventLog")}()
    var cmd string
    if 0 == nextId {
        // get real last stored event id from the db
        var lastEvent *api.Event
        lastEvent, err = core.GetLastEvent(svc.Settings.Id)
        if nil != err {
            return // something happens with core database
        }
        if nil != lastEvent {
            nextId = lastEvent.ExternalId + 1
        }
    }
    if 0 == nextId {
        // get by date
        from := time.Now().AddDate(0, 0, -daysDeep).Format(dtLayout)
        cmd = fmt.Sprintf(`<RIFPlusPacket type="Commands"><Commands><Command id="10010" name="Get events" from="%s" length="%d"/></Commands></RIFPlusPacket>`, from, eventsPacketSize)
    } else {
        // get by id
        cmd = fmt.Sprintf(`<RIFPlusPacket type="Commands"><Commands><Command id="10010" name="Get events" start="%d" length="%d"/></Commands></RIFPlusPacket>`, nextId, eventsPacketSize)
    }
    //svc.Log("Getting the Event Log:", cmd)
    svc.SendCommand(cmd)
    return
}

func (svc *Rif) getGroupIds(devices []_Device) (gids []int64, err error) {
    nextGroup := int64(9e15) // ~Number.MAX_SAFE_INTEGER
    gids = make([]int64, len(devices))
    for i := 0; i < len(devices) && nil == err; i++ {
        if 0 == devices[i].Type {
            gids[i] = nextGroup
            nextGroup--
        } else {
            //fixedId = svc.getDeviceId(&devices[i])
            handle := svc.makeHandle(&devices[i])
            gids[i], err = core.GlobalDeviceId(svc.Settings.Id, handle, devices[i].Name)
            /*if "БЛ086-ИУ1" == devices[i].Name {
                svc.Log(gids[i], devices[i].Name, handle)
            }*/
            if nil != err {
                break
            }
        }
    }
    return
}

func (svc *Rif) populate(devices []_Device) (err error) {
    defer func () {svc.complaints <- de(err, "Populate")}()
    var fixedId int64
    gids, err := svc.getGroupIds(devices)
    if nil != err {return}
    //return fmt.Errorf("AAAAAAAAAAAAAAAA")
    ////////////////////////////////////////

    svc.devices = make(map[int64] *Device) // TODO: check for configuration changes!
    svc.idMap = make(map[int64] int64)

    svc.Lock()
    defer svc.Unlock()
    
    var devPath []*_Device
    links := make(map[int64][]int64)
    isReal := make(map[int64]int64)
    for i := range devices {
        fixedId = gids[i]
        svc.idMap[devices[i].Id] = fixedId
        // ignore duplicates (linked with or "nested" into devices, not groups?)
        //TODO: use if, because cant't jump more than +1 level?
        for devices[i].Level > len(devPath) - 1 {
            devPath = append(devPath, &devices[i])
        }

        if devices[i].Level < len(devPath) - 1 {
            devPath = devPath[:devices[i].Level+1]
        }

        if devices[i].Level == len(devPath) - 1 {
            devPath[devices[i].Level] = &devices[i] // set new parent
        }

        dev := svc.devices[fixedId]
        lvl := devices[i].Level - 1
        // type == 0 is a Rif bug, should be 200
        parentIsNotGroup := lvl >= 0 && lvl < len(devPath) && 0 != devPath[lvl].Type && 200 != devPath[lvl].Type
        // check for linked IU is SSOI or Rif's IU and it's not inside a group
        if parentIsNotGroup && (12 == devices[i].Type || 45 == devices[i].Type) {
            //svc.Log(">>>>>>> DUP >>>>>>>>", devices[i].Name, "in", devPath[lvl].Name, devPath[lvl].Type)
            links[devPath[lvl].Id] = append(links[devPath[lvl].Id], fixedId)
        } else if 0 == isReal[fixedId] {
            isReal[fixedId] = 1
        }
        if nil == dev /*|| devices[i].Level == 0 || lvl >= 0 && lvl < len(devPath) && devPath[lvl].Level == 0*/ {
            svc.devices[fixedId] = makeDevice(fixedId, &devices[i])
        } else if 1 == isReal[fixedId] {
            isReal[fixedId] = 2
            svc.devices[fixedId].Level = devices[i].Level
            svc.devices[fixedId].Order = devices[i].Id
        }
    }

    for i := range svc.devices {
        svc.devices[i].Links = links[svc.devices[i].Order]
    }

    svc.Log("Use", len(svc.devices), "devices of", len(devices))
    // TODO: db in not really n/a, need deep check
    svc.SetServiceStatus(api.EC_SERVICE_ONLINE, api.EC_DATABASE_UNAVAILABLE)
    return
}

func makeDevice(fixedId int64, d *_Device) *Device {
    state := State {
        Id: d.States[0].Id,
        Class: getClassCode(int64(d.States[0].Id), d.Type),
        DateTime: parseTime(d.States[0].DateTime),
        Name: d.States[0].Name,
    }
    return &Device {
        Id: fixedId,
        Order: d.Id, // original id
        Level: d.Level,
        Type: d.Type,
        Name: d.Name,
        Num: [3]int{d.Num1, d.Num2, d.Num3},
        Ip: d.Ip,
        Ip2: d.Ip2,
        Login: d.Login,
        Password: d.Password,
        Option: d.Option,
        Dk: d.Dk,
        States: [2]State{state, {}}}
}

func (svc *Rif) update(devices []_Device) {
    var fixedId int64
    var ok bool
    //log.Printf("%+v\n", len(devices))
    //svc.Log(devices)
    events := make(api.EventsList, 0, len(devices))
    //var list = make(map[int]Device, len(devices))
    svc.Lock()
    for i, _ := range devices {
        if fixedId, ok = svc.idMap[devices[i].Id]; !ok {
            svc.Log("Unknown device", devices[i])
            continue // unknown device
        }
        if _, ok = svc.devices[fixedId]; !ok {
            svc.Log("Unknown device ID", fixedId)
            continue
        }
        for j, _ := range devices[i].States {
            dt := parseTime(devices[i].States[j].DateTime)
            svc.devices[fixedId].States[1] = svc.devices[fixedId].States[0]
            svc.devices[fixedId].States[0] = State {
                Id: devices[i].States[j].Id,
                Class: getClassCode(int64(devices[i].States[j].Id), devices[i].Type),
                DateTime: dt,
                Name: devices[i].States[j].Name}
            
            //svc.Log("RE-STORE:", svc.userRequest)
            eid := int64(svc.devices[fixedId].States[0].Id)
            events = append(events, api.Event {
                FromState:  int64(svc.devices[fixedId].States[1].Id),
                Event:      eid,
                Class:      getClassCode(eid, svc.devices[fixedId].Type),
                Text:       svc.devices[fixedId].States[0].Name,
                DeviceId:   fixedId,
                DeviceName: svc.devices[fixedId].Name,
                UserId:     svc.matchUser(fixedId, eid), //svc.userRequest[eid],
                Time:       dt.Unix()})
        }
    }
    svc.Unlock()
    svc.Broadcast("Events", events)
}

func (svc *Rif) matchUser(deviceId, event int64) (userId int64) {
    key := fmt.Sprintf("%d:%d", deviceId, event)
    //svc.Log("%%%%%%%%%%%%%%%%%%%%", key, svc.waitReply)
    userId = svc.waitReply[key]
    if userId > 0 {
        delete (svc.waitReply, key)
    }
    return
}

func (svc *Rif) queueWait(deviceId, command, cid int64) {
    svc.Lock()
    defer svc.Unlock()
    if _, ok := responses[command]; ok {
        for _, code := range responses[command] {
            key := fmt.Sprintf("%d:%d", deviceId, code)
            svc.waitReply[key] = cid
        }
    }
    //svc.Log("#####################", svc.waitReply)
}

func (svc *Rif) makeHandle(arg interface{}) (handle string) {
    var t, n1, n2, n3 int
    var ip string
    d, _ := arg.(*_Device)
    e, _ := arg.(*_Event)
    if nil != d {
        t, n1, n2, n3, ip = d.Type, d.Num1, d.Num2, d.Num3, d.Ip
    }
    if nil != e {
        t, n1, n2, n3, ip = e.Type, e.Num1, e.Num2, e.Num3, e.Ip
    }
    handle = fmt.Sprintf("%d-%d-%d-%d", t, n1, n2, n3)
    if "" != ip {
        handle += "-" + ip
    }
    return
}

func (svc *Rif) setupApi() {
    svc.Api(map[string] api.Action {
        "ResetAlarm" : svc.resetAlarm,
        
        "ListDevices" : svc.listDevices,
        "ExecCommand" : svc.execCommand})
}

/*func (svc *Rif) mergeStates(id int, st *[]_State) {
    device := svc.devices[id]
    device.States = [2]State{
        {Id: (*st)[0].Id, DateTime: time2num((*st)[0].DateTime), Name: (*st)[0].Name},
        svc.devices[id].States[0]}
    //log.Println(device.States[0].DateTime)
    svc.devices[id] = device
}*/

/*
func (svc *Rif) notify(list interface{}, action string) {
    header := "{\"service\": \"" + svc.Name + "\""
    if "" != action {
        header = header + ", \"action\": \"" + action + "\""
    }
    header = header + ", \"data\": "
    //log.Println(header)
    res, err := json.Marshal(list)
    if err != nil {
        svc.queue <- header + err.Error() + "}"
    } else {
        svc.queue <- header + string(res) + "}"
    }
}*/

// describe error
func de(err error, desc string) error {
    if nil != err {
        return fmt.Errorf("%s: %w", desc, err)
    } else {
        return nil
    }
}

func parseTime(s string) time.Time {
    loc := time.Now().Location()
    dt, err := time.ParseInLocation(dtLayout, s, loc)
    if nil != err {
        // TODO: log err?
        dt = time.Now()
    }
    return dt
}

/*
func getHandle(d _Device) string {
    if 200 == d.Type {
        return fmt.Sprintf("rif-%s-%d-%d-%d-%d", d.Ip, d.Type, d.Num1, d.Num2, d.Num3)
    } else {
        return fmt.Sprintf("rif-%d-%d-%d-%d", d.Type, d.Num1, d.Num2, d.Num3)
    }
}*/
