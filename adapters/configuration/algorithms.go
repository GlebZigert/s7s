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


func (cfg *Configuration) loadAlgorithms() (list []api.Algorithm){
    algorithm := new(api.Algorithm)
    fields := AlgorithmFields(algorithm)

    rows, values := cfg.Table("algorithms").Get(fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *algorithm)
    }
    return
}

func (cfg *Configuration) dbDeleteAlgorithm(id int64) {
    cfg.Table("algorithms").Delete(id)
}


func (cfg *Configuration) dbUpdateAlgorithm(algorithm *api.Algorithm) {
    fields := AlgorithmFields(algorithm)
    cfg.Table("algorithms").Save(fields)
}

func (cfg *Configuration) findDevAlgorithms(e *api.Event) (list []api.Algorithm) {
    source := cfg.Table("algorithms")
    algorithm := new(api.Algorithm)
    fields := AlgorithmFields(algorithm)

    if e.UserId > 0 && e.Class == api.EC_ENTER_ZONE {
        fromZone := cfg.visitorLocation(e.UserId)
        q := "user_id = ? AND (zone_id = ? AND event = ? OR zone_id = ? AND event = ?)"
        source = source.Seek(q, e.UserId, e.ZoneId, api.EC_ENTER_ZONE, fromZone, api.EC_EXIT_ZONE)

    } else if e.ServiceId > 0 {
        q := "service_id = ? AND device_id = ? AND (from_state = ? OR from_state < 0) AND (event = ? OR event < 0)"
        source = source.Seek(q, e.ServiceId, e.DeviceId, e.FromState, e.Event)
    } else {
        return
    }
    
    rows, values := source.Get(fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *algorithm)
    }
    return
}

