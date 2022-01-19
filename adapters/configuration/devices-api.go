package configuration

import (
    "time"
    "encoding/json"
    "../../dblayer"
)

func (cfg *Configuration) deviceZone(serviceId, deviceId int64) (zoneId int64, err error) {
    fields := dblayer.Fields {"target_id": &zoneId}

    rows, values, err := db.Table("external_links").
        Seek("link = ? AND scope_id = ? AND source_id = ?", "zone-device", serviceId, deviceId).
        Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()

    if rows.Next() {
        err = rows.Scan(values...)
    }
    if nil == err {
        err = rows.Err()
    }

    return
}


func (cfg *Configuration) getDeviceName(serviceId, deviceId int64) (name string, err error) {
    fields := dblayer.Fields{"name": &name}

    rows, values, err := db.Table("devices").
        Seek("service_id = ? AND id = ?", serviceId, deviceId).
        Get(nil, fields, 1)
    if nil != err {
        return
    }
    defer rows.Close()
    
    if rows.Next() {
        err = rows.Scan(values...)
    }
    if nil == err {
        err = rows.Err()
    }

    //cfg.Log("GDN:", name, serviceId, deviceId)
    return
}


func (cfg *Configuration) getOneDevice(fields dblayer.Fields, serviceId int64, handle string) (id int64, err error) {
    if nil == fields {
        fields = dblayer.Fields{"id": &id}
    }
    rows, values, _ := db.Table("devices").
        Seek("service_id = ? AND handle = ?", serviceId, handle).
        Get(nil, fields, 1)
    
    if rows.Next() {
        err = rows.Scan(values...)
    }
    rows.Close()
    return
}

func (cfg *Configuration) GlobalDeviceId(serviceId int64, handle, name string) (id int64) {
    id, err := cfg.getOneDevice(nil, serviceId, handle)
    catch(err)

    fields := dblayer.Fields {
        "name":         name,
        "last_seen":    time.Now()}

    if 0 == id {
        fields["service_id"] = serviceId
        fields["handle"] = handle
        id, _ = db.Table("devices").Insert(nil, fields)
    } else {
        db.Table("devices").Seek(id).Update(nil, fields)
    }
    return
}

func (cfg *Configuration) LoadDevices(serviceId int64) (list []Device, err error) {
    dev := new(Device) 
    fields := dblayer.Fields {
        "id":           &dev.Id,
        //"service_id":   &dev.ServiceId,
        "handle":       &dev.Handle,
        "name":         &dev.Name,
        "last_seen":    &dev.LastSeen,
        "data":         &dev.Data}

    rows, values, err := db.Table("devices").Seek("handle IS NOT NULL AND service_id = ?", serviceId).Get(nil, fields)
    
    if nil != err {
        return
    }
    
    defer rows.Close()
    for rows.Next() {
        err = rows.Scan(values...)
        if nil != err {
            break
        }
        list = append(list, *dev)        
    }
    
    return
}

func (cfg *Configuration) SaveDevice(serviceId int64, dev *Device, data interface{}) {
    fields := dblayer.Fields {
        "id":           &dev.Id,
        "service_id":   serviceId,
        "handle":       dev.Handle,
        "name":         dev.Name,
        "last_seen":    time.Now()}
    
    if nil != data {
        fields["data"], _ = json.Marshal(data)
    }

    db.Table("devices").Save(nil, fields)
}

func (cfg *Configuration) DeleteDevice(id int64) (err error) {
    fields := dblayer.Fields {
        "handle":       nil,
        "last_seen":    time.Now()} // deletion time

    db.Table("devices").Seek(id).Update(nil, fields)
    db.Table("external_links").Delete(nil, "link = ? AND target_id = ?", "user-device", id)
    return
}

func (cfg *Configuration) TouchDevice(serviceId int64, dev *Device) {
    dev.LastSeen = time.Now()
    fields := dblayer.Fields {"last_seen": dev.LastSeen}
    db.Table("devices").Seek(dev.Id).Update(nil, fields)
}
