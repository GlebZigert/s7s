package z5rweb

import (
    "os"
    "sync"
    "time"
    "../../api"
    "../../dblayer"
    "../configuration"
)

//type Reply func (int, string, string, interface{})

type Z5RWeb struct {
    nextMessageId   int64 // should be 64-bit aligned for i386
    
    sync.RWMutex
    api.API
    dblayer.DBLayer
    httpLog  *os.File
    
    complaints      chan error
    
    subscription    chan interface{}
    devices         map[int64] *Device
    idCache         map[string] int64
    lastUsers       map[int64] int64 // [deviceId] => userId
    jobQueue        map[int64] []string
    ignoreEvents    map[string] int64 // [devId-reader] => code
    commandAuthors  map[string] int64 // [devId-reader] => code
    lastCards       map[string] LastCard // [devId-reader] => card
    wrongPinTimes   map[string] []time.Time // [card] => [time, time, ...]
}

type LastCard struct {
    Card    string
    Time    time.Time}

type Card struct { // DB
    Card         string
    Flags        int64
    Timezone     string
}

type Timezone struct { // DB
    Zone     int64
    Begin    string
    End      string
    Days     string
}


/*type DeviceData struct {
    InternalZone    int64       `json:"internalZone"`
    ExternalZone    int64       `json:"externalZone"`
}*/

type Device struct {
    configuration.Device
    //DeviceData

    Online          bool            `json:"-"`
    States          [2]api.Event    `json:"states"` // [1] "color" state, [0] - info

    AccessMode      int64       `json:"accessMode"` // hint for GUI (user permissions)
    SerialNumber    int64       `json:"serialNumber"`
    Active          int         `json:"active"`
    Mode            int64       `json:"mode"`
    IP              string      `json:"ip"`
    Hardware        string      `json:"hardware"`
    Firmware        string      `json:"firmware"`
    ConnFirmware    string      `json:"connFirmware"`
    Protocol        string      `json:"protocol"`
    Zones           []configuration.ExtLink `json:"zones"`
}

type DevList []Device // for filtering

type Parcel struct {
    Type        string      `json:"type"`
    SN 		    int64 	    `json:"sn"`
    Messages    []Message   `json:"messages"`
}

type Message struct {
    Id              int64     `json:"id"`
    Operation       string  `json:"operation"`
    Firmware        string  `json:"fw"`
    ConnFirmware    string  `json:"conn_fw"`
    Active          int     `json:"active"`
    Mode            int64   `json:"mode"`
    IP              string  `json:"controller_ip"`
    Protocol        string  `json:"reader_protocol"`
    //
    Events          []Event `json:"events"`
    //
    Success         int     `json:"success"`
    // for check_access
    Card            string  `json:"card"`
    Reader          int64   `json:"reader"`
    
}

type Event struct {
    Id          int64   `json:"id"`
    DeviceId    int64   `json:"device_id"`
    Event       int64   `json:"event"`
    Card        string  `json:"card"`
    Time        string  `json:"time"`
    Flag        int     `json:"flag"`
}

type EventReply struct {
    Id              int64     `json:"id"`
    Operation       string  `json:"operation"`
    EventsSuccess   int     `json:"events_success"`
}

type CommandsList struct {
    Date        string          `json:"date"`
    Interval    int 	        `json:"interval"`
    Messages    []interface{}   `json:"messages"`
}

/////////////////////////////////////////////////////////
//////////////////// C O M M A N D S ////////////////////
/////////////////////////////////////////////////////////

type CheckAccessReply struct {
    Id          int64     `json:"id"`
    Operation   string  `json:"operation"`
    Granted     int     `json:"granted"`
}

type SetTimezoneCmd struct {
    Id          int64     `json:"id"`
    Operation   string  `json:"operation"`
    Zone        int     `json:"zone"`
    Begin       string  `json:"begin"`
    End         string  `json:"end"`
    Days        string  `json:"days"`
}

type SetActiveCmd struct {
    Id          int64     `json:"id"`
    Operation   string  `json:"operation"`
    Active      int     `json:"active"`
    Online      int     `json:"online"`
}

type ClearCardsCmd struct {
    Id          int64         `json:"id"`
    Operation   string      `json:"operation"`
}

type AddCardsCmd struct {
    Id          int64         `json:"id"`
    Operation   string      `json:"operation"`
    Cards       []OneCard   `json:"cards"`
}

type OneCard struct {
    Card    string  `json:"card"`
    Flags   int     `json:"flags"`
    TZ      int     `json:"tz"`
}

func (devices DevList) GetList() []int64 {
    ids := make([]int64, len(devices))
    for i := range devices {
        ids[i] = devices[i].Id
    }
    return ids
}

func (devices DevList) Filter (list map[int64]int64) interface{} {
    var res DevList
    for i := range devices {
        // list[0] > 0 => whole service accessible
        devices[i].AccessMode = list[0]
        if 0 == devices[i].AccessMode {
            devices[i].AccessMode = list[devices[i].Id]
        }
        if devices[i].AccessMode > 0 {
            res = append(res, devices[i])
        }
    }
    return res
}

/*
    {"type":"Z5RWEB",
     "sn":49971,
     "messages":[                   {"id":719346228,"operation":"power_on","fw":"3.28","conn_fw":"1.0.128","active":0,"mode":0,"controller_ip":"192.168.0.79","reader_protocol":"wiegand"}
     ]}
*/

