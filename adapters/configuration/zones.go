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

    rows, values := cfg.Table("zones").Seek("archived IS NULL").Get(fields)
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

    rows, values := cfg.Table("zones").Seek(id).Get(fields)
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
    table := cfg.Table("external_links")
    table.Delete("target_id = ? AND link", id, clean)

    fields := dblayer.Fields{"archived": time.Now()}
    cfg.Table("zones").Seek(id).Update(fields)
}


func (cfg *Configuration) dbUpdateZone(zone *Zone) {
    fields := zoneFields(zone)
    cfg.Table("zones").Save(fields)
}
