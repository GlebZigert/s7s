package configuration

import (
    "encoding/json"
    "s7server/api"
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
    err := cfg.CompleteShift(cid)
    catch(err)
    return true, false // don't broadcast
}

///////////////////////////////////////////////////////////////////
/////////////////////// L O C A T I O N S /////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listLocations(cid int64, data []byte) (interface{}, bool) {
    res, err := cfg.entranceEvents()
    catch(err)
    return res, false // don't broadcast
}

///////////////////////////////////////////////////////////////////
////////////////////////// E V E N T S ////////////////////////////
///////////////////////////////////////////////////////////////////
// set reason & action for event
func (cfg *Configuration) describeEvent(cid int64, data []byte) (interface{}, bool) {
    event := new(api.Event)
    json.Unmarshal(data, &event)
    // TODO: use err separately from return value?
    if cfg.dbDescribeEvent(event) {
        return event, true // broadcast
    } else { // event id unknown or db error
        return false, false // don't broadcast error
    }
}

func (cfg *Configuration) loadJournal(cid int64, data []byte) (interface{}, bool) {
    var serviceId int64
    json.Unmarshal(data, &serviceId)
    res, err := cfg.dbLoadJournal(cid, serviceId) // TODO: handle err
    catch(err)
    return res, false // don't broadcast
}

func (cfg *Configuration) listEvents(cid int64, data []byte) (interface{}, bool) {
    filter := new(EventFilter)
    json.Unmarshal(data, filter) // TODO: handle err
    res, err := cfg.loadEvents(filter)
    catch(err)
    return res, false // don't broadcast
}

///////////////////////////////////////////////////////////////////
//////////////////////////// A L G O S ////////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listAlgorithms(cid int64, data []byte) (interface{}, bool) {
    res, err := cfg.loadAlgorithms()
    catch(err)
    return res, false // don't broadcast
}

func (cfg *Configuration) updateAlgorithm(cid int64, data []byte) (interface{}, bool) {
    algorithm := new(api.Algorithm)
    json.Unmarshal(data, algorithm) // TODO: handle err
    err := cfg.dbUpdateAlgorithm(algorithm)
    catch(err)
    return algorithm, true // broadcast
}

func (cfg *Configuration) deleteAlgorithm(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    err := cfg.dbDeleteAlgorithm(id)
    catch(err)
    return id, true // broadcast
}

///////////////////////////////////////////////////////////////////
//////////////////////////// Z O N E S ////////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listZones(cid int64, data []byte) (interface{}, bool) {
    ee := make(map[int64] []api.Event) // entrance events
    zones, err := cfg.loadZones()
    catch(err)
    
    events, err := cfg.entranceEvents()
    catch(err)
    
    // entrance events
    for i := range events {
        ee[events[i].ZoneId] = append(ee[events[i].ZoneId], events[i])
    }
    
    // lin
    for i := range zones {
        zones[i].EntranceEvents = ee[zones[i].Id]
        // TODO: handle err
        zones[i].Devices, err = cfg.LoadLinks(zones[i].Id, "zone-device")
        catch(err)
    }
    
    return zones, false // don't broadcast
}

func (cfg *Configuration) updateZone(cid int64, data []byte) (interface{}, bool) {
    zone := new(Zone)
    json.Unmarshal(data, zone) // TODO: handle err
    if 1 != zone.Id { // should not update "Внешняя территория"
        err := cfg.dbUpdateZone(zone)
        catch(err)
        err = cfg.SaveLinks(zone.Id, "zone-device", zone.Devices)
        catch(err)
        return zone, true // broadcast
    } else {
        return "error", false // don't broadcast
    }
}

func (cfg *Configuration) deleteZone(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    if 1 != id {
        err := cfg.dbDeleteZone(id)
        catch(err)
        return id, true // broadcast
    } else {// should not delete "Внешняя территория"
        return "error", false // don't broadcast
    }
}

///////////////////////////////////////////////////////////////////
///////////////////////////// M A P S /////////////////////////////
///////////////////////////////////////////////////////////////////
func (cfg *Configuration) listMaps(cid int64, data []byte) (interface{}, bool) {
    maps, err := cfg.loadMaps()
    catch(err)
    // let's filter them
    // 1. collect all devices
    var devices []int64
    for i := range maps {
        for j:= range maps[i].Shapes {
            devices = append(devices, maps[i].Shapes[j].DeviceId)
        }
    }
    return maps, false // don't broadcast
}

func (cfg *Configuration) updateMap(cid int64, data []byte) (interface{}, bool) {
    myMap := new(Map)
    json.Unmarshal(data, myMap) // TODO: handle err
    err := cfg.dbUpdateMap(myMap)
    catch(err)
    return myMap, true // broadcast
}

func (cfg *Configuration) deleteMap(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    err := cfg.dbDeleteMap(id)
    catch(err)
    return id, true // broadcast
}


/////////////////////////////////////////////////////////////////////
///////////////////////////// R U L E S /////////////////////////////
/////////////////////////////////////////////////////////////////////
func (cfg *Configuration) listRules(cid int64, data []byte) (interface{}, bool) {
    res, err := cfg.loadAllRules()
    catch(err)
    return res, false // don't broadcast
}

func (cfg *Configuration) updateRule(cid int64, data []byte) (interface{}, bool) {
    //res := cfg.loadUsers()
    //res := cfg.loadRules()
    //var res []*Rule
    rule := new(Rule)
    json.Unmarshal(data, rule) // TODO: handle err
    err := cfg.dbUpdateRules(rule)
    catch(err)
    return rule, rule.Id > 0 // broadcast
}

func (cfg *Configuration) deleteRule(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    err := cfg.dbDeleteRule(id)
    catch(err)
    return id, true // broadcast
}

/////////////////////////////////////////////////////////////////////
///////////////////////////// U S E R S /////////////////////////////
/////////////////////////////////////////////////////////////////////
func (cfg *Configuration) userInfo(cid int64, data []byte) (interface{}, bool) {
    var id int64
    var err error
    json.Unmarshal(data, &id)
    //res := cfg.loadUsers()
    user := User{Id: id}
    user.Zones, err = cfg.LoadLinks(id, "user-zone")
    catch(err)
    user.Devices, err = cfg.LoadLinks(id, "user-device")
    catch(err)
    user.Cards, err = cfg.loadUserCards(id)
    catch(err)

    return user, false
}

func (cfg *Configuration) listUsers(cid int64, data []byte) (interface{}, bool) {
    users, err := cfg.loadUsers()
    catch(err)
    return users, false
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
    err := cfg.dbUpdateUser(user, filter)
    catch(err)
    user.Password = ""
    if 0 != user.Id && nil != filter["devices"] {
        children, err = cfg.cache.expandChildren(user.Id) // TODO: handle err
    }
    catch(err)
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
    if id > 1 {
        err := cfg.dbDeleteUser(id)
        catch(err)
        return id, true
    } else {
        return "Нельзя удалить встроенного администратора", false
    }
}

/////////////////////////////////////////////////////////////////////
////////////////////////// S E R V I C E S //////////////////////////
/////////////////////////////////////////////////////////////////////

func (cfg *Configuration) deleteService(cid int64, data []byte) (interface{}, bool) {
    var id int64
    json.Unmarshal(data, &id)
    err := cfg.dbDeleteService(id)
    catch(err)
    //panic("AAAAAAAAAAAAAAAAAA")
    return id, true
}


func (cfg *Configuration) updateService(cid int64, data []byte) (interface{}, bool) {
    service := api.Settings{}
    json.Unmarshal(data, &service) // TODO: handle err
    service.Password = service.NewPassword
    service.NewPassword = ""
    service.DBPassword = service.NewDBPassword
    service.NewDBPassword = ""

    var err error
    if 0 == service.Id {
        // TODO: timeout for multiple submissions (no spam!)
        err = cfg.newService(&service)
    } else {
        err = cfg.updService(service)
    }
    catch(err)
    return &service, true
}

func catch(err error) {
    if nil != err {
        panic(err)
    }
}
