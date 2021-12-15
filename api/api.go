package api

import (
    "log"
    "time"
    "runtime"
    "strconv"
    "context"
    //"../adapters/configuration"
)

const (
    winStoragePath = "storage/"
    linStoragePath = "/var/lib/s7server/"
)
    

func DescribeEvent(code int64) string {
    return "Код #" + strconv.FormatInt(code, 10)
}

func DescribeClass(code int64) (text string) {
    text = ClassText[code]
    if "" == text {
        text = "Класс #" + strconv.FormatInt(code, 10)
    }
    return
}


// subscribe for messages from Configuration
func NewAPI(s *Settings, broadcast Broadcast, cfg interface{}) *API {
//    s.Status.TCP = "offline"
//    s.Status.DB = "offline"
    return &API{
        name: s.Type + "-" + strconv.FormatInt(s.Id, 10),
        broadcast: broadcast,
        Configuration: cfg,
        Settings: s}
}


// TODO: colorize output => https://twinnation.org/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go
func (api *API) Log(args... interface{}) {
    log.Println(append([]interface{}{api.name}, args...)...)
}

func (api *API) Warn(args... interface{}) {
    log.Println(append([]interface{}{api.name}, args...)...)
}

func (api *API) Err(args... interface{}) {
    log.Println(append([]interface{}{api.name}, args...)...)
}

// expose API
func (api *API) Api(actions map[string] Action) {
    // endpoint is case sensitive!
    //api.actions = make(map[string] Action)
    //api.use("ListDevices", a)
    api.actions = actions
}

func (api *API) GetName() string {
    return api.Settings.Type + "-" + strconv.FormatInt(api.Settings.Id, 10)
}

func (api *API) GetStorage() string {
    var path string
    if runtime.GOOS == "windows" {
        path = winStoragePath
    } else {
        path = linStoragePath
    }
    return path + api.GetName()
}


/*func (api *API) GetTitle() string {
    return api.Settings.Title
}*/

// exec action handler
// one thread per client
func (api *API) Do(cid int64, action string, json []byte) (data interface{}, broadcast bool) {
    defer func() {
        if r := recover(); r != nil {
            api.Err("!!! Action '" + action + "' failed for user #", cid, " - ", r)
            data = "Операция не выполнена (сбой сервера)"
            broadcast = false
        }
    }()    
    if _, ok := api.actions[action]; true == ok {
        return api.actions[action](cid, json)
    } 
    return nil, false
    //log.Println(api.Name, "- unknown action:", action)
    //TODO: return "Unknown action: " + action ?
}

// used to notify clients when event happened (was no any queries from client)
func (api *API) Broadcast(action string, data interface{}) {
    reply := ReplyMessage{Service: api.Settings.Id, Action: action, Task: 0, Data: data}
    api.broadcast(0, &reply)
}


func (api *API) GetSettings() *Settings {
    return api.Settings
}

func (api *API) Cancelled(ctx context.Context) bool {
    select {
        case <- ctx.Done(): return true
        default: return false
    }
}

func (api *API) Sleep(ctx context.Context, delay time.Duration) bool {
    select {
        case <-ctx.Done():
            return false
        case <-time.After(delay):
            return true
    }
}

func (api *API) ReportStartup() {
    api.Broadcast("Events", EventsList{Event{
        Class: EC_SERVICE_STARTED,
        Text: "Сервис запущен",
        Time: time.Now().Unix()}})
}

func (api *API) ReportShutdown() {
    api.Log("Reporting shutdown")
    api.Broadcast("Events", EventsList{Event{
        Class: EC_SERVICE_SHUTDOWN,
        Text: "Сервис остановлен",
        Time: time.Now().Unix()}})
}

func (api *API) SetServiceStatus(tcp, db string) {
    var events EventsList
        moreTCP := " " + api.Settings.Status.TCP + " -> " + tcp
        moreDB := " " + api.Settings.Status.DB + " -> " + db

    api.Settings.Status.Lock()
    if tcp != "" && api.Settings.Status.TCP != tcp {
        var event Event
        if "online" == tcp {
            event.Class = EC_SERVICE_ONLINE
            event.Text = "Соединение установлено" + moreTCP
        } else if "offline" == tcp {
            event.Class = EC_SERVICE_OFFLINE
            event.Text = "Соединение потеряно" + moreTCP
        } else {
            event.Text = "Неопределённое состояние TCP" + moreTCP
        }
        events = append(events, event)

        api.Settings.Status.TCP = tcp
    }
    if db != "" && api.Settings.Status.DB != db {
        var event Event
        if "online" == db {
            event.Class = EC_DATABASE_READY
            event.Text = "БД готова" + moreDB
        } else if "offline" == db {
            event.Class = EC_DATABASE_UNAVAILABLE
            event.Text = "БД недоступна" + moreDB
        } else {
            event.Text = "Неопределённое состояние БД" + moreDB
        }
        events = append(events, event)

        api.Settings.Status.DB = db
    }
    status := api.Settings.Status
    api.Settings.Status.Unlock()
    //api.Log(":::::::::::::: STATUS Events:", events)
    if len(events) > 0 {
        api.Broadcast("Events", events)
        api.Broadcast("StatusUpdate", &status) // TODO: it's legacy?
    }
}

// TODO: make status boolean?
func (api *API) SetTCPStatus(value string) {
    api.SetServiceStatus(value, "")
}

// TODO: make status boolean?
func (api *API) SetDBStatus(value string) {
    api.SetServiceStatus("", value)
}