package configuration

import (
    "log"
    "encoding/json"
    "../../api"
)

func (svc *Configuration) resetAlarm(cid int64, data []byte) (interface{}, bool) {
    events := api.EventsList{{
        UserId: cid,
        Class: api.EC_INFO_ALARM_RESET}}
    svc.Broadcast("Events", events)

    return true, true // broadcast
}

func (cfg *Configuration) runAlarm(cid int64, data []byte) (interface{}, bool) {
    events := api.EventsList{{
        ServiceId: 0,
        ServiceName: "Система",
        UserId: cid,
        Class: api.EC_GLOBAL_ALARM}}
    cfg.Broadcast("Events", events)

    return true, false // don't broadcast
}

func (cfg *Configuration) completeShift(cid int64, data []byte) (interface{}, bool) {
    cfg.CompleteShift(cid)
    return true, false // don't broadcast
}

///////////////////////////////////////////////////////////////////
/////////////////////// L O C A T I O N S /////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listLocations(cid int64, data []byte) (interface{}, bool) {
    res := cfg.entranceEvents()
    //log.Println("FFF:", filter)
    return res, false // don't broadcast
}

///////////////////////////////////////////////////////////////////
////////////////////////// E V E N T S ////////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) describeEvent(cid int64, data []byte) (interface{}, bool) {
    event := new(api.Event)
    json.Unmarshal(data, &event)
    if cfg.dbDescribeEvent(event) {
        /*if event.ServiceId > 0 { // reset alarm
            cfg.Broadcast("Events", api.EventsList{{
                ServiceId: event.ServiceId,
                DeviceId: event.DeviceId,
                Class: api.EC_INFO_ALARM_RESET}})
        }*/
        return event, true // broadcast
    } else { // event id unknown
        return false, false // don't broadcast error
    }
}

func (cfg *Configuration) loadJournal(cid int64, data []byte) (interface{}, bool) {
    var serviceId int64
    json.Unmarshal(data, &serviceId)
    res := cfg.dbLoadJournal(cid, serviceId)
    //log.Println("FFF:", filter)
    return res, false // don't broadcast
}

func (cfg *Configuration) listEvents(cid int64, data []byte) (interface{}, bool) {
    filter := new(EventFilter)
    json.Unmarshal(data, filter) // TODO: handle err
    res := cfg.loadEvents(filter)
    //log.Println("FFF:", filter)
    return res, false // don't broadcast
}

///////////////////////////////////////////////////////////////////
//////////////////////////// A L G O S ////////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listAlgorithms(cid int64, data []byte) (interface{}, bool) {
    res := cfg.loadAlgorithms()
    return res, false // don't broadcast
}

func (cfg *Configuration) updateAlgorithm(cid int64, data []byte) (interface{}, bool) {
    algorithm := new(api.Algorithm)
    json.Unmarshal(data, algorithm) // TODO: handle err
    cfg.dbUpdateAlgorithm(algorithm)
    return algorithm, true // broadcast
}

func (cfg *Configuration) deleteAlgorithm(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    err := cfg.dbDeleteAlgorithm(id)
    if nil != err {
        panic(err)
    }
    return id, true // broadcast
}

///////////////////////////////////////////////////////////////////
//////////////////////////// Z O N E S ////////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listZones(cid int64, data []byte) (interface{}, bool) {
    ee := make(map[int64] []api.Event) // entrance events
    zones := cfg.loadZones()
    events := cfg.entranceEvents()
    
    // entrance events
    for i := range events {
        ee[events[i].ZoneId] = append(ee[events[i].ZoneId], events[i])
    }
    
    // lin
    for i := range zones {
        zones[i].EntranceEvents = ee[zones[i].Id]
        zones[i].Devices = cfg.LoadLinks(zones[i].Id, "zone-device")
    }
    
    // load zones links
    
    
    return zones, false // don't broadcast
}

func (cfg *Configuration) updateZone(cid int64, data []byte) (interface{}, bool) {
    zone := new(Zone)
    json.Unmarshal(data, zone) // TODO: handle err
    if 1 != zone.Id { // should not update "Внешняя территория"
        cfg.dbUpdateZone(zone)
        cfg.SaveLinks(zone.Id, "zone-device", zone.Devices)
        return zone, true // broadcast
    } else {
        return "error", false // don't broadcast
    }
}

func (cfg *Configuration) deleteZone(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    if 1 != id {
        cfg.dbDeleteZone(id)
        return id, true // broadcast
    } else {// should not delete "Внешняя территория"
        return "error", false // don't broadcast
    }
}

///////////////////////////////////////////////////////////////////
///////////////////////////// M A P S /////////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listMaps(cid int64, data []byte) (interface{}, bool) {
    maps := cfg.loadMaps()
    // let's filter them
    // 1. collect all devices
    var devices []int64
    for i := range maps {
        for j:= range maps[i].Shapes {
            devices = append(devices, maps[i].Shapes[j].DeviceId)
        }
    }
    cfg.Log("map devs", devices)
    return maps, false // don't broadcast
}

func (cfg *Configuration) updateMap(cid int64, data []byte) (interface{}, bool) {
    myMap := new(Map)
    json.Unmarshal(data, myMap) // TODO: handle err
    cfg.dbUpdateMap(myMap)
    return myMap, true // broadcast
}

func (cfg *Configuration) deleteMap(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    cfg.dbDeleteMap(id)
    return id, true // broadcast
}


/////////////////////////////////////////////////////////////////////
///////////////////////////// R U L E S /////////////////////////////
/////////////////////////////////////////////////////////////////////
func (cfg *Configuration) listRules(cid int64, data []byte) (interface{}, bool) {
    res := cfg.loadAllRules()
    return res, false // don't broadcast
}

func (cfg *Configuration) updateRule(cid int64, data []byte) (interface{}, bool) {
    //res := cfg.loadUsers()
    //res := cfg.loadRules()
    //var res []*Rule
    rule := new(Rule)
    json.Unmarshal(data, rule) // TODO: handle err
    cfg.dbUpdateRules(rule)
    return rule, rule.Id > 0 // broadcast
}

func (cfg *Configuration) deleteRule(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    cfg.dbDeleteRule(id)
    return id, true // broadcast
}

/////////////////////////////////////////////////////////////////////
///////////////////////////// U S E R S /////////////////////////////
/////////////////////////////////////////////////////////////////////
func (cfg *Configuration) userInfo(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    //res := cfg.loadUsers()
    user := User{Id: id}
    user.Zones = cfg.LoadLinks(id, "user-zone")
    user.Devices = cfg.LoadLinks(id, "user-device")
    user.Cards = cfg.loadUserCards(id)
    return user, false
}

func (cfg *Configuration) listUsers(cid int64, data []byte) (interface{}, bool) {
    //res := cfg.loadUsers()
    res := cfg.loadUsers()
    return res, false
}

func (cfg *Configuration) updateUser(cid int64, data []byte) (interface{}, bool) {
    var children []int64
    var filter map[string] interface{}
    json.Unmarshal(data, &filter)
    // TODO: use filer based on ARM role / user permissions
    //cfg.Log("UpdateUser", string(data), filter)    

    user := new(User)
    json.Unmarshal(data, user) // TODO: handle err
    //cfg.Log("UpdateUser", *user)
    // TODO: timeout lo timit multiple submissions (no spam!)
    cfg.Log("Update/create user:", *user)
    if 0 != user.Id && nil != filter["devices"] {
        // TODO: check real modifications, not just updates
        list := []int64{user.Id}
        children = cfg.childrenList(user.Id)
        cfg.cache.RLock()
        for _, id := range children {
            list = append(list, cfg.cache.children[id]...)
        }
        cfg.cache.RUnlock()
        children = append(children, list...)
        cfg.Log("User permissions updated", children)
        
    }
    cfg.dbUpdateUser(user, filter)
    user.Password = ""
    
    if len(children) > 0 {
        cfg.Broadcast("AffectedUsers", children)
    }
    
    if len(user.Errors) > 0 {
        return user, false
    } else {
        return user, true
    }
}

func (cfg *Configuration) deleteUser(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    log.Println("[Configuration] Delete user", id)
    if 1 != id {
        cfg.dbDeleteUser(id)
        return id, true
    } else {
        return "Нельзя удалить встроенного администратора", false
    }
}

/////////////////////////////////////////////////////////////////////
////////////////////////// S E R V I C E S //////////////////////////
/////////////////////////////////////////////////////////////////////
func (cfg *Configuration) listServices(cid int64, data []byte) (interface{}, bool) {
    // TODO: load only required services
    cfg.Log("################# CID:", cid)
    res := cfg.loadServices()
    for _, r := range res {
        r.DBPassword = ""
        r.Password = ""
    }
    return res, false
}

func (cfg *Configuration) deleteService(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    log.Println("[Configuration] Delete", id)
    cfg.dbDeleteService(id)
    return id, true
}


func (cfg *Configuration) updateService(cid int64, data []byte) (interface{}, bool) {
    service := api.Settings{}
    json.Unmarshal(data, &service) // TODO: handle err
    service.Password = service.NewPassword
    service.NewPassword = ""
    service.DBPassword = service.NewDBPassword
    service.NewDBPassword = ""
    log.Println(service)
    if 0 == service.Id {
        // new service
        // TODO: timeout for multiple submissions (no spam!)
        //log.Println("New service:", service)
        cfg.newService(&service)
    } else {
        // update service
        //log.Println("Update service:", service)
        cfg.updService(service)
    }
    return &service, true
}
