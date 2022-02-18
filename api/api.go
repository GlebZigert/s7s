package api

import (
    "log"
    "time"
    "strconv"
    "context"
    //"s7server/adapters/configuration"
)

var DataStoragePath string

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
func NewAPI(s *Settings, broadcast Broadcast) *API {
//    s.Status.TCP = "offline"
//    s.Status.DB = "offline"
    return &API{
        name: s.Type + "-" + strconv.FormatInt(s.Id, 10),
        broadcast: broadcast,
        Settings: s}
}


// TODO: colorize output => https://twinnation.org/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go
func (api *API) Log(args... interface{}) {
    log.Println(append([]interface{}{"[" + api.name + "]"}, args...)...)
}

func (api *API) Warn(args... interface{}) {
    log.Println(append([]interface{}{"[" + api.name + "]"}, args...)...)
}

func (api *API) Err(args... interface{}) {
    log.Println(append([]interface{}{"[" + api.name + "]"}, args...)...)
}

// expose API
func (api *API) Api(actions map[string] Action) {
    // endpoint is case sensitive!
    api.Lock()
    api.actions = actions
    api.Unlock()
}

func (api *API) ErrChecker(ctx context.Context, complaints chan error, okCode, errCode int64) {
    timer := time.NewTimer(0) // 1->
    fail := false
    var lastErr string
    for nil == ctx.Err() {
        select {
            case <-ctx.Done():

            case err := <-complaints:
                if !timer.Stop() && len(timer.C) > 0 { // ->1
                    <-timer.C // drain the channel for reuse: https://pkg.go.dev/time#Timer.Stop
                }
                if nil != err  {
                    if lastErr != err.Error() {
                        lastErr = err.Error()
                        api.Err(err)
                    }
                    if !fail {
                        api.SetServiceStatus(errCode)
                    }
                    fail = true
                } else if fail {
                    fail = false
                    timer.Reset(1 * time.Second)
                }

            case <-timer.C:
                api.SetServiceStatus(okCode)
        }
    }
    api.Log("Error checker for", okCode, "<->", errCode, "stopped")
}


func (api *API) GetName() string {
    return api.Settings.Type + "-" + strconv.FormatInt(api.Settings.Id, 10)
}

func (api *API) GetStorage() string {
    return DataStoragePath + api.GetName()
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
    api.RLock()
    do, _ := api.actions[action]
    api.RUnlock()
    if nil != do {
        a, b := do(cid, json)
        if err, ok := a.(error); ok {
            panic(err)
        }
        return a, b
    } 
    return nil, false // TODO: return errors.New(...), false
}

// used to notify clients when event happened (was no any queries from client)
func (api *API) Broadcast(action string, data interface{}) {
    if events, _ := data.(EventsList); len(events) > 0 {
        for i := range events {
            events[i].ServiceId = api.Settings.Id
            events[i].ServiceName = api.Settings.Title
        }
    }
    
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

func (api *API) SetServiceStatus(states ...int64) {
    var events EventsList
    keys := map[string] *int64 {
        "self": &api.Settings.Status.Self,
        "tcp": &api.Settings.Status.TCP,
        "db": &api.Settings.Status.DB}
    //api.Log("S-S:", states)
    for _, sid := range states {
        ptr := keys[serviceStatuses[sid]]
        if nil != ptr {
            api.Settings.Status.Lock()
            if *ptr != sid { // don't duplicate events/states
                *ptr = sid
                events = append(events, Event{Class: sid})
            }
            api.Settings.Status.Unlock()
        } else {
            api.Err("Unknown service status:", sid)
        }
    }
    //api.Log("S-S:", states, events)
    if len(events) > 0 {
        api.Broadcast("Events", events)
    }
}
