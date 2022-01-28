package configuration

import (
//    "database/sql"
    "../../api"
    "../../dblayer"
)

func AlgorithmFields(algorithm *api.Algorithm) dblayer.Fields {
    fields := dblayer.Fields {
        "id": &algorithm.Id,
        "name": &algorithm.Name,
        "service_id": &algorithm.ServiceId,
        "device_id": &algorithm.DeviceId,
        "zone_id": &algorithm.ZoneId,
        "user_id": &algorithm.UserId,
        "target_service_id": &algorithm.TargetServiceId,
        "target_device_id": &algorithm.TargetDeviceId,
        "target_zone_id": &algorithm.TargetZoneId,
        "from_state": &algorithm.FromState,
        "event": &algorithm.Event,
        "command": &algorithm.Command,
        "argument": &algorithm.Argument}

    return fields
}


func (cfg *Configuration) loadAlgorithms() (list []api.Algorithm, err error){
    algorithm := new(api.Algorithm)
    fields := AlgorithmFields(algorithm)

    rows, values, err := db.Table("algorithms").Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()

    for rows.Next() {
        err = rows.Scan(values...)
        if nil != err {
            break
        }
        list = append(list, *algorithm)
    }
    return
}

func (cfg *Configuration) dbDeleteAlgorithm(id int64) (err error) {
    err = db.Table("algorithms").Delete(nil, id)
    return
}


func (cfg *Configuration) dbUpdateAlgorithm(algorithm *api.Algorithm) (err error) {
    fields := AlgorithmFields(algorithm)
    err = db.Table("algorithms").Save(nil, fields)
    return
}

func (cfg *Configuration) findDevAlgorithms(e *api.Event) (list []api.Algorithm, err error) {
    source := db.Table("algorithms")
    algorithm := new(api.Algorithm)
    fields := AlgorithmFields(algorithm)

    if e.UserId > 0 && e.Class == api.EC_ENTER_ZONE {
        var fromZone int64
        fromZone, err = cfg.visitorLocation(e.UserId)
        q := "user_id = ? AND (zone_id = ? AND event = ? OR zone_id = ? AND event = ?)"
        source = source.Seek(q, e.UserId, e.ZoneId, api.EC_ENTER_ZONE, fromZone, api.EC_EXIT_ZONE)
    } else if e.ServiceId > 0 {
        q := "service_id = ? AND device_id = ? AND (from_state = ? OR from_state < 0) AND (event = ? OR event < 0)"
        source = source.Seek(q, e.ServiceId, e.DeviceId, e.FromState, e.Event)
    } else {
        return
    }
    
    if nil != err {
        return
    }
    
    rows, values, err := source.Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()

    for rows.Next() && nil == err {
        err = rows.Scan(values...)
        if nil == err {
            list = append(list, *algorithm)
        }
    }
    if nil == err {
        err = rows.Err()
    }
    return
}

