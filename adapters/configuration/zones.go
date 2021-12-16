package configuration

import (
//    "database/sql"
//    "../../api"
    "time"
    "../../dblayer"
)

func zoneFields(zone *Zone) dblayer.Fields {
    fields := dblayer.Fields {
        "id": &zone.Id,
        "name": &zone.Name,
        "max_visitors": &zone.MaxVisitors}
    return fields
}


func (cfg *Configuration) loadZones() (list []Zone) {
    zone := new(Zone)
    fields := zoneFields(zone)

    rows, values, _ := db.Table("zones").Seek("archived IS NULL").Get(nil, fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *zone)
    }
    return
}

func (cfg *Configuration) getZone(id int64) (zone *Zone) {
    zone = new(Zone)
    fields := zoneFields(zone)

    rows, values, _ := db.Table("zones").Seek(id).Get(nil, fields)
    defer rows.Close()

    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
    }
    return
}

func (cfg *Configuration) dbDeleteZone(id int64) {
    // "zone-device" => security devices in zone
    // "device-zone" => ACS point in/out
    clean := []string{"zone-device", "device-zone", "user-zone"}
    table := db.Table("external_links")
    table.Delete(nil, "target_id = ? AND link", id, clean)

    fields := dblayer.Fields{"archived": time.Now()}
    db.Table("zones").Seek(id).Update(nil, fields)
}


func (cfg *Configuration) dbUpdateZone(zone *Zone) {
    fields := zoneFields(zone)
    db.Table("zones").Save(nil, fields)
}
