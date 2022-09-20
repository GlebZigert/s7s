package parus

import (
//    "log"
//    "fmt"
    "encoding/json"
    "s7server/api"
)

func (svc *Parus) resetAlarm(cid int64, data []byte) (interface{}, bool) {
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

func (svc *Parus) listDevices(cid int64, data []byte) (interface{}, bool) {
    var list DevList
    svc.RLock()
    defer svc.RUnlock()

    for _, dev := range svc.devices {
        list = append(list, *dev)
    }
    return list, false // don't broadcast
}

func (svc *Parus) updateDevice(cid int64, data []byte) (interface{}, bool) {
    var msg string
    device := new(Device)
    err := json.Unmarshal(data, device) // TODO: handle err
    catch(err)
    //svc.Log("DEV:", device)

    svc.RLock() //>>>>>>>>>>>>>>>>>>>>>>
    for _, dev := range svc.devices {
        //svc.Log(dev.Id, dev.Handle, " = ", device.Id, device.IP)
        if dev.IP == device.IP && dev.Id != device.Id {
            msg = "Это устройство уже присутствует в системе как " + dev.Name
            break
        }
    }
    dev := svc.devices[device.Id]
    svc.RUnlock() //<<<<<<<<<<<<<<<<<<<

    if "" != msg {
        return apiErr(msg)
    }
    if nil == dev && device.Id != 0 {
        return apiErr("Устройство удалено или отсутствует в системе.")
    }

    // encrypt password
    device.password = device.NewPassword
    device.NewPassword, err = core.Encrypt(device.password)
    catch(err)

    err = core.SaveDevice(svc.Settings.Id, &device.Device, &device.DeviceData)
    catch(err)
    
    svc.Lock()
    defer svc.Unlock()
    
    if nil == dev {
        // new device: fill initial data
        dev = device
        dev.Handle = dev.IP
        dev.StateClass = api.EC_NA
        dev.NewPassword = ""
        svc.devices[device.Id] = dev
    } else {
        // apply submitted values
        // if device was deleted just before it, nothing harmful happens
        dev.Name = device.Name
        dev.IP = device.IP
        dev.Login = device.Login
        dev.password = device.password
    }

    return *dev, true
}

func (svc *Parus) deleteDevice(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    err := core.DeleteDevice(id)
    catch(err)
    svc.Lock()
    delete(svc.devices, id)
    svc.Log("Devices left:", len(svc.devices))
    svc.Unlock()

    return id, true // broadcast
}

/*func (svc *Parus) updateDevice(cid int64, data []byte) (interface{}, bool) {
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


func (svc *Parus) deleteDevice(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    svc.dbDeleteDevice(id)
    return id, true // broadcast
}
*/

func apiErr(msg string) (*api.ErrorData, bool) {
    return &api.ErrorData{0, msg}, false
}

func catch(err error) {
    if nil != err {
        panic(err)
    }
}
