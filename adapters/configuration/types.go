package configuration

import (
    "time"
    "sync"
//    "net/http"
    
    "s7server/api"
//    "s7server/dblayer"
)

type Device struct {
    Id          int64       `json:"id"`
    ParentId    int64       `json:"parentId"`
    Type        int64       `json:"type"`
    LastSeen    time.Time   `json:"-"`
    //ServiceId   int64       `json:"serviceId"`
    Handle      string      `json:"-"`
    Name        string      `json:"name"`
    Data        string      `json:"-"`
}


// each link is a combination of scope_id (e.g. service_id) device_id (or rule_id or ...) and flags
// "device-zone" => ACS in/out
// "zone-device" => devices in zone
type ExtLink   [3]int64 // [scopeId, deviceId, flags]

type User struct {
    Id          int64       `json:"id"`
    ParentId    int64       `json:"parentId"`
    Type        int         `json:"type"`
    Role        int         `json:"role"`
    Archived    bool        `json:"-"`
    
    Name        string      `json:"name,omitempty"`
    Surename    string      `json:"surename,omitempty"`
    MiddleName  string      `json:"middleName,omitempty"`
    Rank        string      `json:"rank,omitempty"`
    Organization string     `json:"organization,omitempty"`
    Position    string      `json:"position,omitempty"`
    
    Login       string      `json:"login,omitempty"`
    Password    string      `json:"password"`
    //NewPassword string      `json:"newPassword"` // input from extrenal form
    Cards       []string    `json:"cards"`
    
    Zones       []ExtLink   `json:"zones"`      //  [0, zone_id, rule_id]
    Devices     []ExtLink   `json:"devices"`    // [] scope_id : device_id
    // frontend error reportion
    Warnings    []string    `json:"warnings"`    // warning messages
    Errors      []string    `json:"errors"`    // warning messages
}

// user summary for export to another services
type UserSummary struct {
    UserId      int64
    Cards       []string
    Rules       []int64
    Devices     []int64
}

type Zone struct {
    Id              int64       `json:"id"`
    Name            string      `json:"name"`
    MaxVisitors     int         `json:"maxVisitors"`
    
    // not stored in db
    Devices         []ExtLink   `json:"devices"`    // [] scope_id : device_id
    EntranceEvents  []api.Event `json:"entranceEvents"`
}

type Rule struct {
    Id          int64       `json:"id"`
    Name        string      `json:"name"`
    Description string      `json:"description"`
    StartDate   time.Time   `json:"startDate"`
    EndDate     time.Time   `json:"endDate"`
    Priority    int         `json:"priority"`
    TimeRanges  []TimeRange `json:"timeRanges"`
}

type TimeRange struct {
    RuleId      int64       `json:"-"`
    Direction   int         `json:"direction"`
    From        time.Time   `json:"from"`
    To          time.Time   `json:"to"`
}

type Map struct {
    Id          int64       `json:"id"`
    Type        string      `json:"type"`
    Name        string      `json:"name"`
    CX          float32     `json:"cx"`
    CY          float32     `json:"cy"`
    Zoom        float32     `json:"zoom"`
    Shapes      []Shape     `json:"shapes"`
    //Picture   []byte
}

type Shape struct {
    Id          int64       `json:"id"`
    MapId       int64       `json:"mapId"`
    ServiceId   int64       `json:"sid"` // service id
    DeviceId    int64       `json:"did"` // device id
    Type        string      `json:"type"`
    X           float32     `json:"x"`
    Y           float32     `json:"y"`
    Z           float32     `json:"z"` // z-order
    W           float32     `json:"w"` // width
    H           float32     `json:"h"` // height
    R           float32     `json:"r"` // rotation
    Data        string      `json:"data"` // points, text, icon, etc...
}

type VisitorLocation struct {
    UserId      int64   `json:"userId"`
    ParentId    int64   `json:"userId"`
    ZoneId      int64   `json:"zoneId"`
}

type Configuration struct {
    //sync.RWMutex
    //dblayer.DBLayer
    api.API
    
    complaints      chan error
    cache RelationsCache
    //reply   dispatcher.Reply
    //db              *sql.DB
}

type RelationsCache struct {
    sync.RWMutex
    //ruleLinks map[int64] []UserLink // for groups only
    //devLinks map[int64] []UserLink // for groups only
    children map[int64] []int64
    parents map[int64] []int64
}


type Filterable interface {
    GetList() []int64
    Filter(list map[int64]int64) interface{}
}

//////////////////////////////////////////////////////////////////////
//                    ZONES filtering                               //
//////////////////////////////////////////////////////////////////////

func (z Zone) GetList() []int64 {
    list := make([]int64, 0, len(z.Devices))
    for i := range z.Devices {
        list = append(list, z.Devices[i][1])
    }
    return list
}

func (z *Zone) Filter(filter map[int64]int64) interface{} {
    devices := z.Devices
    z.Devices = make([]ExtLink, 0, len(z.Devices))
    for i := range devices {
        // filter[0] > 0 => all id are acceptable
        if filter[0] > 0 || filter[devices[i][1]] > 0 {
             z.Devices = append(z.Devices, devices[i])
        }
    }
    return z
}

/*******************************************************************************/

type ZoneList []Zone

func (zones ZoneList) GetList() []int64 {
    list := make([]int64, 0, len(zones))
    
    for _, z := range zones {
        list = append(list, z.GetList()...)
    }

    return list
}

func (zones ZoneList) Filter(filter map[int64]int64) interface{} {
    var res ZoneList
    for _, z := range zones {
        z.Filter(filter)
        if filter[0] > 0 || len(z.Devices) > 0 {
            res = append(res, z)
        }
    }
    return res
}

//////////////////////////////////////////////////////////////////////
//                    MAPS filtering                               //
//////////////////////////////////////////////////////////////////////


func (m Map) GetList() []int64 {
    list := make([]int64, 0, len(m.Shapes))
    for i := range m.Shapes {
        list = append(list, m.Shapes[i].DeviceId)
    }
    return list
}

func (m *Map) Filter(filter map[int64]int64) interface{} {
    shapes := m.Shapes
    m.Shapes = make([]Shape, 0, len(m.Shapes))
    for i := range shapes {
        // filter[0] > 0 => all id are acceptable
        if filter[0] > 0 || filter[shapes[i].DeviceId] > 0 {
             m.Shapes = append(m.Shapes, shapes[i])
        }
    }
    return m
}

/*********************************************************************************/

type MapList []Map

func (maps MapList) GetList() []int64 {
    list := make([]int64, 0, len(maps))
    
    for _, m := range maps {
        list = append(list, m.GetList()...)
    }

    return list
}

func (maps MapList) Filter(filter map[int64]int64) interface{} {
    var res MapList
    for _, m := range maps {
        m.Filter(filter)
        if filter[0] > 0 || len(m.Shapes) > 0 {
            res = append(res, m)
        }
    }
    return res
}


//////////////////////////////////////////////////////////////////////
//                    ConfigAPI                                     //
//////////////////////////////////////////////////////////////////////


type ConfigAPI interface {
    Get()           []*api.Settings
    //Subscribe()                     chan interface{}
    //Unsubscribe(chan interface{})
    
    Authenticate(string, string)  (userId, role int64, err error)
    Authorize(userId int64, devices []int64) (map[int64]int64, error)
    
    // automatic actions (algorithms)
    //CheckEvent(event *api.Event) []Algorithm
    //ResetAlarm(serviceId, deviceId int64)
    ProcessEvents(event api.EventsList) error
    ImportEvents([]api.Event) error
    GetLastEvent(serviceId int64) (*api.Event, error)

    GlobalDeviceId(systemId int64, handle, name string) (id int64, err error)
    SaveDevice(serviceId int64, device *Device, data interface{}) (err error)
    DeleteDevice(id int64) error
    LoadDevices(serviceId int64) ([]Device, error)
    TouchDevice(serviceId int64, dev *Device) error
    
    LoadLinks(sourceId int64, link string) (list []ExtLink, err error)
    SaveLinks(sourceId int64, linkType string, list []ExtLink) error
    
    // ACS
    //GetAccessRules(serviceId int64) (rules []*Rule)
    //GetActiveCards(serviceId, deviceId int64)
    UserByCard(card string) (int64, error)
    RequestPassage(zoneId int64, card, pin string) (userId, errCode int64)
    //GetCards(deviceId int64) map[string][]*Rule
    //SameZoneDevices(deviceId int64) []int64
    EnterZone(event api.Event)
    //UsersWithLinks(int64)           []*User
    
    StartNewShift(userId int64) error
    CompleteShift(userId int64) error
    GetUser(id int64) (*User, error)
    GetUser_for_Axxon(id int64) *User
}

type EventFilter struct {
    Start       time.Time   `json:"start"`
    End         time.Time   `json:"end"`
    ServiceId   int64       `json:"serviceId"`
    UserId      int64       `json:"userId"`
    Limit       int64       `json:"limit"`
    Class       int         `json:"class"`
}

/*func (info *EnterZone) Filter (list map[int64]int64) interface{} {}*/


/*
type JSONRules struct {
     struct {
        Date        JSONDate        `json:"date"`
        DayNumber   int
        RegularDays []TimeRange `json:"specialDays"`
    }
}*/

///////////////////////////// SUPPLY TYPES //////////////////////////////
// http://choly.ca/post/go-json-marshalling/
// https://stackoverflow.com/questions/45303326/how-to-parse-non-standard-time-format-from-json
/*
type JSONDate time.Time

func (j *JSONDate) UnmarshalJSON(b []byte) error {
    s := strings.Trim(string(b), "\"")
    t, err := time.Parse("02.01.2006", s)
    if err != nil {
        return err
    }
    *j = JSONDate(t)
    return nil
}
    
func (j JSONDate) MarshalJSON() ([]byte, error) {
    return json.Marshal(j)
}
*/
