package configuration

import (
    "time"
    "encoding/json"
    "../../dblayer"
)

func (cfg *Configuration) deviceZone(serviceId, deviceId int64) (zoneId int64) {
    fields := dblayer.Fields {"target_id": &zoneId}

    rows, values := cfg.Table("external_links").
        Seek("link = ? AND scope_id = ? AND source_id = ?", "zone-device", serviceId, deviceId).
        Get(fields)
    defer rows.Close()

    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
    }

    return
}


func (cfg *Configuration) getDeviceName(serviceId, deviceId int64) (name string) {
    fields := dblayer.Fields{"name": &name}

    rows, values := cfg.Table("devices").
        Seek("service_id = ? AND id = ?", serviceId, deviceId).
        Get(fields, 1)
    
    if rows.Next() {
        _ = rows.Scan(values...)
    }
    rows.Close()
    cfg.Log("GDN:", name, serviceId, deviceId)
    return
}


func (cfg *Configuration) getOneDevice(fields dblayer.Fields, serviceId int64, handle string) (id int64, err error) {
    if nil == fields {
        fields = dblayer.Fields{"id": &id}
    }
    rows, values := cfg.Table("devices").
        Seek("service_id = ? AND handle = ?", serviceId, handle).
        Get(fields, 1)
    
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
        id = cfg.Table("devices").Insert(fields)
    } else {
        cfg.Table("devices").Seek(id).Update(fields)
    }
    return
}

func (cfg *Configuration) LoadDevices(serviceId int64) (list []Device) {
    dev := new(Device) 
    fields := dblayer.Fields {
        "id":           &dev.Id,
        //"service_id":   &dev.ServiceId,
        "handle":       &dev.Handle,
        "name":         &dev.Name,
        "last_seen":    &dev.LastSeen,
        "data":         &dev.Data}

    rows, values := cfg.Table("devices").Seek("handle IS NOT NULL AND service_id = ?", serviceId).Get(fields)
    
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *dev)
    }
    rows.Close()
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

    cfg.Table("devices").Save(fields)
}

func (cfg *Configuration) DeleteDevice(id int64) (err error) {
    fields := dblayer.Fields {
        "handle":       nil,
        "last_seen":    time.Now()} // deletion time

    cfg.Table("devices").Seek(id).Update(fields)
    cfg.Table("external_links").Delete("link = ? AND target_id = ?", "user-device", id)
    return
}

func (cfg *Configuration) TouchDevice(serviceId int64, dev *Device) {
    dev.LastSeen = time.Now()
    fields := dblayer.Fields {"last_seen": dev.LastSeen}
    cfg.Table("devices").Seek(dev.Id).Update(fields)
}
