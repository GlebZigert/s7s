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

func (svc *Z5RWeb) HTTPHandler(w http.ResponseWriter, r *http.Request) {
    svc.RLock()
    ready := svc.devices != nil
    svc.RUnlock()
    
    if !ready {
        code := http.StatusServiceUnavailable
        http.Error(w, http.StatusText(code), code)
        return
    }

    if "POST" != r.Method {
        code := http.StatusMethodNotAllowed
        http.Error(w, http.StatusText(code), code)
        return
    }
    
    parts := strings.Split(r.URL.Path, "/")
    if 3 != len(parts) || "z5rweb" != parts[2] {
        http.NotFound(w, r)
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
    err := json.Unmarshal(body, &parcel)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    dev, devId := svc.findDevice(svc.makeHandle(parcel.Type, parcel.SN))
    
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
                reply = svc.handlePowerOn(parcel.Type, parcel.SN, &msg)
                    
            case "events":
                svc.Log("!! EVENTS !!")
                svc.Log(msg)
                reply = svc.handleEvents(parcel.Type, parcel.SN, &msg)

            case "ping":
                //svc.Log("!! PING !!")
                reply = svc.handlePing(parcel.Type, parcel.SN, &msg)
        
            case "check_access":
                svc.Log("!! CA !!")
                reply = svc.checkAccess(parcel.Type, parcel.SN, &msg)
                
        }
        if nil != reply {
            msg, ok := reply.(string)
            if !ok {
                msgb, err := json.Marshal(reply)
                catch(err)
                msg = string(msgb)
            }
            messages = append(messages, msg)
            usedBytes += len(msg)
        }
    }
    
    svc.logDevice(parcel.Type, parcel.SN)
    
    //message, err := json.Marshal(cmd)
    //catch(err)
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
}

func (svc *Z5RWeb) logDevice(dType string, sn int64) {
    var events api.EventsList
    handle := svc.makeHandle(dType, sn)
    dev, devId := svc.findDevice(handle)
    if nil != dev {
        //svc.Log(dev)
        svc.Lock()
        device := dev.Device
        svc.Unlock()
        svc.cfg.TouchDevice(svc.Settings.Id, &device) // TODO: restore it
        
        if !dev.Online {
            dev.Online = true
            events = api.EventsList{svc.setState(devId, EID_DEVICE_ONLINE, "", "", "")}
        }
        
        if nil != events {
            svc.Broadcast("Events", events)
        }
    } else {
        //svc.Log("Dev found!")
    }
}

//////////////////////////////////////////////////////////
///////////////////// H A N D L E R S ////////////////////
//////////////////////////////////////////////////////////
func (svc *Z5RWeb) checkAccess(dType string, sn int64, msg *Message) interface{} {
    //svc.Log("***** CHECK ACCESS ****", *msg)
    var card string
    var granted int
    var zoneId int64
    var errCode int64
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
            _, errCode = svc.cfg.RequestPassage(zoneId, msg.Card, "")
            if 1 == errCode && "" != card { // maybe msg.Card is a PIN?
                _, errCode = svc.cfg.RequestPassage(zoneId, card, msg.Card)
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
            event := int64(msg.Reader) - 1 + map[int64] int64 {
                68: 68, // blocked
                api.ACS_WRONG_PIN: 62,
                api.ACS_UNKNOWN_CARD: 2,
                api.ACS_ANTIPASSBACK: 54,
                api.ACS_MAX_VISITORS: 66,
                api.ACS_ACCESS_DENIED: 6}[errCode]
            events = append(events, svc.setState(devId, event, "", msg.Card, ""))

            if api.ACS_WRONG_PIN == errCode {
                if svc.logWrongPin(card) {
                    events = append(events, svc.setState(devId, 64 + int64(msg.Reader) - 1, "", card, ""))
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
        Granted: granted}
}

func (svc *Z5RWeb) handlePing(dType string, sn int64, msg *Message) interface{} {
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
        return &SetActiveCmd {Id: svc.getMessageId(), Operation: "set_active", Active: 0, Online: 0}
    }

    return nil
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
func (svc *Z5RWeb) handlePowerOn(dType string, sn int64, msg *Message) interface{} {
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
        svc.appendDevice(dev)
    }
    
    online := 1
    /*if msg.Mode > 0 {
        online = 0
    }*/
    
    ret := &SetActiveCmd {
        Id: svc.getMessageId(),
        Operation: "set_active",
        Active: 1,
        Online: online}
    
    //svc.Log("##### SET ACT ####", ret)
    return ret
}

/*
func (svc *Z5RWeb) handlePing(dType string, sn int64, msg *Message) interface{} {
    now := time.Now()
    fields := dblayer.Fields {
        "active": msg.Active,
        "mode": msg.Mode,
        "last_seen": now}

    if 0 == svc.Table("devices").Seek("type = ? AND serial_number = ?", dType, sn).Update(nil, fields) {
        // TODO: re-activate?
        svc.Warn("Device not found, re-activate:", dType, sn)
        return &SetActiveCmd {Id: svc.nextMessageId, Operation: "set_active", Active: 0, Online: 0}
    }
    return nil
}

func (svc *Z5RWeb) handlePowerOn(dType string, sn int64, msg *Message) interface{} {
    now := time.Now()
    fields := dblayer.Fields {
        "firmware": msg.Firmware,
        "conn_firmware": msg.ConnFirmware,
        "active": msg.Active,
        "mode": msg.Mode,
        "ip": msg.IP,
        "protocol": msg.Protocol,
        "internal_zone": 0,
        "external_zone": 0,
        "last_seen": now}

    if 0 == svc.Table("devices").Seek("type = ? AND serial_number = ?", dType, sn).Update(nil, fields) {
        fields["name"] = dType + " " + strconv.FormatInt(sn, 10)
        fields["type"] = dType
        fields["serial_number"] = sn
        svc.Table("devices").Insert(nil, fields)
        // TODO: load only new device?
        svc.devices = svc.loadDevices()
        // TODO: broadcast updates
    }
    
    svc.nextMessageId += 1
    return &SetActiveCmd {
        Id: svc.nextMessageId,
        Operation: "set_active",
        Active: 1,
        Online: 0}
}*/
func (svc *Z5RWeb) handleEvents(dType string, sn int64, msg *Message) interface{} {
    handle := svc.makeHandle(dType, sn)
    dev, devId := svc.findDevice(handle)
    if nil == dev {
        svc.Warn("Device not found, re-activate:", dType, sn)
        return &SetActiveCmd {Id: svc.getMessageId(), Operation: "set_active", Active: 0, Online: 0}

    }
    events := make(api.EventsList, 0, len(msg.Events))
    for _, event := range msg.Events {
        text := describeEvent(&event)
        if !svc.ignoredEvent(event.Card, event.Event) {
            events = append(events, svc.setState(devId, event.Event, text, event.Card, event.Time))
        }
        
    }
    // broadcast
    if len(events) > 0 { // skip ignored
        svc.Broadcast("Events", events)
    }
    return EventReply {
        Id: svc.getMessageId(),
        Operation: "events",
        EventsSuccess: len(msg.Events)}
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