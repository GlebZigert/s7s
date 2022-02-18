package dispatcher

import (
    "sync"
    "net/http"
    "context"
//    "database/sql"
    "encoding/json"
    "golang.org/x/net/websocket"
    
    "s7server/api"
    "s7server/adapters/configuration"
)

//type Reply func (int, int, string, int, interface{})
type Broadcast func (*websocket.Conn, interface{})
type Subscribe func () chan interface{}

type Factory func (*api.API) Service
//type Do func (int, string, string)

type HTTPAPI interface {
    HTTPHandler(http.ResponseWriter, *http.Request) error
}

type ManageableZones interface {
    ZoneCommand(userId, zoneCommand int64, devices []int64)
}

type Service interface {
    GetName()       string
    GetSettings()   *api.Settings
    GetList()       []int64
    Do(int64, string, []byte) (interface{}, bool)
    Run(configuration.ConfigAPI) error
    Shutdown()
}

type Client struct {
	ws *websocket.Conn
	//token	string
    role    int64
}

type Error struct {
    Code    int64   `json:"code"`
    Error   string  `json:"error"`
}

type Credentials struct {
    Login   string   `json:"login"`
    Token   string   `json:"token"`
}

type ZoneCommand struct {
    ZoneId      int64   `json:"zoneId"`
    Command     int64   `json:"command"`
}

type Dispatcher struct {
    // TODO: mutex required?
    sync.RWMutex
    ctx             context.Context
    cfg             configuration.ConfigAPI
	services		map[int64] Service
	clients			map[int64] Client
    queue           chan string
	clientsCount	int
	nextClient		int
    //db              *sql.DB
    factory         Factory
}

type Query struct {
	Service int64
    Action  string
    Task    int
    Data json.RawMessage // delay parsing (https://golang.org/pkg/encoding/json/#RawMessage)
}
