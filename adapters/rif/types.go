package rif

import (
	"os"
	"net"
    "sync"
    "time"
    "../../api"
    "../../dblayer"
)

type Reply func (int, string, string, interface{})

type Rif struct {
    sync.RWMutex
    dblayer.DBLayer
    api.API
    xmlLog  *os.File
    
	// use int key only for session, for long-term storage use [type-num1-num2-num3] key
    devices         map[int64] *Device
    idMap           map[int64] int64
    waitReply       map[string] int64 // ["device:command"] = userId
    queryEventsChan chan int64
    complaints      chan error

	//Name    string
    //queue   chan string
    //reply   dispatcher.Reply
    tcpControl  chan struct{}
    
    conn    net.Conn
	//log     *os.File
}

type Device struct {
	Id         int64    `json:"id"`
    AccessMode int64    `json:"accessMode"` // hint for GUI
    Order      int64    `json:"order"` // original id
    Handle     string   `json:"-"`
	Level      int 	    `json:"level"`
	Type       int      `json:"type"`
	Name       string   `json:"name"`
	Num        [3]int   `json:"num"`
    Ip         string   `json:"ip"`
    Ip2        string   `json:"ip"`
    Login      string   `json:"login"`
    Password   string   `json:"password"`
    Option     int 	    `json:"option"`
    Dk         int 	    `json:"dk"`
	States 	   [2]State `json:"states"`
}

type DevList []Device // for filtering

type State struct {
	Id 			int 	    `json:"id"`
	Class       int64       `json:"class"`
    DateTime 	time.Time 	`json:"datetime"`
	Name 		string 	    `json:"name"`
}

//<state id="0" datetime="2019-07-11 10:13:24" name="Неопр. сост."/>
type _State struct {
	Id 			int 	`xml:"id,attr"`
	DateTime 	string 	`xml:"datetime,attr"`
	Name 		string 	`xml:"name,attr"`
}

// <device id="1" level="1" type="0" num1="0" num2="0" num3="0" name="БЛ-IP(253) Шкаф Радиомодем" lat="0.00000000" lon="0.00000000" description="(null)">
type _Device struct {
	Id         int64   `xml:"id,attr"`
	Level      int     `xml:"level,attr"`
	Type       int 	   `xml:"type,attr"`
	Num1       int 	   `xml:"num1,attr"`
	Num2       int 	   `xml:"num2,attr"`
	Num3       int     `xml:"num3,attr"`
	Name       string	`xml:"name,attr"`
    Ip         string   `xml:"ip,attr"`
    Ip2        string   `xml:"ip2,attr"`
    Login      string   `xml:"login,attr"`
    Password   string   `xml:"password,attr"`
    Option     int 	    `xml:"option,attr"`
    Dk         int 	    `xml:"dk,attr"`
	States	[]_State    `xml:"states>state"`
}

type _Event struct {
	Id         int64   `xml:"id,attr"`
	DateTime   string  `xml:"datetime,attr"`
    DeviceName string  `xml:"object,attr"`
	Text       string  `xml:"comment,attr"`
    Reason     string  `xml:"reason,attr"`
    Reaction   string  `xml:"measures,attr"`
    Event      int64   `xml:"type,attr"`
    Type       int     `xml:"objecttype,attr"`
	Num1       int 	   `xml:"d1,attr"`
	Num2       int 	   `xml:"d2,attr"`
	Num3       int     `xml:"d3,attr"`
    Ip         string  `xml:"direction,attr"`
}

type RIFPlusPacket struct {
	Type 	string `xml:"type,attr"`
	Devices []_Device `xml:"devices>device"`
    Events  []_Event `xml:"jours>jour"`
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
        if devices[i].AccessMode > 0 || 0 == devices[i].Type {
             res = append(res, devices[i])
        }
    }
    // TODO: maybe just return res?
    if len(res) > 0 {
        return res
    } else {
        return nil
    }
}