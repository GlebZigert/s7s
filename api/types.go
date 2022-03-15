package api

import (
    //"time"
    "sync"
    "context"
    //"golang.org/x/net/websocket"
)

type Action func (cid int64, json []byte) (interface{}, bool)

/*type ServiceStatus struct {
    TCP         string
    Database    string
}*/

type Broadcast func (int64, *ReplyMessage)

type ReplyMessage struct {
    UserId  int64       `json:"-"`
    Service int64       `json:"service"`
    Action  string      `json:"action"`
    Task    int         `json:"task"`
    Data    interface{} `json:"data"`
}

type ErrorData struct {
    ErrCode     int64       `json:"errCode"`
    ErrText     string      `json:"errText"`
}

type Algorithm struct {
    Id              int64   `json:"id"`
    Name            string  `json:"name"`
    ServiceId       int64   `json:"serviceId"`
    DeviceId        int64   `json:"deviceId"`
    ZoneId          int64   `json:"zoneId"`
    UserId          int64   `json:"userId"`
    FromState       int64   `json:"fromState"`
    Event           int64   `json:"event"`
    TargetServiceId int64   `json:"targetServiceId"`
    TargetDeviceId  int64   `json:"targetDeviceId"`
    TargetZoneId    int64   `json:"targetZoneId"`
    Command         int64   `json:"command"`
    Argument        int64   `json:"argument"`
    Extra           string  `json:"conditions"` // JSON-serialized extra conditions and other data
}

type Event struct {
    Id          int64       `json:"id"`
    ExternalId  int64       `json:"externalId"`
    FromState   int64       `json:"fromState"`
    Event       int64       `json:"event"`
    Class       int64       `json:"class"`
    Data        string      `json:"data"`
    Text        string      `json:"text"`
    ServiceId   int64       `json:"serviceId"`
    DeviceId    int64       `json:"deviceId"`
    UserId      int64       `json:"userId"`
    ZoneId      int64       `json:"zoneId"`

    Reason      string      `json:"reason"`
    Reaction    string      `json:"reaction"`
    
    // applied commands (according to automatic algorithms)
    Commands    string      `json:"commands"`// [serviceId, deviceId, command, argument]

    // https://github.com/mattn/go-sqlite3/issues/190#issuecomment-343341834
    Time        int64       `json:"time"`

    // don't stored in DB, used for JSON
    ServiceName     string      `json:"serviceName"`
    DeviceName      string      `json:"deviceName"`
    UserName        string      `json:"userName"`
    ZoneName        string      `json:"zoneName"`
    
    Algorithms      []Algorithm `json:"-"`
    RelatedDevices  []int64     `json:"-"` // for event filtering
}

type EventsList []Event // for filtering

type Command struct {
	DeviceId 	int64 	`json:"deviceId"`
	Command 	int64 	`json:"command"`
    Argument 	int64 	`json:"argument"`
}

type API struct {
    sync.RWMutex

    Settings    *Settings
    broadcast   Broadcast
    name        string  // for logging purposes

    Cancel      context.CancelFunc
    Stopped     chan struct{}
    
    actions     map[string] Action
    //tasks       map[int][int]
    //Status              ServiceStatus
}

type Settings struct {
//    Name       string   `json:"name"` // type-id

    Id              int64    `json:"id"`
    Type            string   `json:"type"`
    Title           string   `json:"title"`
    Host            string   `json:"host"`   // host:port/url
	//Port       int
    //URL        string
    Login           string   `json:"login"`
    Password        string   `json:"-"`
    NewPassword     string   `json:"password"` // input from external form
    KeepAlive       int      `json:"keepAlive"`
    
    DBHost          string   `json:"dbHost"`   // host:port
	//DBPort     int
    DBName          string   `json:"dbName"`
    DBLogin         string   `json:"dbLogin"`
    DBPassword      string   `json:"-"`
    NewDBPassword   string   `json:"newDBPassword"` // input from external form 
    
    Status     struct {
        sync.RWMutex
        Self    int64   `json:"self"`   // Internal service status
        TCP     int64   `json:"tcp"`    // External TCP service connection
        DB      int64   `json:"db"`     // External DB status
    }    `json:"status"`
}

//type ServicesList []Settings // for filtering

/*func (services ServicesList) GetList() []int64 {
    return nil
}

func (services ServicesList) Filter (list map[int64]int64) interface{} {
    var res ServicesList
    for i := range services {
        // list[0] > 0 => whole service accessible
        if list[0] > 0 || list[services[i].Id] > 0 {
            res = append(res, services[i])
        }
    }
    return res
}*/

func (events EventsList) GetList () []int64 {
    list := make([]int64, 0, len(events))
    
    for _, ev := range events {
        list = append(list, ev.DeviceId)
    }

    return list
}

func (events EventsList) Filter (userId int64, devFilter map[int64]int64, classFilter map[int64] struct{}) interface{} {
    var res EventsList
    for i := range events {
        // list[0] > 0 => whole service accessible
        if devFilter[0] > 0 || userId == events[i].UserId || devFilter[events[i].DeviceId] > 0 {
            res = append(res, events[i])
        } else {
            if _, ok := classFilter[events[i].Class]; ok {
                res = append(res, events[i])
            }
            /*if 0 == events[i].ServiceId {
                res = append(res, events[i])
            }*/
            /*for _, ec := range classFilter {
                if ec == events[i].Class {
                    res = append(res, events[i])
                    break
                }
            }*/
        }
    }
    if len(res) > 0 {
        return res
    } else {
        return nil
    }
}

/*func (events EventsList) RelatedAP () (list map[int64]int64) {
    //list = make(map[int64]int64)
    for i := range events {
        if events[i].ZoneId > 0 {
            list = list[events[i].DeviceId] | AM_RELATED_AP
        }
    }
    return
}*/
