package rif

import (
    "fmt"
    //"time"
    "encoding/json"
    "s7server/api"
)

var cmdList = map[int64]string {
    100: "Запрос на выключение",
    101: "Запрос на включение",
    110: "Запрос на закрытие",
    111: "Запрос на открытие"}

func (svc *Rif) resetAlarm(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    //core.DeleteDevice(id)

    svc.RLock()
    defer svc.RUnlock()
    dev := svc.devices[id]
    if nil != dev {
        events := api.EventsList{{
            ServiceId: svc.Settings.Id,
            ServiceName: svc.Settings.Title,
            DeviceId: dev.Id,
            DeviceName: dev.Name,
            UserId: cid,
            Class: api.EC_INFO_ALARM_RESET}}
        svc.Broadcast("Events", events)
        
        return true, true // broadcast
    }
    return false, false // don't broadcast
}

func (svc *Rif) commandXML(devId, command int64) (xml string) {
    svc.RLock()
    defer svc.RUnlock()
    d := svc.devices[devId]
    if nil == d {
        return
    }

    xml = fmt.Sprintf(`<RIFPlusPacket type="Commands">
        <Commands>
            <Command id="%d"/>
            <device id="%d" level="%d" type="%d" num1="%d" num2="%d" num3="%d" />
        </Commands></RIFPlusPacket>`,
        command, d.Order, d.Level, d.Type, d.Num[0], d.Num[1], d.Num[2])    
    return
}

func (svc *Rif) execCommand(cid int64, data []byte) (interface{}, bool) {
    var xml string
    command := new(api.Command)
    json.Unmarshal(data, command) // TODO: handle err
    
    
    if 0 == command.DeviceId {
        switch command.Command {
            case 133: // DK
                xml = `<RIFPlusPacket type="Commands"><Commands><Command id="133"/><device id="0" level="0" type="1" num1="0" num2="0" num3="0" /></Commands></RIFPlusPacket>`
            case 903:
                xml = `<RIFPlusPacket type="AlarmsReset"></RIFPlusPacket>`
        }
        
    } else  { // ignore unknown (deleted) devices
        xml = svc.commandXML(command.DeviceId, command.Command)
    }
    if "" != xml {
        //svc.Log("Sending:", xml)
        svc.queueWait(command.DeviceId, command.Command, cid)
        //svc.Log("STORE:", svc.userRequest)
        svc.SendCommand(xml)
    }

    return "", false
}

func (svc *Rif) listDevices(cid int64, data []byte) (interface{}, bool) {
    var list DevList
    svc.RLock()
    defer svc.RUnlock()
    
    for _, dev := range svc.devices {
        list = append(list, *dev)
    }

    return list, false
}

///////////////////////////////////////////////////////////////////////////////////////////


func describeCommand(code int64) (text string) {
    text = cmdList[code]
    if "" == text {
        fmt.Sprintf("Неизвестная команда: %d", code)
    }
    return 
}

func catch(err error) {
    if nil != err {
        panic(err)
    }
}
