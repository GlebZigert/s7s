package z5rweb

import (
//    "log"
    "fmt"
    "os"
    "time"
    "strconv"
    "context"
    "math/rand"
    "sync/atomic"
    "encoding/json"
    
    "s7server/api"
    "s7server/adapters/configuration"
//	"strings"
)

const (
    LogExchange = false
    maxLosses = 3 // no reporting for 3 times to become "offline"
    maxPayloadSize = 2 * 1024 - 64 - 128 // 64 = JSON overhead, 128 - reserve
    pinWaitTimeout = 10 // seconds
    
    maxWrongPins = 3 //
    wrongPinsInterval = 60 // seconds
    journalInterval = 7 * time.Second
)

var core configuration.ConfigAPI

func (svc *Z5RWeb) Run(_ configuration.ConfigAPI) (err error) {
    configuration.ExportCore(&core)
    var ctx context.Context
    ctx, svc.Cancel = context.WithCancel(context.Background())
    defer svc.Cancel()
    svc.Stopped = make(chan struct{})
    defer close(svc.Stopped)
    
    svc.complaints = make(chan error, 100)
    go svc.ErrChecker(ctx, svc.complaints, api.EC_SERVICE_READY, api.EC_SERVICE_FAILURE)
    
    svc.journalLoaded = make(chan int64)
    go svc.journalMon(ctx)

    
    rand.Seed(time.Now().UnixNano())
    svc.nextMessageId = int64(1e6 + rand.Intn(1e6)) // TODO: use timestamp?
    
    //svc.lastUsers = make(map[int64]int64)
    svc.jobQueue = make(map[int64] []string)
    svc.ignoreEvents = make(map[string] int64)
    svc.lastCards = make(map[string] LastCard)
    svc.wrongPinTimes = make(map[string] []time.Time)
    svc.commandAuthors = make(map[string] int64)
    
    //svc.openDB(svc.GetStorage() + ".db")

    if LogExchange {
        svc.httpLog, err = os.OpenFile(svc.GetStorage() + ".json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if nil != err {
            return
        }
        defer svc.httpLog.Close()
    }
    
    err = svc.loadDevices()
    if err != nil {
        return
    }
    
    go svc.monitorDevices(ctx)
    
    svc.setupApi()
    svc.SetServiceStatus(api.EC_SERVICE_READY)

    <-ctx.Done()
    ////////////////////////////////////////////////////////////
    
    svc.Log("Shutting down...")
    svc.SetServiceStatus(api.EC_SERVICE_SHUTDOWN)
    return
}

func (svc *Z5RWeb) Shutdown() {
    svc.RLock()
    ret := nil == svc.Cancel || nil == svc.Stopped
    svc.RUnlock()
    if ret {
        svc.Err("Shutdown is called before Run!")
        return
    }

    svc.Cancel()
    <-svc.Stopped
}

// Return all devices IDs for user filtering
func (svc *Z5RWeb) GetList() []int64 {
    svc.RLock()
    defer svc.RUnlock()

    list := make([]int64, 0, len(svc.devices))
    
    for id := range svc.devices {
        list = append(list, id)
    }

    return list
}

func (svc *Z5RWeb) ZoneCommand(userId, zoneCommand int64, devList []int64) {
    var modes = map[int64]int64{
        api.EC_NORMAL_ACCESS: 0,
        api.EC_POINT_BLOCKED: 1,
        api.EC_FREE_PASS: 2}
    //svc.Log("ZONE-COMMAND", devList, zoneCommand)
    for _, devId := range devList {
        if code, ok := modes[zoneCommand]; ok {
            cmd := fmt.Sprintf(`{"deviceId": %d, "command": %d, "argument": %d}`, devId, 370 + code, 0)
            //svc.Log("CMD:", cmd)
            svc.execCommand(userId, []byte(cmd))
        }
    }
}

func (svc *Z5RWeb) journalNotice(list map[int64]struct{}) {
    var events api.EventsList
    var devices []*Device

    svc.RLock()
    for id := range list {
        dev := svc.devices[id]
        if nil != dev {
            devices = append(devices, dev)
        }
    }
    svc.RUnlock()

    for _, dev := range devices {
        // INFO: it will never return an error because no user affected (card = "")
        ev, _ := svc.setState(dev, &Event{Event: EID_EVENTS_LOADED})
        events = append(events, ev)
    }
    if len(events) > 0 {
        svc.Broadcast("Events", events)
    }
}

func (svc *Z5RWeb) journalMon(ctx context.Context) {
    defer svc.Log("Journal update monitor stopped")
    list := make(map[int64]struct{})
    timer := time.NewTimer(0)
    for nil == ctx.Err() {
        select {
            case <-ctx.Done():
                return

            case id := <-svc.journalLoaded:
                if 0 == len(list) {
                    timer.Reset(journalInterval)
                }
                list[id] = struct{}{}
            
            case <-timer.C:
                svc.journalNotice(list)
                list = make(map[int64]struct{})
        }
    }
}


func (svc *Z5RWeb) monitorDevices(ctx context.Context) {
    defer svc.Log("Devices monitor stopped")
    maxDuration := time.Duration(svc.Settings.KeepAlive * maxLosses + 1) * time.Second
    for svc.Sleep(ctx, maxDuration) {
        var events api.EventsList
        var offlineDevices []*Device
        svc.Lock()
        for _, dev := range svc.devices {
            if dev.Online && time.Since(dev.LastSeen) > maxDuration {
                dev.Online = false
                offlineDevices = append(offlineDevices, dev)
            }
        }
        svc.Unlock()
        //svc.Log("Offline devs:", len(offlineDevices))
        for _, dev := range offlineDevices {
            // INFO: never return error because no user affected (card = "")
            ev, _ := svc.setState(dev, &Event{Event: EID_DEVICE_OFFLINE})
            events = append(events, ev)
        }
        if nil != events {
            svc.Broadcast("Events", events)
        }
    }
}

func (svc *Z5RWeb) setState(dev *Device, event *Event) (eee api.Event, err error) {
    svc.RLock()
    devId := dev.Id
    svc.RUnlock()
    
    reader := getReader(event.Event)
    card := event.Card
    userId := svc.getCommandAuthor(devId, event.Event)
    
    if 0 == userId && "" != event.Card {
        //TODO: get from cfg by card# ?
        //userId = svc.getLastUser(dev.Id)
        userId, err = core.UserByCard(card)
        if nil == err && 0 == userId {
            if crd := svc.getLastCard(devId, reader); "" != crd {
                card = crd
                userId, err = core.UserByCard(card)
                //svc.cleanLastCard(dev.Id, reader)
            }
        }
    }
    if nil != err {return}

    event.Card = card // WARN: input data change!
    ev := svc.adoptEvent(dev, event)
    svc.Lock()
    defer svc.Unlock()

    if dev.States[0].Class >= 200 {
        dev.States[1] = dev.States[0]
    }
    ev.UserId = userId
    dev.States[0] = *ev
    
    return dev.States[0], nil
}

// !!! check devId existance before calling!
func (svc *Z5RWeb) adoptEvent(dev *Device, event *Event) (eee *api.Event) {
    var zoneId int64
    svc.RLock()
    defer svc.RUnlock()

    dt := parseDateTime(event.Time)
    text, class := describeEvent(event)
    reader := getReader(event.Event)
    for i := range dev.Zones {
        if reader == dev.Zones[i][2] {
            zoneId = dev.Zones[i][1]
        }
    }
    return &api.Event{
        ServiceId: svc.Settings.Id,
        DeviceId: dev.Id,
        DeviceName: dev.Name,
        Class: class,
        Event: event.Event,
        Text: text,
        ZoneId: zoneId,
        Time: dt.Unix(),
    }
}

func (svc *Z5RWeb) loadDevices() (err error) {
    var zones []configuration.ExtLink
    svc.devices = make(map[int64] *Device)
    devices, err := core.LoadDevices(svc.Settings.Id)
    if nil != err {return}

    for i := range devices {
        dev := Device{Device: devices[i]}
        if "" != dev.Data {
            json.Unmarshal([]byte(dev.Data), &dev.DeviceData)
        }
        dev.Online = false
        dev.States[0].Class = api.EC_LOST
        dev.States[0].Text = api.DescribeClass(dev.States[0].Class)
        dev.States[0].DeviceId = dev.Id
        dev.States[0].DeviceName = dev.Name
        zones, err = core.LoadLinks(dev.Id, "device-zone")
        if nil != err {break}
        if 2 == len(zones) {
            dev.Zones = [2]configuration.ExtLink{zones[0], zones[1]}
        }
        svc.Lock()
        svc.devices[dev.Id] = &dev
        svc.Unlock()
    }
    //svc.SetTCPStatus("online")
    //svc.Log("::::::::::::::::: DEVICES LOADED !", svc.Settings.Id, len(svc.devices))
    return
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

func (svc *Z5RWeb) appendDevice(dev *Device) (err error) {
    err = core.SaveDevice(svc.Settings.Id, &dev.Device, nil)
    dev.States[0].DeviceId = dev.Id
    dev.States[0].DeviceName = dev.Name
    svc.Lock()
    svc.devices[dev.Id] = dev
    svc.Unlock()
    // INFO: it will never return an error because no user affected (card = "")
    svc.setState(dev, &Event{Event: EID_DEVICE_ONLINE})
    devs, _ := svc.listDevices(0, nil)
    svc.Broadcast("ListDevices", devs)
    return
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
    //svc.Log("WPT:", alarm, wpt)
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
    //svc.Log("NEW JOB:", payload)
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

func parseDateTime(dts string) (dt time.Time) {
    dt, err := time.ParseInLocation(dateFormat, dts, time.Now().Location())
    if nil != err {
        dt = time.Now()
    }
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
