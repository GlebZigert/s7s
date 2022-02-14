package ipmon

import (
//    "log"
//    "fmt"
    "encoding/json"
    "../../api"
)

func (svc *IPMon) resetAlarm(cid int64, data []byte) (interface{}, bool) {
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

func (svc *IPMon) listDevices(cid int64, data []byte) (interface{}, bool) {
    var list DevList
    svc.RLock()
    defer svc.RUnlock()

    for _, dev := range svc.devices {
        list = append(list, *dev)
    }
    return list, false // don't broadcast
}

func (svc *IPMon) updateDevice(cid int64, data []byte) (interface{}, bool) {
    device := new(Device)
    json.Unmarshal(data, device) // TODO: handle err
    svc.Lock()
    defer svc.Unlock()
    dev := svc.devices[device.Id]
    if nil == dev {
        dev = new(Device)
        dev.Handle = device.IP
        dev.StateClass = api.EC_NA
    }
    dev.Name = device.Name
    dev.IP = device.IP
    core.SaveDevice(svc.Settings.Id, &dev.Device, &dev.DeviceData)
    if nil == svc.devices[dev.Id] {
        svc.devices[dev.Id] = dev
    }
    // TODO: handle new device with duplicate IP
    return *dev, true
}

func (svc *IPMon) deleteDevice(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    core.DeleteDevice(id)

    svc.Lock()
    delete(svc.devices, id)
    svc.Log("Devices left:", len(svc.devices))
    svc.Unlock()

    return id, true // broadcast
}

/*func (svc *IPMon) updateDevice(cid int64, data []byte) (interface{}, bool) {
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


func (svc *IPMon) deleteDevice(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    svc.dbDeleteDevice(id)
    return id, true // broadcast
}
*/