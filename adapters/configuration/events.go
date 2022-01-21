package configuration

import (
//    "database/sql"
    "time"
    "strings"
    "../../api"
    "../../dblayer"
)

func eventFields(event *api.Event) dblayer.Fields {
    fields := dblayer.Fields {
        "id": &event.Id,
        "external_id": &event.ExternalId,
        "service_id": &event.ServiceId,
        "device_id": &event.DeviceId,
        "zone_id": &event.ZoneId,
        "from_state": &event.FromState,
        "event": &event.Event,
        "class": &event.Class,
        "text": &event.Text,
        "user_id": &event.UserId,
        "commands": &event.Commands,
        "time": &event.Time}

    return fields
}

func eventFieldsEx(event *api.Event) dblayer.Fields {
    fields := dblayer.Fields {
        "e.id": &event.Id,
        "e.external_id": &event.ExternalId,
        "e.from_state": &event.FromState,
        "e.event": &event.Event,
        "e.class": &event.Class,
        "e.text": &event.Text,
        "e.time": &event.Time,
        "e.service_id": &event.ServiceId,
        "e.device_id": &event.DeviceId,
        "e.user_id": &event.UserId,
        "e.zone_id": &event.ZoneId,
        "e.commands": &event.Commands,
        "e.reason": &event.Reason,
        "e.reaction": &event.Reaction,
        "COALESCE(s.title, 'Система')": &event.ServiceName,
        "COALESCE(d.name, '')": &event.DeviceName,
        "COALESCE(z.name, '')": &event.ZoneName,
        "COALESCE(u.name || ' ' || u.surename, '')": &event.UserName}

    return fields
}


func (cfg *Configuration) dbDescribeEvent(event *api.Event) bool {
    fields := dblayer.Fields {
        "reason": event.Reason,
        "reaction": event.Reaction}
    num, err := db.Table(`events`).Seek(event.Id).Update(nil, fields)
    return  num > 0 && nil != err
}

func (cfg *Configuration) dbLoadJournal(userId, serviceId int64) (list api.EventsList, err error) {
    //user := cfg.GetUser(userId)
    //devices := cfg.LoadLinks(userId, "user-device")

    event := new(api.Event)
    fields := eventFieldsEx(event)
    table := db.Table(`events e
                        LEFT JOIN services s ON e.service_id = s.id
                        LEFT JOIN devices d ON e.device_id = d.id
                        LEFT JOIN zones z ON e.zone_id = z.id
                        LEFT JOIN users u ON e.user_id = u.id`)
    
    from := time.Now().AddDate(0, 0, -2).Unix()
    //var fromId int64
    fromId, err := cfg.currentShiftId(userId)
    if nil != err {
        return
    }
    //cfg.Log("SHIFT EVENT ID #", fromId)
    // TODO: load events starting from session opening
    rows, values, _ := table.
        Seek("e.service_id = ? AND e.time > ? AND e.id >= ?", serviceId, from, fromId).
        Order("e.time, e.id").
        Get(nil, fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        /*if user.Role != 1 {
            for i := range devices {
                if userId == event.UserId ||
                    devices[i][0] == event.ServiceId &&
                    devices[i][1] == event.DeviceId {
                    list = append(list, *event)
                    break
                }
            }
            
        } else {*/
            list = append(list, *event)
        //}
    }
    //cfg.Log("JRN:", list)
    return
}


func (cfg *Configuration) loadEvents(filter *EventFilter) (list []api.Event){
    event := new(api.Event)
    fields := eventFieldsEx(event)

    table := db.Table(`events e
                        LEFT JOIN services s ON e.service_id = s.id
                        LEFT JOIN devices d ON e.device_id = d.id
                        LEFT JOIN zones z ON e.zone_id = z.id
                        LEFT JOIN users u ON e.user_id = u.id`)
    
    cond := make([]string, 0)
    args := make([]interface{}, 0)
    
    cond = append(cond, "e.time >= ?")
    args = append(args, filter.Start.Unix())
    
    cond = append(cond, "e.time <= ?")
    args = append(args, filter.End.Unix())

    if filter.ServiceId > 0 {
        cond = append(cond, "e.service_id = ?")
        args = append(args, filter.ServiceId)
    }
    
    if filter.UserId > 0 {
        cond = append(cond, "e.user_id = ?")
        args = append(args, filter.UserId)
    }
    
    if filter.Limit < 100 || filter.Limit > 1000 {
        filter.Limit = 100
    }
    

    var classes []string
    for _, class := range api.EventClasses {
        if 0 == filter.Class || filter.Class & (1 << uint(class / 100)) > 0 {
            //classes = append(classes, class)
            classes = append(classes, "e.class BETWEEN ? AND ?")
            args = append(args, class, class + 99)
        }
        //cfg.Log(filter.Class , (1 << class), classes)
    }
    cond = append(cond, "(" + strings.Join(classes, " OR ") + ")")
    
    args = append([]interface{}{strings.Join(cond, " AND ")}, args...)
    rows, values, _ := table.Seek(args...).Order("e.time, e.id").Get(nil, fields, filter.Limit)
    
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *event)
    }
    return
}

func (cfg *Configuration) dbLogEvent(event *api.Event) (err error) {
    fields := eventFields(event)
    delete(fields, "id")
    event.Id, err = db.Table("events").Insert(nil, fields)
    return
}

func (cfg *Configuration) dbLogEvents(events api.EventsList) (err error) {
	tx, err := db.Tx(qTimeout)
    if nil != err {
        return
    }
    for i := 0; i < len(events) && nil == err; i++ {
        fields := eventFields(&events[i])
        delete(fields, "id")
        events[i].Id, err = db.Table("events").Insert(tx, fields)
    }
    
    if nil == err {
        return tx.Commit() // TODO: what if commit failed? It's time to panic?
    }
    
    for i := range events {
        events[i].Id = 0 // mark events unprocessed
    }
    return tx.Rollback() // TODO: what if rollback failed? It's time to panic?
}

func (cfg *Configuration) importEvent(event *api.Event) (err error) {
    event.Id = 0 // just in case
    ev := new(api.Event)
    fields := eventFields(ev)
    rows, values, err := db.Table("events").
        Seek("external_id = 0 AND service_id = ? AND device_id = ? AND time = ? AND event = ?",
             event.ServiceId, event.DeviceId, event.Time, event.Event).
        Order("id").Get(nil, fields, 1)
    if nil != err {
        return
    }
    if rows.Next() {
        err = rows.Scan(values...)
        event.Id = ev.Id
    }
    rows.Close()

    if 0 == event.Id {
        err = cfg.dbLogEvent(event)
    } else { // Unknown event, save it
        _, err = db.Table("events").Seek(event.Id).Update(nil, dblayer.Fields{"external_id": event.ExternalId})
    }
    return
}

func (cfg *Configuration) ImportEvents(events []api.Event) (err error) {
    defer func () {cfg.cdbe(err)}()
    for i := range events {
        err = cfg.importEvent(&events[i])
        if nil != err {
            break
        }
    }
    return
}

func (cfg *Configuration) GetLastEvent(serviceId int64) (event *api.Event, err error) {
    defer func () {cfg.cdbe(err)}()
    event = new(api.Event)
    fields := eventFields(event)
    rows, values, err := db.Table("events").
        Seek("external_id > 0 AND service_id = ?", serviceId).
        Order("external_id DESC").
        Get(nil, fields, 1)
    defer rows.Close()
    if rows.Next() {
        err = rows.Scan(values...)
    } else {
        event = nil
    }
    if err != nil {
        event = nil
    }
    return
}
