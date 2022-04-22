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
    defer func () {svc.complaints <- err}()
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
    defer func () {svc.complaints <- err}()
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

func (svc *Rif) populate(devices []_Device) (err error) {
    defer func () {svc.complaints <- err}()
    var fixedId int64
    nextGroup := int64(9e15) // ~Number.MAX_SAFE_INTEGER

    gids := make([]int64, len(devices))
    for i := 0; i < len(devices) && nil == err; i++ {
        if 0 == devices[i].Type {
            gids[i] = nextGroup
            nextGroup--
        } else {
            //fixedId = svc.getDeviceId(&devices[i])
            handle := svc.makeHandle(&devices[i])
            gids[i], err = core.GlobalDeviceId(svc.Settings.Id, handle, devices[i].Name)
            if nil != err {
                break
            }
        }
    }
    if nil != err {
        return
    }
    //return fmt.Errorf("AAAAAAAAAAAAAAAA")
    ////////////////////////////////////////

    svc.devices = make(map[int64] *Device) // TODO: check for configuration changes!
    svc.idMap = make(map[int64] int64)

    svc.Lock()
    defer svc.Unlock()
    
    typeAtLevel := []int{}
    for i := range devices {
        fixedId = gids[i]
        state := State {
            Id: devices[i].States[0].Id,
            Class: getClassCode(int64(devices[i].States[0].Id), devices[i].Type),
            DateTime: parseTime(devices[i].States[0].DateTime),
            Name: devices[i].States[0].Name,
        }
        
        // ignore duplicates (linked with or "nested" into devices, not groups?)
        for devices[i].Level > len(typeAtLevel) - 1 {
            typeAtLevel = append(typeAtLevel, devices[i].Type)
        }
        for devices[i].Level < len(typeAtLevel) - 1 {
            typeAtLevel = typeAtLevel[:len(typeAtLevel)-1]
        }

        dev := svc.devices[fixedId]
        if nil == dev || devices[i].Level == 0 || typeAtLevel[devices[i].Level - 1] == 0 {
            svc.idMap[devices[i].Id] = fixedId
            svc.devices[fixedId] = &Device {
                Id: fixedId,
                Order: devices[i].Id, // original id
                Level: devices[i].Level,
                Type: devices[i].Type,
                Name: devices[i].Name,
                Num: [3]int{devices[i].Num1, devices[i].Num2, devices[i].Num3},
                Ip: devices[i].Ip,
                Ip2: devices[i].Ip2,
                Login: devices[i].Login,
                Password: devices[i].Password,
                Option: devices[i].Option,
                Dk: devices[i].Dk,
                States: [2]State{state, {}}}
        }
    }
    svc.Log("Use", len(svc.devices), "devices of", len(devices))
    // TODO: db in not really n/a, need deep check
    svc.SetServiceStatus(api.EC_SERVICE_ONLINE, api.EC_DATABASE_UNAVAILABLE)
    return
}

func (svc *Rif) update(devices []_Device) {
    var fixedId int64
    var ok bool
    //log.Printf("%+v\n", len(devices))
    //svc.Log(devices)
    events := make(api.EventsList, 0, len(devices))
    //var list = make(map[int]Device, len(devices))
    for i, _ := range devices {
        if fixedId, ok = svc.idMap[devices[i].Id]; !ok {
            svc.Log("Unknown device", devices[i])
            continue // unknown device
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
    svc.RLock()
    svc.RUnlock()
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
