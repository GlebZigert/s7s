package sigur

import (
    "log"
    "time"
    "strconv"
    //"errors"
)

const (
    cacheTimeout = 30 // seconds
    eventMaxWait = 3 // seconds
    eventsFlushInterval = 5 // seconds
)

func (api *Sigur) messageServer(c chan []string) {
    var ok bool
    var data []string
    
    ticker := time.NewTicker(eventsFlushInterval * time.Second)
	defer ticker.Stop()
    
    api.devices = make(map[int] *Device)
    api.objects = make(map[int] *ObjectInfo)
	
	for {
        ok = false
        select {
            case _ = <-ticker.C:
            ok = false // TODO: data = []?
            
            case data, ok = <-c:
                if !ok {
                    // channel closed
                    log.Println(api.GetName(), "Event dispatcher stopped")
                    return
                }
        }
        if ok {
            // have data to process
            switch data[0] {
                case "OBJECTINFO":
                    api.cacheObjectInfo(data)

                case "EVENT_CE":
                    api.storeEvent(data)
                
                case "APINFO":
                api.cacheDevice(data)
            }
        }
        // check and send events
        api.processEvents()
    }
}

func (api *Sigur) cacheObjectInfo(data []string) {
    o := parseInfo(data, 2)
    if len(o) > 0 {
        id, _ := strconv.Atoi(o["ID"])
        api.objects[id] = &ObjectInfo{
            Id: id,
            Name: o["NAME"],
            Ready: true,
            Timestamp: time.Now().Unix()}
    }
}

func (api *Sigur) cacheDevice(data []string) {
    o := parseInfo(data, 1)
    if len(o) > 0 {
        id, _ := strconv.Atoi(o["ID"])
        api.Lock()
        defer api.Unlock()
        // TODO: compare old & new and send message about update
        api.devices[id] = &Device{
            Id: id,
            Name: o["NAME"],
            Ready: true,
            Timestamp: time.Now().Unix()}
    }
}


func (api *Sigur) processEvents() {
    var events []*Event
    now := time.Now().Unix()
    //eList = append(eList, ev)
    for idx, e := range api.events {
        // event described or timeout expired
        ready := api.describeEvent(e) || now > e.Timestamp + eventMaxWait
        if ready {
            events = append(events, e)
            // remove from list
            last := len(api.events)-1
            api.events[idx] = api.events[last]
            api.events = api.events[:last]
        }
    }
    
    if len(events) > 0 {
        //log.Println(api.GetName(), "SENDING:", events[0])
        //api.Reply(-1, "UpdateDevices", events)
        api.Broadcast("UpdateDevices", events)
    }
}

// extend IDs into text descriptions
func (api *Sigur) describeEvent(e *Event) bool {
    var dev *Device
    var obj *ObjectInfo
    now := time.Now().Unix()    

    if e.ObjectId > 0 {
        // get object name
        if obj = api.objects[e.ObjectId]; obj == nil {
            obj = new(ObjectInfo)
            api.objects[e.ObjectId] = obj
        }
        if obj.Ready || 0 == obj.Timestamp /* just created */ {
            if now < obj.Timestamp + cacheTimeout {
                e.ObjectName = obj.Name
            } else {
                obj.Ready = false
                obj.Timestamp = now // not just created
                cmd := "GETOBJECTINFO OBJECTID " + strconv.Itoa(e.ObjectId)
                api.send(cmd)
            }
        }
    }

    if e.DeviceId > 0 {
        // get device name
        if dev = api.devices[e.DeviceId]; dev == nil {
            dev = new(Device)
            api.Lock()
            api.devices[e.DeviceId] = dev
            api.Unlock()
        }
        if dev.Ready || 0 == dev.Timestamp /* just created */ {
            if now < dev.Timestamp + cacheTimeout {
                e.DeviceName = dev.Name
            } else {
                api.Lock()
                dev.Ready = false
                api.Unlock()
                dev.Timestamp = now // not just created
                cmd := "GETAPINFO " + strconv.Itoa(e.DeviceId)
                api.send(cmd)
            }
        }
    }
    return (e.ObjectId == 0 || obj.Ready) && (e.DeviceId == 0 || dev.Ready)
}


func (api *Sigur) storeEvent(data []string) {
    e := new(Event)
    e.Timestamp = time2num(data[1])
    e.TypeId, _ = strconv.Atoi(data[2])
    if _, ok := evTypes[e.TypeId]; ok == true {
        e.Name = evTypes[e.TypeId]
    } else {
        e.Name = "Неизвестный тип события (код " + data[2] + ")"
    }
    e.DeviceId, _ = strconv.Atoi(data[3])
    e.ObjectId, _ = strconv.Atoi(data[4])
    
    api.describeEvent(e)
    api.events = append(api.events, e)
}


/********************** H E L P E R S ***************************/

func parseInfo(data []string, skip int) map[string]string {
    info := make(map[string]string)
    for i:= skip; i < len(data) - 1; i+=2 {
        info[data[i]] = data[i + 1]
    }
    return info
}
