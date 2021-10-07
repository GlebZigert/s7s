package z5rweb

import (
//    "log"
    "fmt"
    "encoding/json"
    
    "../../api"
)

/****************** Z5RWeb+  Actions ******************/

/*func (svc *Z5RWeb) setMode(devId, mode int64) {
    var modes = []int64{api.EC_NORMAL_ACCESS, api.EC_POINT_BLOCKED, api.EC_FREE_PASS}
    svc.RLock()
    defer svc.RUnlock()
    dev := svc.devices[devId]
    //svc.Log("SET MODE:", mode, dev.Mode, dev)
    if nil == dev || int64(dev.Mode) == mode || mode > 2 {
        return
    }
    
    dev.Mode = int(mode)
    
    
    svc.Broadcast("Events", api.EventsList{api.Event{
        Class: modes[mode],
        ServiceId: svc.Settings.Id,
        ServiceName: svc.Settings.Title,
        DeviceId: dev.Id,
        DeviceName: dev.Name}})
}*/

func (svc *Z5RWeb) resetAlarm(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    //svc.cfg.DeleteDevice(id)

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


func (svc *Z5RWeb) execCommand(cid int64, data []byte) (interface{}, bool) {
    command := new(api.Command)
    json.Unmarshal(data, command) // TODO: handle err
    svc.Log("Command:", command)
    devId := command.DeviceId
    svc.RLock()
    dev := svc.devices[devId]
    if nil == dev {
        svc.RUnlock()
        return "Устройство не найдено", false
    }

    payload := make(map[int64]string)
    switch command.Command {
        case 8, 9: // Открыто оператором по сети (вход/выход)
            id := svc.getMessageId()
            tpl := `{"id": %d, "operation": "open_door", "direction": %d}`
            payload[id] = fmt.Sprintf(tpl, id, command.Command - 8) // 0 = in, 1 = out
        case 37: // Переключение режимов работы
            //svc.setMode(devId, command.Argument)
            //TODO: this works only in offline mode?
            if dev.Mode != command.Argument { // skip current mode
                id := svc.getMessageId()
                tpl := `{"id": %d, "operation": "set_mode", "mode": %d}`
                payload[id] = fmt.Sprintf(tpl, id, command.Argument)
            }
            /*id = svc.getMessageId()
            tpl = `{"id": %d, "operation":"set_active", "active":1, "online": %d}`
            online := 1
            if command.Argument > 0 {
                online = 0
            }
            payload[id] = fmt.Sprintf(tpl, svc.getMessageId(), online)*/
    }
    svc.RUnlock()
    //svc.Log("######################################### PAYLOAD:", payload)
    for id := range payload {
        svc.newJob(devId, id, payload[id])
    }
    if len(payload) > 0 {
        svc.setCommandAuthor(devId, command.Command, cid)
    }
    return "", false
}

func (svc *Z5RWeb) listDevices(cid int64, data []byte) (interface{}, bool) {
    var list DevList
    svc.RLock()
    defer svc.RUnlock()

    for _, dev := range svc.devices {
        list = append(list, *dev)
    }
    return list, false // don't broadcast
}

func (svc *Z5RWeb) updateDevice(cid int64, data []byte) (interface{}, bool) {
    device := new(Device)
    json.Unmarshal(data, device) // TODO: handle err
    //svc.dbUpdateDevice(device)
    //svc.Log(device)
    svc.Lock()
    defer svc.Unlock()
    if dev, ok := svc.devices[device.Id]; ok {
        dev.Name = device.Name
        //dev.ExternalZone = device.ExternalZone
        //dev.InternalZone = device.InternalZone
        dev.Zones = device.Zones
        svc.cfg.SaveDevice(svc.Settings.Id, &dev.Device, nil)
        svc.cfg.SaveLinks(dev.Id, "device-zone", dev.Zones)
        return *dev, true // broadcast
    }
    return "Устройство не найдено", false // don't broadcast error
}

func (svc *Z5RWeb) deleteDevice(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    svc.cfg.DeleteDevice(id)

    svc.Lock()
    delete(svc.devices, id)
    svc.Log("Devices left:", len(svc.devices))
    svc.Unlock()
    
    return id, true // broadcast
}

/*func (svc *Z5RWeb) updateDevice(cid int64, data []byte) (interface{}, bool) {
    device := new(Device)
    json.Unmarshal(data, device) // TODO: handle err
    svc.dbUpdateDevice(device)
    if dev, ok := svc.devices[device.Id]; ok {
        dev.Name = device.Name
        dev.ExternalZone = device.ExternalZone
        dev.InternalZone = device.InternalZone
        return dev, true // broadcast
    }
    return "Устройство не найдено", false // don't broadcast error
}


func (svc *Z5RWeb) deleteDevice(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    svc.dbDeleteDevice(id)
    return id, true // broadcast
}
*/