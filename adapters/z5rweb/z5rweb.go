package z5rweb

import (
//    "log"
    "fmt"
    "os"
    "time"
    "strings"
    "strconv"
    "context"
    "math/rand"
    "sync/atomic"
    //"encoding/json"
    
    "../../api"
    "../configuration"
//	"strings"
)

const (
    maxLosses = 3 // no reporting for 3 times to become "offline"
    maxPayloadSize = 2 * 1024 - 64 - 128 // 64 = JSON overhead, 128 - reserve
    pinWaitTimeout = 10 // seconds
    
    maxWrongPins = 3 //
    wrongPinsInterval = 60 // seconds
)

func (svc *Z5RWeb) Run() {
    svc.Settings.Status.TCP = "offline"
    
    svc.cfg = svc.Configuration.(configuration.ConfigAPI)
    rand.Seed(time.Now().UnixNano())
    svc.nextMessageId = int64(1e6 + rand.Intn(1e6)) // TODO: use timestamp?
    
    //svc.lastUsers = make(map[int64]int64)
    svc.jobQueue = make(map[int64] []string)
    svc.ignoreEvents = make(map[string] int64)
    svc.lastCards = make(map[string] LastCard)
    svc.wrongPinTimes = make(map[string] []time.Time)
    svc.commandAuthors = make(map[string] int64)
    
    //svc.openDB(svc.GetStorage() + ".db")

    var err error
    svc.httpLog, err = os.OpenFile(svc.GetStorage() + ".json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        svc.Log(err)
    }
    
    svc.loadDevices()
    svc.subscription = svc.cfg.Subscribe()
    go svc.subscriptionLoop()
    
    var ctx context.Context
    ctx, svc.Cancel = context.WithCancel(context.Background())
    go svc.monitorDevices(ctx)
    
    svc.setupApi()
    svc.ReportStartup()
}

/*func (svc *Z5RWeb) waitDevices(ctx context.Context) {
    // TODO:
    // 1. find last active devices MAX(lastSeen) - 1.5 * svc.Settings.KeepAlive
    // 2. wait them all until T + 1.5 * KeepAlive
    if svc.Sleep(ctx, time.Duration(svc.Settings.KeepAlive * 3 / 2) * time.Second) {
        svc.Lock()
        svc.ready = true
        svc.Unlock()
        go svc.SetTCPStatus("online")
    }
}*/

/*func (svc *Z5RWeb) setLastUser(deviceId, userId int64) {
    svc.Lock()
    defer svc.Unlock()
    svc.lastUsers[deviceId] = userId
}

func (svc *Z5RWeb) getLastUser(deviceId int64) int64 {
    svc.RLock()
    defer svc.RUnlock()
    return svc.lastUsers[deviceId]
    //id, ok := svc.lastUsers[deviceId]
    //if ok {
//        delete(svc.lastUsers, deviceId)
  //  }
    return id
}*/

func (svc *Z5RWeb) ZoneCommand(userId, zoneCommand int64, devList []int64) {
    var modes = map[int64]int64{
        api.EC_NORMAL_ACCESS: 0,
        api.EC_POINT_BLOCKED: 1,
        api.EC_FREE_PASS: 2}
    svc.Log("ZONE-COMMAND", devList, zoneCommand)
    for _, devId := range devList {
        if code, ok := modes[zoneCommand]; ok {
            cmd := fmt.Sprintf(`{"deviceId": %d, "command": %d, "argument": %d}`, devId, 37, code)
            svc.Log("CMD:", cmd)
            svc.execCommand(userId, []byte(cmd))
        }
    }
}


func (svc *Z5RWeb) monitorDevices(ctx context.Context) {
    maxDuration := time.Duration(svc.Settings.KeepAlive * maxLosses + 1) * time.Second
    for svc.Sleep(ctx, maxDuration) {
        var events api.EventsList
        var offlineDevices []int64
        svc.Lock()
        for _, dev := range svc.devices {
            if dev.Online && time.Since(dev.LastSeen) > maxDuration {
                dev.Online = false
                offlineDevices = append(offlineDevices, dev.Id)
            }
        }
        svc.Unlock()
        //svc.Log("Offline devs:", offlineDevices)
        for _, devId := range offlineDevices {
            events = append(events, svc.setState(devId, EID_DEVICE_OFFLINE, "", "", ""))
        }
        if nil != events {
            svc.Broadcast("Events", events)
        }
    }
    svc.Log("Devices monitor stopped")
}

func (svc *Z5RWeb) setState(devId, code int64, text, card, dts string) api.Event {
    var userId, zoneId int64
    reader := getReader(code)
    svc.RLock()
    dev := svc.devices[devId]
    svc.RUnlock()
    
    userId = svc.getCommandAuthor(devId, code)
    if 0 == userId && "" != card {
        //TODO: get from cfg by card# ?
        //userId = svc.getLastUser(dev.Id)
        userId = svc.cfg.UserByCard(card)
        if 0 == userId {
            card = svc.getLastCard(devId, reader)
            if "" != card {
                userId = svc.cfg.UserByCard(card)
                //svc.cleanLastCard(dev.Id, reader)
            }
        }
    }

    dt, err := time.ParseInLocation(dateFormat, dts, time.Now().Location())
    if nil != err {
        dt = time.Now()
    }
    
    info, _ := evTypes[code]
    text = info.Text
    if "" == text {
        text = "Неизвестное состояние"
    }
    
    //text += " MODE: " + strconv.Itoa(dev.Mode)
    
    shortCard := strings.TrimLeft(card, "0")
    if "" != shortCard {
        text += " (#" + shortCard + ")"
    }
    
    svc.Lock()
    for i := range dev.Zones {
        if reader == dev.Zones[i][2] {
            zoneId = dev.Zones[i][1]
        }
    }
    if dev.States[0].Class >= 200 {
        dev.States[1] = dev.States[0]
    }
    state := &dev.States[0]
    state.DeviceName = dev.Name    
    //state.FromState = dev.State.Event
    state.Class = info.Class
    state.Event = code
    state.Text = text
    state.UserId = userId
    state.ZoneId = zoneId
    state.Time = dt.Unix()
    svc.Unlock()
    
    // extra for EnterZone
    
    
    /*if code == 16 || code == 17 {
        //state.RelatedDevices = svc.cfg.SameZoneDevices(dev.Id) // for event filtering
        svc.cfg.EnterZone(*state)
    }*/
    
    return dev.States[0]
}

func (svc *Z5RWeb) loadDevices() {
    svc.devices = make(map[int64] *Device)
    devices := svc.cfg.LoadDevices(svc.Settings.Id)

    for i := range devices {
        dev := Device{Device: devices[i]}
        dev.Online = false
        dev.States[0].Class = api.EC_LOST
        dev.States[0].Text = api.DescribeClass(dev.States[0].Class)
        dev.States[0].DeviceId = dev.Id
        dev.States[0].DeviceName = dev.Name
        dev.Zones = svc.cfg.LoadLinks(dev.Id, "device-zone")
        svc.Lock()
        svc.devices[dev.Id] = &dev
        svc.Unlock()
    }
    svc.SetTCPStatus("online")
    svc.Log("::::::::::::::::: DEVICES LOADED !", svc.Settings.Id, len(svc.devices))
}


func (svc *Z5RWeb) makeHandle(dType string, sn int64) string {
    return dType + "-" + strconv.FormatInt(sn, 10)
}

func (svc *Z5RWeb) findDevice(handle string) (*Device, int64) {
    svc.RLock()
    defer svc.RUnlock()

    for _, d := range svc.devices {
        if d.Handle == handle {
            return d, d.Id
        }
    }
    return nil, 0
}

func (svc *Z5RWeb) appendDevice(dev *Device) {
    svc.cfg.SaveDevice(svc.Settings.Id, &dev.Device, nil)
    devId := dev.Id
    dev.States[0].DeviceId = dev.Id
    dev.States[0].DeviceName = dev.Name
    svc.Lock()
    svc.devices[dev.Id] = dev
    svc.Unlock()
    svc.setState(devId, EID_DEVICE_ONLINE, "", "", "")
    devs, _ := svc.listDevices(0, nil)
    svc.Broadcast("ListDevices", devs)
}

// mark as ignored event code for device
func (svc *Z5RWeb) ignoreEvent(card string, code int64) {
    svc.Lock()
    defer svc.Unlock()
    svc.ignoreEvents[card] = code
}

// check event code is ignored for this device
func (svc *Z5RWeb) ignoredEvent(card string, code int64) (ignored bool) {
    svc.Lock()
    defer svc.Unlock()
    
    if svc.ignoreEvents[card] == code {
        // TODO: delete in any case?
        delete(svc.ignoreEvents, card)
        ignored = true
    }
    return
}


func (svc *Z5RWeb) logWrongPin(card string) (alarm bool) {
    svc.Lock()
    defer svc.Unlock()
    wpt := svc.wrongPinTimes[card] // alias
    wpt = append(wpt, time.Now())
    if len(wpt) >= maxWrongPins {
        wpt = wpt[len(wpt)-maxWrongPins:]

        if wpt[len(wpt)-1].Sub(wpt[0]) < time.Second * wrongPinsInterval {
            alarm = true
        }
    }
    svc.wrongPinTimes[card] = wpt
    svc.Log("WPT:", alarm, wpt)
    return
}

func (svc *Z5RWeb) clearLastCard(devId, reader int64) {
    pair := makePair(devId, reader)
    svc.Lock()
    defer svc.Unlock()
    delete(svc.lastCards, pair)
}

func (svc *Z5RWeb) setCommandAuthor(devId, code, userId int64) {
    pair := makePair(devId, code)
    svc.Lock()
    defer svc.Unlock()
    svc.commandAuthors[pair] = userId
}

func (svc *Z5RWeb) getCommandAuthor(devId, code int64) (userId int64) {
    pair := makePair(devId, code)
    svc.Lock()
    defer svc.Unlock()
    userId = svc.commandAuthors[pair]
    delete(svc.commandAuthors, pair)
    return
}


func (svc *Z5RWeb) setLastCard(devId, reader int64, card string) {
    pair := makePair(devId, reader)
    svc.Lock()
    defer svc.Unlock()
    svc.lastCards[pair] = LastCard{card, time.Now()}
}

func (svc *Z5RWeb) getLastCard(devId, reader int64) (last string) {
    pair := makePair(devId, reader)
    svc.Lock()
    defer svc.Unlock()
    //svc.Log("L-CARDS:", svc.lastCards)
    if lc, ok := svc.lastCards[pair]; ok {
        if time.Now().Sub(lc.Time) < time.Second * pinWaitTimeout {
            last = lc.Card
        }
        //delete(svc.lastCards, pair)
    }
    return
}


func makePair(devId, reader int64) string {
    return strconv.FormatInt(devId, 10) + "-" + strconv.FormatInt(reader, 10)
}


func (svc *Z5RWeb) subscriptionLoop() {
    for msg := range svc.subscription {
        svc.Log("Got message:", msg)
    }
}

func (svc *Z5RWeb) Shutdown() {
    svc.Log("Shutting down...")
    svc.Cancel()
    svc.cfg.Unsubscribe(svc.subscription)
    if nil != svc.httpLog {
        svc.httpLog.Close()
    }
    svc.ReportShutdown()
}

func (svc *Z5RWeb) setupApi() {
    svc.Api(map[string] api.Action {
        "ResetAlarm" : svc.resetAlarm,
        "ExecCommand" : svc.execCommand,
        "DeleteDevice" : svc.deleteDevice,
        "ListDevices" : svc.listDevices,
        "UpdateDevice" : svc.updateDevice})
}

func (svc *Z5RWeb) getMessageId() int64 {
    return atomic.AddInt64(&svc.nextMessageId, 1)
}

func (svc *Z5RWeb) newJob(devId, jobId int64, payload string) {
    svc.Lock()
    svc.jobQueue[devId] = append(svc.jobQueue[devId], payload)
    svc.Unlock()
    svc.Log("NEW JOB:", payload)
}

func (svc *Z5RWeb) getJob(devId int64, usedBytes int) (list []string){
    bytesLeft := maxPayloadSize - usedBytes
    svc.Lock()
    defer svc.Unlock()
    for len(svc.jobQueue[devId]) > 0 && len(svc.jobQueue[devId][0]) < bytesLeft {
        list = append(list, svc.jobQueue[devId][0])
        bytesLeft -= len(svc.jobQueue[devId][0]) + 1 // 1 - space for comma
        // TODO: remove only completed job
        svc.jobQueue[devId] = svc.jobQueue[devId][1:]
    }
    return
}


func catch(err error) {
    if nil != err {
        panic(err)
    }
}