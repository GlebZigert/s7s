package configuration

import (
//    "database/sql"
//    "s7server/api"
    "time"
    "s7server/dblayer"
)

func zoneFields(zone *Zone) dblayer.Fields {
    fields := dblayer.Fields {
        "id": &zone.Id,
        "name": &zone.Name,
        "max_visitors": &zone.MaxVisitors}
    return fields
}


func (cfg *Configuration) loadZones() (list ZoneList, err error) {
    zone := new(Zone)
    fields := zoneFields(zone)

    rows, values, err := db.Table("zones").Seek("archived IS NULL").Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()

    for rows.Next() {
        err = rows.Scan(values...)
        if nil != err {
            break
        }
        list = append(list, *zone)
    }
    return
}

func (cfg *Configuration) getZone(id int64) (zone *Zone, err error) {
    zone = new(Zone)
    fields := zoneFields(zone)

    rows, values, err := db.Table("zones").Seek(id).Get(nil, fields)
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

func (cfg *Configuration) dbDeleteZone(id int64) (err error) {
    tx, err := db.Tx(qTimeout)
    if nil != err {
        return
    }
    defer func () {completeTx(tx, err)}()

    // "zone-device" => security devices in zone
    // "device-zone" => ACS point in/out
    clean := []string{"zone-device", "device-zone", "user-zone"}
    table := db.Table("external_links")
    
    err = table.Delete(tx, "target_id = ? AND link", id, clean)
    if nil != err {
        return
    }

    fields := dblayer.Fields{"archived": time.Now()}
    _, err = db.Table("zones").Seek(id).Update(tx, fields)
    return
}


func (cfg *Configuration) dbUpdateZone(zone *Zone) error {
    fields := zoneFields(zone)
    return db.Table("zones").Save(nil, fields)
}
