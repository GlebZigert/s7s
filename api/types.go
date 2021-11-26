package api

import (
    //"time"
    "sync"
    "context"
    //"golang.org/x/net/websocket"
)

var ARMFilter = map[int64] []int64 { // [armType] => event class to catch if no suitable device/user in event
    //1: nil, // all events
    EC_GLOBAL_ALARM: {ARM_UNIT, ARM_CHECKPOINT, ARM_GUARD, ARM_OPERATOR, ARM_SECRET/*, ARM_BUREAU*/},
    EC_ENTER_ZONE: {ARM_SECRET},
    EC_ACCESS_VIOLATION: {ARM_UNIT, ARM_CHECKPOINT, ARM_GUARD},
    EC_ACCESS_VIOLATION_ENDED: {ARM_UNIT, ARM_CHECKPOINT, ARM_GUARD}}
    
// access control RequestPassage codes
const (
    ACS_ACCESS_GRANTED = 0
    ACS_UNKNOWN_CARD = 1
    ACS_ANTIPASSBACK = 2
    ACS_ACCESS_DENIED = 3
    ACS_PIN_REQUIRED = 4
    ACS_WRONG_PIN = 5
    ACS_MAX_VISITORS = 6
)

// User Types
const (
    UT_GROUP = 1
    UT_PERSONAL = 2
    UT_GUEST = 3
    UT_CAR = 4
)

// User ARM
const (
    ARM_ADMIN       = 1 // админ
    ARM_UNIT        = 2 // начальник ВЧ
    ARM_CHECKPOINT  = 3 // начальник КПП
    ARM_GUARD       = 4 // начальник караула
    ARM_OPERATOR    = 5 // оператор
    ARM_SECRET      = 6 // гостайна
    ARM_BUREAU      = 7 // бюро пропусков
)

// Access Modes
const (
    AM_WATCH = 1
    AM_CONTROL = 2
    AM_RELATED_AP = 4 // related access point (same zone)
)

var ClassText = map[int64] string {
    EC_CONNECTION_OK: "Связь установлена",
    EC_CONNECTION_LOST: "Связь отсутствует",
    EC_GLOBAL_ALARM: "Общая тревога",
    EC_INFO_ALARM_RESET: "Сброс тревог",
    EC_USER_LOGGED_IN: "Пользователь подключился",
    EC_ALREADY_LOGGED_IN: "Пользователь уже подключен",
    EC_LOGIN_FAILED: "Ошибка аутентификации",
    EC_USER_LOGGED_OUT: "Пользователь отключился",
    EC_ARM_TYPE_MISMATCH: "Смена типа АРМ недопустима",
    EC_LOGIN_TIMEOUT: "Реквизиты доступа не получены вовремя",
    EC_USERS_LIMIT_EXCEED: "Превышено максимальное число пользователей",
    EC_USER_SHIFT_STARTED: "Начало новой смены",
    EC_USER_SHIFT_COMPLETED: "Смена завершена",
    EC_ACCESS_VIOLATION: "Нарушение режима доступа в зону",
    EC_ACCESS_VIOLATION_ENDED: "Прекращено нарушение режима доступа в зону",
    EC_ONLINE: "Связь установлена",
    EC_LOST: "Связь отсутствует",
    EC_ARMED: "Поставлено на охрану",
    EC_DISARMED: "Снято с охраны",
    EC_POINT_BLOCKED: "Проход запрещён",
    EC_FREE_PASS: "Свободный проход",
    EC_NORMAL_ACCESS: "Штатный доступ",
    EC_ALGO_STARTED: "Алгоритм запущен",
    EC_UPS_PLUGGED: "Питание от сети",
    EC_UPS_UNPLUGGED: "Питание от батарей"}


// event classes
// event classes may turn into universal codes in the future
const (
    EC_NA = 0 //iota
    // INFO
    EC_INFO                 = 100
    EC_ENTER_ZONE           = 101
    EC_EXIT_ZONE            = 102        // virtual code
    EC_INFO_ALARM_RESET     = 103
    EC_USER_LOGGED_IN       = 104
    EC_USER_LOGGED_OUT      = 105
    EC_ARM_TYPE_MISMATCH    = 106
    EC_LOGIN_TIMEOUT        = 107
    EC_USER_SHIFT_STARTED   = 108
    EC_USER_SHIFT_COMPLETED = 109
    EC_SERVICE_STARTED      = 110
    EC_SERVICE_SHUTDOWN     = 111
    EC_ARMED                = 112
    EC_DISARMED             = 113
    EC_POINT_BLOCKED        = 114
    EC_FREE_PASS            = 115
    EC_NORMAL_ACCESS        = 116
    EC_ALGO_STARTED         = 117
    
    // OK
    EC_OK                     = 200
    EC_ACCESS_VIOLATION_ENDED = 201
    EC_CONNECTION_OK          = 202
    EC_SERVICE_ONLINE         = 203
    EC_DATABASE_READY         = 204
    EC_ONLINE                 = 205
    EC_UPS_PLUGGED            = 206
    
    // ERROR
    EC_ERROR                = 300
    EC_USERS_LIMIT_EXCEED   = 301
    
    
    // LOST (no link)
    EC_LOST                 = 400
    EC_CONNECTION_LOST      = 401
    EC_SERVICE_OFFLINE      = 402
    EC_DATABASE_UNAVAILABLE = 403
    
    
    // ALARM
    EC_ALARM                = 500
    EC_GLOBAL_ALARM         = 501
    EC_ACCESS_VIOLATION     = 502
    EC_ALREADY_LOGGED_IN    = 503
    EC_LOGIN_FAILED         = 504
    EC_UPS_UNPLUGGED        = 505
    //EC_PICKING_PIN_DETECTED = 503
)

var EventClasses = []int64 {
    EC_NA,
    EC_INFO,
    EC_OK,
    EC_ERROR,
    EC_LOST,
    EC_ALARM}


type Action func (cid int64, json []byte) (interface{}, bool)

/*type ServiceStatus struct {
    TCP         string
    Database    string
}*/

type Broadcast func (int64, *ReplyMessage)

type ReplyMessage struct {
	Service int64       `json:"service"`
    Action  string      `json:"action"`
    Task    int         `json:"task"`
    Data    interface{} `json:"data"`
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
    Settings    *Settings
    broadcast   Broadcast
    name        string  // for logging purposes

    Cancel      context.CancelFunc
    
    // using empty type for Config to avoid extra package with shared data types
    // and access type names without package.* prefixes (e.g. User, not package.User)
    // in configuration package
    Configuration      interface{}
    
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
    NewPassword     string  `json:"password"` // input from extrenal form 
    KeepAlive       int      `json:"keepAlive"`
    
    DBHost          string   `json:"dbHost"`   // host:port
	//DBPort     int
    DBName          string   `json:"dbName"`
    DBLogin         string   `json:"dbLogin"`
    DBPassword      string   `json:"-"`
    NewDBPassword   string   `json:"newDBPassword"` // input from extrenal form 
    
    Status     struct {
        sync.RWMutex
        TCP     string   `json:"tcp"`
        DB      string   `json:"db"`
    }    `json:"status"`
}

//type ServicesList []Settings // for filtering


type Task struct {
    ClientId    int
    TaskId      int
}


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
