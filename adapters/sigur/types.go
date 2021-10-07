package sigur

import (
	"net"
    "sync"
    "time"
    "../../api"
    "../../dblayer"
    "../configuration"
)


type Sigur struct {
    sync.RWMutex
    api.API
    dblayer.DBLayer
    
    //Settings    *api.Settings
    //devices map[int] Device
    //queue   chan string
    //reply   dispatcher.Reply
    cfg     configuration.ConfigAPI
    conn    net.Conn
    
    devices     map[int] *Device    // access points
    objects     map[int] *ObjectInfo
    
    
    //apInfo      map[int] APInfo

    
    events      []*Event

    //tcpDone     chan    struct{}
}

type Device struct {
    Id          int     `json:"id"`     // field for db.Query
    ParentId    int     `json:"parentId"`
    Type        string  `json:"type"`
    Name        string  `json:"name"`
    StateId     string  `json:"state"`
    StateName   string  `json:"state"`
    //////////////////
    Timestamp   int64
    Ready        bool    
}


type Personal struct {
    Id          int
    ParentId    int
    Type        string
    Name        string 
    CodeKey     []byte
}

type ObjectInfo struct {
    Id          int
    Name        string
    Timestamp   int64
    Ready       bool    
}


// "EVENT_CE" <date-time-spec> <event-type-id> <ap-id> <object-id> <direction-code> <key> 

type Event struct {
    Timestamp 	    int64 	`json:"datetime"`
    TypeId          int     `json:"typeId"`
    Name            string  `json:"name"`
    
    DeviceId        int     `json:"deviceId"`
    DeviceName      string  `json:"deviceName"`
    
    ObjectId        int     `json:"objectId"`
    ObjectName      string  `json:"objectName"`
    
    DirectionCode   int     `json:"directionCode"`
    Key             string  `json:"key"`
}

type AccessRule struct {
    Id          int64
    RuleType    string
    PowerIdx    int64
    Name        string
    Description string
    StartDate   time.Time
    EndDate     time.Time
    NRules      int64
    NSpecRules  int64
    StdWeekMode bool
    
    Rules       [32][60*24]byte   // composite field Rule0...Rule31
    SpecRules   [32][60*24]byte
    SpecDates   [32]time.Time
/*    Rule0       byte
    Rule1       byte
    Rule2       byte
    Rule3       byte
    Rule4       byte
    Rule5       byte
    Rule6       byte
    Rule7       byte
    Rule8       byte
    Rule9       byte
    Rule10      byte
    Rule11      byte
    Rule12      byte
    Rule13      byte
    Rule14      byte
    Rule15      byte
    Rule16      byte
    Rule17      byte
    Rule18      byte
    Rule19      byte
    Rule20      byte
    Rule21      byte
    Rule22      byte
    Rule23      byte
    Rule24      byte
    Rule25      byte
    Rule26      byte
    Rule27      byte
    Rule28      byte
    Rule29      byte
    Rule30      byte
    Rule31      byte*/
}