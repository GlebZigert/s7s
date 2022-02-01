// case open: [{"type":"Z5RWEB","sn":49971,"messages":[{"id":2049197811,"operation":"events","events":[{"flag": 1291,"event": 39,"time": "2021-04-27 15:44:51","card": "000000000000"}]}]}]

// "card": "7D4E87747474"
// "card": "0000007D4E87"
package z5rweb

import (
//    "io"
    "fmt"
    "time"   
    "net/http"
    "strings"
    "encoding/json"
//    "strconv"
//    "bytes"
    "io/ioutil"
//    "../../dblayer"
    "../../api"
)

const (
    maxUploadSize = 2048
    maxDownloadSize = 16384
    dateFormat = "2006-01-02 15:04:05"
)


var addCardStep = 0

func (svc *Z5RWeb) HTTPHandler(w http.ResponseWriter, r *http.Request) (err error) {
    var httpErrCode int
    defer func () {
        // TODO: create err in case of err == nil && httpErrCode != 0
        svc.complaints <- err
        if nil != err && 0 == httpErrCode {
            httpErrCode = http.StatusInternalServerError
        }
        if 0 != httpErrCode {
            http.Error(w, http.StatusText(httpErrCode), httpErrCode)
        }
    }()
    svc.RLock()
    ready := svc.devices != nil
    svc.RUnlock()
    
    if !ready {
        httpErrCode = http.StatusServiceUnavailable
        return
    }

    if "POST" != r.Method {
        httpErrCode = http.StatusMethodNotAllowed
        return
    }
    
    parts := strings.Split(r.URL.Path, "/")
    if 3 != len(parts) || "z5rweb" != parts[2] {
        httpErrCode = http.StatusNotFound //http.NotFound(w, r)
        return
    }

    /*cmd := CommandsList {
        Date: time.Now().Format(dateFormat),
        Interval: svc.Settings.KeepAlive,
        Messages: []interface{}{}}*/

    var parcel Parcel
    body, _ := ioutil.ReadAll(r.Body)
    //svc.Log(string(body))
    //err := json.NewDecoder(r.Body).Decode(&parcel)
    svc.httpLog.Write([]byte("\n\n========== <<<R ==========\n\n"))
    svc.httpLog.Write([]byte(body))
    err = json.Unmarshal(body, &parcel)
    if err != nil {
        httpErrCode = http.StatusBadRequest
        return
    }
    
    err = svc.logDevice(parcel.Type, parcel.SN)
    if nil != err {
        return
    }

    //////////////////////// S C A N   M E S S A G E S ////////////////
    var messages []string
    var usedBytes int
    for _, msg := range parcel.Messages {
        var reply interface{}
        //svc.Log(msg.Operation)
        switch (msg.Operation) {
            case "":
                if msg.Success == 1 {
                    svc.Log("!! SUCC:", msg.Id)
                }
            case "power_on":
                svc.Log("!! P_ON !!")
                reply, err = svc.handlePowerOn(parcel.Type, parcel.SN, &msg)
                    
            case "events":
                svc.Log("!! EVENTS !!")
                svc.Log(msg)
                reply, err = svc.handleEvents(parcel.Type, parcel.SN, &msg)

            case "ping":
                //svc.Log("!! PING !!")
                reply, err = svc.handlePing(parcel.Type, parcel.SN, &msg)
        
            case "check_access":
                svc.Log("!! CA !!")
                reply, err = svc.checkAccess(parcel.Type, parcel.SN, &msg)
                
        }
        if nil != err {
            return
        }
        
        if nil != reply {
            msg, ok := reply.(string)
            if !ok { // not string
                msgb, _ := json.Marshal(reply)
                msg = string(msgb)
            }
            messages = append(messages, msg)
            usedBytes += len(msg)
        }
    }
    
    dev, devId := svc.findDevice(svc.makeHandle(parcel.Type, parcel.SN))        
    if nil != dev {
        messages = append(messages, svc.getJob(devId, usedBytes)...)
    }
    message := fmt.Sprintf(`{"date": "%s","interval": %d,"messages": [%s]}`,
        time.Now().Format(dateFormat),
        svc.Settings.KeepAlive,
        strings.Join(messages, ","))

    svc.httpLog.Write([]byte("\n\n========== S>>> ==========\n\n"))
    svc.httpLog.Write([]byte(message))

    w.Write([]byte(message))
    return
}

func (svc *Z5RWeb) logDevice(dType string, sn int64) (err error) {
    defer func () {svc.complaints <- err}()
    var events api.EventsList
    handle := svc.makeHandle(dType, sn)
    dev, devId := svc.findDevice(handle)
    if nil != dev {
        //svc.Log(dev)
        svc.Lock()
        device := dev.Device
        svc.Unlock()
        err = core.TouchDevice(svc.Settings.Id, &device) // TODO: restore it
        if !dev.Online {
            dev.Online = true
            // INFO: never return error because no user affected (card = "")
            ev, _ := svc.setState(devId, EID_DEVICE_ONLINE, "", "", "")
            events = api.EventsList{ev}
        }
        
        if nil != events {
            svc.Broadcast("Events", events)
        }
    } else {
        //svc.Log("Dev found!")
    }
    return
}

//////////////////////////////////////////////////////////
///////////////////// H A N D L E R S ////////////////////
//////////////////////////////////////////////////////////
func (svc *Z5RWeb) checkAccess(dType string, sn int64, msg *Message) (res interface{}, err error) {
    //svc.Log("***** CHECK ACCESS ****", *msg)
    var card string
    var granted int
    var zoneId int64
    var errCode int64
    var ev api.Event
    var events api.EventsList
    handle := svc.makeHandle(dType, sn)
    dev, devId := svc.findDevice(handle)
    if nil != dev {
        svc.RLock()
        for i := range dev.Zones {
            if msg.Reader == dev.Zones[i][2] {
                zoneId = dev.Zones[i][1]
            }
        }
        blocked := 1 == dev.Mode
        svc.RUnlock()
        if blocked {
            errCode = 68 // blocked
        } else {
            card = svc.getLastCard(devId, msg.Reader)
            // try msg.Card as a plain card
            _, errCode = core.RequestPassage(zoneId, msg.Card, "")
            if 1 == errCode && "" != card { // maybe msg.Card is a PIN?
                _, errCode = core.RequestPassage(zoneId, card, msg.Card)
            }
        }
        if 0 == errCode {
            granted = 1
        } else if api.ACS_PIN_REQUIRED == errCode {
            // wait for pin
            svc.setLastCard(devId, msg.Reader, msg.Card)
            svc.ignoreEvent(msg.Card, 2 + int64(msg.Reader) - 1)
            svc.Log("Wait for pin")
        } else {
            // analyze passage attempt results
            event := mapEventCode(msg.Reader, errCode)
            ev, err = svc.setState(devId, event, "", msg.Card, "")
            if nil != err {
                return
            }
            events = append(events, ev)

            if api.ACS_WRONG_PIN == errCode {
                if svc.logWrongPin(card) {
                    ev, err = svc.setState(devId, 64 + int64(msg.Reader) - 1, "", card, "")
                    if nil != err {
                        return
                    }
                    events = append(events, ev)
                }
                svc.clearLastCard(devId, msg.Reader)
            }
            svc.Broadcast("Events", events)
            svc.ignoreEvent(msg.Card, 2 + int64(msg.Reader) - 1)
        }
    }
    //svc.setLastUser(dev.Id, userId)
    
    return &CheckAccessReply {
        Id: msg.Id,
        Operation: "check_access",
        Granted: granted}, nil
}

func mapEventCode(reader, errCode int64) int64 {
    return reader - 1 + map[int64] int64 {
        68: 68, // blocked
        api.ACS_WRONG_PIN: 62,
        api.ACS_UNKNOWN_CARD: 2,
        api.ACS_ANTIPASSBACK: 54,
        api.ACS_MAX_VISITORS: 66,
        api.ACS_ACCESS_DENIED: 6}[errCode]
}

func (svc *Z5RWeb) handlePing(dType string, sn int64, msg *Message) (res interface{}, err error) {
    var modes = []int64{api.EC_NORMAL_ACCESS, api.EC_POINT_BLOCKED, api.EC_FREE_PASS}
    handle := svc.makeHandle(dType, sn)
    dev, devId := svc.findDevice(handle)
    //svc.Log("CURRENT MODE:", msg.Mode)
    if nil != dev && msg.Active == 1{
        dev.Active = msg.Active
        if dev.Mode != msg.Mode && int(msg.Mode) < len(modes) {
            svc.RLock()
            name := dev.Name
            svc.RUnlock()
            svc.Broadcast("Events", api.EventsList{api.Event{
                Class: modes[msg.Mode],
                ServiceId: svc.Settings.Id,
                ServiceName: svc.Settings.Title,
                DeviceId: devId,
                DeviceName: name}})
        }
        dev.Mode = msg.Mode
        
        dev.LastSeen = time.Now()
    } else {// re-activate
        svc.Warn("Device not found, re-activate:", dType, sn)
        res = &SetActiveCmd {Id: svc.getMessageId(), Operation: "set_active", Active: 0, Online: 0}
    }

    return
    /*
    if addCardStep == 0 {
        addCardStep += 1
        svc.nextMessageId += 1
        return &SetTimezoneCmd {
            Id: svc.nextMessageId,
            Operation: "set_timezone",
            Zone: 1,
            Begin: "12:00",
            End: "18:00",
            Days: "11111110"}
    } else if addCardStep == 1 {
        addCardStep += 1
        svc.nextMessageId += 1
        return &ClearCardsCmd {
            Id: svc.nextMessageId,
            Operation: "clear_cards"}
    } else if && addCardStep == 2 {
        addCardStep += 1
        svc.nextMessageId += 1
        return &AddCardsCmd {
            Id: svc.nextMessageId,
            Operation: "add_cards",
            // add test card
            // 9609036 => 000000929F4C
            // 9894522 => 00000096FA7A
            Cards: []OneCard{
                // "00045696FA7A"
                {Card: "000321929F4C", Flags: 4, TZ: 255},
                {Card: "00012396FA7A", Flags: 4, TZ: 255},
                {Card: "000000000911", Flags: 32, TZ: 255}}}
    } else {
        return nil
    }*/
}

//[{"type":"Z5RWEB","sn":49971,"messages":[{"id":1101513929,"operation":"power_on","fw":"3.28","conn_fw":"1.0.128","active":0,"mode":0,"controller_ip":"192.168.0.79","reader_protocol":"wiegand"}]}]
func (svc *Z5RWeb) handlePowerOn(dType string, sn int64, msg *Message) (ret interface{}, err error) {
    handle := svc.makeHandle(dType, sn)
    dev, devId := svc.findDevice(handle)
    if nil == dev {
        dev = new(Device)
        dev.Name = handle
    }
    
    svc.Lock()
    //dev.Data = dev.DeviceData
    dev.Hardware = dType
    dev.SerialNumber = sn
    dev.Firmware = msg.Firmware
    dev.ConnFirmware = msg.ConnFirmware
    dev.Active = msg.Active
    dev.Mode = msg.Mode
    dev.IP = msg.IP
    dev.Protocol = msg.Protocol
    dev.Active = msg.Active
    dev.Mode = msg.Mode
    dev.LastSeen = time.Now()
    dev.Handle = handle
    //dev.Online = true
    svc.Unlock()
    
    if 0 == devId { // new device, save it
        err = svc.appendDevice(dev)
    }
    if nil != err {
        return
    }
    
    online := 1
    /*if msg.Mode > 0 {
        online = 0
    }*/
    
    ret = &SetActiveCmd {
        Id: svc.getMessageId(),
        Operation: "set_active",
        Active: 1,
        Online: online}
    
    //svc.Log("##### SET ACT ####", ret)
    return
}


func (svc *Z5RWeb) handleEvents(dType string, sn int64, msg *Message) (res interface{}, err error) {
    handle := svc.makeHandle(dType, sn)
    dev, devId := svc.findDevice(handle)
    if nil == dev {
        svc.Warn("Device not found, re-activate:", dType, sn)
        res = &SetActiveCmd {Id: svc.getMessageId(), Operation: "set_active", Active: 0, Online: 0}
        return
    }
    var ev api.Event
    events := make(api.EventsList, 0, len(msg.Events))
    for _, event := range msg.Events {
        text := describeEvent(&event)
        if !svc.ignoredEvent(event.Card, event.Event) {
            ev, err = svc.setState(devId, event.Event, text, event.Card, event.Time)
            if nil != err {
                break
            }
            events = append(events, ev)
        }
        
    }
    if nil != err {
        return
    }
    // broadcast
    if len(events) > 0 { // skip ignored
        svc.Broadcast("Events", events)
    }
    return EventReply {
        Id: svc.getMessageId(),
        Operation: "events",
        EventsSuccess: len(msg.Events)}, err
}

////////////////////////////////////////////////////////
/////////////////////// E X T R A //////////////////////
////////////////////////////////////////////////////////
func (svc *Z5RWeb) getDeviceId(dType string, sn int64) int64 {
    id := int64(-1)
    for _, d := range svc.devices {
        if d.Hardware == dType && d.SerialNumber == sn {
            id = d.Id
            break
        }
    }
    return id
}