package configuration

import (
    "time"
    "database/sql"
    "encoding/json"
    "s7server/dblayer"
)

func (cfg *Configuration) deviceZone(serviceId, deviceId int64) (zoneId int64, err error) {
    fields := dblayer.Fields {"target_id": &zoneId}

    err = db.Table("external_links").
        Seek("link = ? AND scope_id = ? AND source_id = ?", "zone-device", serviceId, deviceId).
        First(nil, fields)

    if sql.ErrNoRows == err {
        err = nil // it's not an error
    }

    return
}


func (cfg *Configuration) getDeviceName(deviceId int64) (name string, err error) {
    fields := dblayer.Fields{"name": &name}
    err = db.Table("devices").Seek(deviceId).First(nil, fields)
    if sql.ErrNoRows == err {
        err = nil // it's not an error
    }
    //cfg.Log("GDN:", name, serviceId, deviceId)
    return
}


func (cfg *Configuration) getOneDevice(fields dblayer.Fields, serviceId int64, handle string) (id int64, err error) {
    if nil == fields {
        fields = dblayer.Fields{"id": &id}
    }
    err = db.Table("devices").
        Seek("service_id = ? AND handle = ?", serviceId, handle).
        First(nil, fields)

    if sql.ErrNoRows == err {
        err = nil // it's not an error
    }
    
    return
}

func (cfg *Configuration) GlobalDeviceId(serviceId int64, handle, name string) (id int64, err error) {
    id, err = cfg.getOneDevice(nil, serviceId, handle)
    if nil != err {
        return
    }

    fields := dblayer.Fields {
        "name":         name,
        "last_seen":    time.Now()}

    if 0 == id {
        fields["service_id"] = serviceId
        fields["handle"] = handle
        id, err = db.Table("devices").Insert(nil, fields)
    } else {
        _, err = db.Table("devices").Seek(id).Update(nil, fields)
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

    err = db.Table("devices").
        Seek("handle IS NOT NULL AND service_id = ?", serviceId).
        Rows(nil, fields).
        Each(func () {
            list = append(list, *dev)
        })
    
    return
}

func (cfg *Configuration) SaveDevice(serviceId int64, dev *Device, data interface{}) (err error) {
    fields := dblayer.Fields {
        "id":           &dev.Id,
        "service_id":   serviceId,
        "handle":       dev.Handle,
        "name":         dev.Name,
        "last_seen":    time.Now()}
    
    if nil != data {
        fields["data"], _ = json.Marshal(data)
    }

    err = db.Table("devices").Save(nil, fields)
    return
}

func (cfg *Configuration) DeleteDevice(id int64) (err error) {
    fields := dblayer.Fields {
        "handle":       nil,
        "last_seen":    time.Now()} // deletion time

    tx, err := db.Tx(qTimeout)
    if nil != err {return}
    defer func () {completeTx(tx, err)}()
    
    _, err = db.Table("devices").Seek(id).Update(tx, fields)
    if nil != err {return}
    err = db.Table("external_links").Delete(tx, "link = ? AND target_id = ?", "user-device", id)
    return
}

func (cfg *Configuration) TouchDevice(serviceId int64, dev *Device) (err error) {
    dev.LastSeen = time.Now()
    fields := dblayer.Fields {"last_seen": dev.LastSeen}
    _, err = db.Table("devices").Seek(dev.Id).Update(nil, fields)
    return
}
