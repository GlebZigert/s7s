package dispatcher

import (
    "io"
	"log"
	"fmt"
	"time"
//    "io/ioutil"
    "strings"
    "strconv"
    "encoding/json"
//    "encoding/base64"
    "net/http"
	"golang.org/x/net/websocket"
)

import (
    "../api"    
	"../adapters/rif"
    //"../adapters/sigur"
    "../adapters/axxon"
    "../adapters/z5rweb"
    "../adapters/ipmon"
    "../adapters/parus"
    "../adapters/configuration"
)

const (
    maxClients = 100
    loginTimeout = 3 // seconds
    keepAliveInterval = 10 + 2 //seconds
)

//var config *[]api.Settings

var armFilter map[int64] map[int64] struct{}

func factory(api *api.API) Adapter {
    var service Adapter
    switch (*api).Settings.Type {
        case "configuration":
            service = &configuration.Configuration{API: *api}
        case "rif":
            service = &rif.Rif{API: *api}
        /*case "sigur":
            service = &sigur.Sigur{API: *api}*/
        case "axxon":
            service = &axxon.Axxon{API: *api}
        case "z5rweb":
            service = &z5rweb.Z5RWeb{API: *api}
        case "ipmon":
            service = &ipmon.IPMon{API: *api}
        case "parus":
            service = &parus.Parus{API: *api}
    }
    return service
}


func seedFilter() {
    armFilter = make(map[int64] map[int64] struct{})
    // [role][class]
    for class, _ := range api.ARMFilter {
        for _, role := range api.ARMFilter[class] {
            if _, ok := armFilter[role]; !ok {
                armFilter[role] = make(map[int64] struct{})
            }
            armFilter[role][class] = struct{}{}
        }
    }
}

func New() Dispatcher {
    seedFilter()
	var d = Dispatcher{}
	d.services = make(map[int64] Service)
	d.clients = make(map[int64] Client)
    d.queue = make(chan string, 10)
    //d.factory = factory
    //
    // TODO: store cfg in dispatcher for fufutre usage
    cfg := factory(api.NewAPI(&api.Settings{Id: 0, Type: "configuration"}, d.broadcast, nil))
    d.useService(cfg)
    d.cfg = cfg.(configuration.ConfigAPI)
    settings := d.cfg.Get()
    for _, s := range settings {
        service := factory(api.NewAPI(s, d.broadcast, d.cfg))
        if nil == service {
            log.Println("[Dispatcher] Unknown service type:", s)
        } else {
            d.useService(service)
        }
    }
    log.Println("Dispatcher startup completed")
    return d
}

func (dispatcher *Dispatcher) useService(adapter Adapter) {
    settings := adapter.GetSettings()
    id := settings.Id
    dispatcher.Lock()
    dispatcher.services[id] = adapter
    dispatcher.Unlock()
    if 0 == id {
        adapter.Run()
    } else {
        go adapter.Run() // only config should run async
    }
    log.Println("UseService: service started", adapter.GetName())
}

func (dispatcher *Dispatcher) HTTPHandler(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) != 3 {
        http.NotFound(w, r)
        return
    }
    
    id, err := strconv.Atoi(parts[1])
    if err != nil {
        http.NotFound(w, r)
        return
    }
    
    dispatcher.RLock()
    service, ok := dispatcher.services[int64(id)]
    dispatcher.RUnlock()
    
    if ok {
        svc, ok := service.(HTTPAPI)
        if (ok) {
            //log.Println("[Dispatcher] HTTP", r.Method, "path:", r.URL.Path)
            svc.HTTPHandler(w, r)
            return
        }
    }
    http.NotFound(w, r)
}

func (dispatcher *Dispatcher) loggedIn(userId int64) (really bool) {
    _, really = dispatcher.clients[userId]
    return
}


func (dispatcher *Dispatcher) SocketServer(ws *websocket.Conn) {
    defer time.Sleep(100 * time.Millisecond)
    var cred Credentials
    if nil == dispatcher.cfg {
        log.Println("not ready, try later")
        return // not ready, try later
    }
    
    log.Println("New client")
    
    ws.SetReadDeadline(time.Now().Add(loginTimeout * time.Second))
    err := websocket.JSON.Receive(ws, &cred)
    if nil != err {
        log.Println("Failed to login:", err)
        dispatcher.loginError(ws, api.EC_LOGIN_TIMEOUT)
        return
    }
    ws.SetReadDeadline(time.Time{}) // reset deadline

    reply := api.ReplyMessage{Service: 0, Action: "ChangeUser", Task: 0}
    user, errClass := dispatcher.changeUser(0, ws, &cred)

    if 0 == errClass {
        reply.Data = user
    } else {
        reply.Data = Error{errClass, api.DescribeClass(errClass)}
        
    }
    websocket.JSON.Send(ws, reply) 
    if 0 == errClass {
        dispatcher.serveClient(user.Id, ws)
        //go heartbeat(ws)
    }
}


func (dispatcher *Dispatcher) loginError(ws *websocket.Conn, class int64) {
    ev := api.Event{Event: class, Class: api.EC_ERROR}
    dispatcher.cfg.ProcessEvent(&ev)
    payload := fmt.Sprintf(`{"service": 0, "action": "Error", "task": 0, "data": {"class": %d, "text": "%s"}}`, class, ev.Text)
    //log.Println("ERR PAYLOAD:", payload)
    websocket.Message.Send(ws, payload) 
}

func (dispatcher *Dispatcher) serveClient(userId int64, ws *websocket.Conn) {
    defer func () {
        dispatcher.Lock()
        delete(dispatcher.clients, userId)
        dispatcher.Unlock()
        log.Println("Stop serving #", userId)
        dispatcher.cfg.ProcessEvent(&api.Event{
            Class: api.EC_USER_LOGGED_OUT,
            UserId: userId})
    }()
    log.Println("Serving #", userId)

    //var msg string
    var q Query
	var err error
	
    for {
		//if err = websocket.Message.Receive(dispatcher.clients[cid].ws, &msg); err != nil {
        ws.SetReadDeadline(time.Now().Add(keepAliveInterval * time.Second))
        if err = websocket.JSON.Receive(ws, &q); err != nil {
            if io.EOF == err {
                log.Println(fmt.Sprintf("Connection for user #%d closed: %s", userId, err.Error()))
                break
            } else {
                log.Println(err.Error())
                ws.Close()
                break;
                //websocket.Message.Send(ws, err.Error())
            }
        } else if q.Service == 0 && q.Action == "KeepAlive" {
            //log.Println("KeepAlive", userId)
        } else {
            log.Println("Client", userId, "=>", q.Service, q.Action + "([", len(q.Data), "]byte)")
            // userId may be changed (TODO: ws is the same?)
            if res := dispatcher.preprocessQuery(&userId, ws, q); res != nil {
                reply := api.ReplyMessage{Service: q.Service, Action: q.Action, Task: q.Task, Data: res}
                dispatcher.reply(userId, &reply)
            } else {
                dispatcher.do(userId, &q)
            }
            //websocket.Message.Send(ws, res)
  	    }
        q.Task = 0
        //message, _ := json.Marshal(q)
        //log.Println("[QQQ]", string(message))
	}
}

func (dispatcher *Dispatcher) changeUser(userId int64, ws *websocket.Conn, cred *Credentials) (*configuration.User, int64) {
    var errClass int64
    clientId, role := dispatcher.cfg.Authenticate(cred.Login, cred.Token)

    if clientId == 0 {
        dispatcher.cfg.ProcessEvent(&api.Event{
            Class: api.EC_LOGIN_FAILED,
            Text: api.DescribeClass(api.EC_LOGIN_FAILED) + " (" + cred.Login + ")"})
        return nil, api.EC_LOGIN_FAILED
    }

    dispatcher.Lock()
    sameRole := userId == 0 || role == dispatcher.clients[userId].role
    _, loggedIn := dispatcher.clients[clientId]
    maxExceed := userId == 0 && len(dispatcher.clients) > maxClients
    
    if 0 == errClass && loggedIn {
        errClass = api.EC_ALREADY_LOGGED_IN
    }
    if 0 == errClass && !sameRole {
        errClass = api.EC_ARM_TYPE_MISMATCH
    }
    if 0 == errClass && maxExceed {
        errClass = api.EC_USERS_LIMIT_EXCEED
    }
    if errClass == 0 {
        delete(dispatcher.clients, userId)
        dispatcher.clients[clientId] = Client{ws, role}
    }
    dispatcher.Unlock()

    if errClass > 0 {
        dispatcher.cfg.ProcessEvent(&api.Event{
            Class: errClass,
            UserId: clientId})

        return nil, errClass
    }
    if userId > 0 {
        dispatcher.cfg.CompleteShift(userId)
        dispatcher.cfg.ProcessEvent(&api.Event{
            Class: api.EC_USER_LOGGED_OUT,
            UserId: userId})
    }

    dispatcher.cfg.ProcessEvent(&api.Event{
        Class: api.EC_USER_LOGGED_IN,
        UserId: clientId})
    
    dispatcher.cfg.StartNewShift(clientId)
    
    return dispatcher.cfg.GetUser(clientId), 0
}

func (dispatcher *Dispatcher) do(userId int64, q *Query) {
    // 0 - client id for permissions check
    dispatcher.RLock()
    service, ok := dispatcher.services[q.Service]
    dispatcher.RUnlock()
    if !ok {
        log.Println("[Dispatcher] Unknown service:", q.Service)
        return
    }

    //////////////// A C T I O N /////////////////
    res, broadcast := service.Do(userId, q.Action, q.Data)
    if nil == res {
        log.Println("[Dispatcher] Unknown action (or nil result)", q.Action, "for #", q.Service)
        return
    }
    reply := api.ReplyMessage{Service: q.Service, Action: q.Action, Task: q.Task, Data: res}
    
    // send to client...
    if 0 != userId {
        dispatcher.reply(userId, &reply)
    }
    
    // ...and broadcast if needed
    if broadcast {
        q.Task = 0
        dispatcher.broadcast(userId, &reply)
    }
    
    /////////////// POSTPROCESSING ////////////////
    if 0 == q.Service {
        switch q.Action {
            case "UpdateService":
            dispatcher.updateService(res) // TODO: control wrong settings
            case "DeleteService":
                dispatcher.deleteService(res)
        }
    }
}

func (dispatcher *Dispatcher) reply(cid int64, reply *api.ReplyMessage) {
    log.Println("Reply to", cid, ">", reply.Service, reply.Action, "task", reply.Task)
    //reply := ReplyMessage{Service: service, Action: action, Task: 0, Data: data}
    //log.Println(header)

    dispatcher.RLock()
    client, ok := dispatcher.clients[cid]
    dispatcher.RUnlock()
    if !ok {
        return
    }

    data := reply.Data // store original data
    /*if 0 == reply.Service && "UpdateService" == reply.Action {
        auth := dispatcher.cfg.Authorize(cid, reply.Service, api.AM_WATCH | api.AM_CONTROL)
        fmt.Println("%%%%%%%%%%%%%%%%%% AUTHO", auth)
        if 0 == len(auth) {
            reply.Data = nil
        }
    } */
    if settings, ok := reply.Data.(*api.Settings); ok {
        auth := dispatcher.cfg.Authorize(cid, settings.Id, api.AM_WATCH | api.AM_CONTROL)
        //fmt.Println("%%%%%%%%%%%%%%%%%% AUTHO", auth)
        if 0 == len(auth) {
            reply.Data = nil
        }
    } else if events, ok := reply.Data.(api.EventsList); ok {
        // filter by devices permissions
        log.Println("::: APPLY EV FILTER :::", len(events), " for svc #", reply.Service)
        devFilter := dispatcher.cfg.Authorize(cid, reply.Service, api.AM_WATCH | api.AM_CONTROL)
        reply.Data = events.Filter(cid, devFilter, armFilter[client.role])
    } else if original, ok := reply.Data.(configuration.Filterable); ok {
        // filter by devices permissions
        log.Println("::: APPLY DEV FILTER :::", reply.Service, reply.Action)
        // INFO: perform filtering inside services to handle special conditions such as groups (virtual elements)
        filter := dispatcher.cfg.Authorize(cid, reply.Service, api.AM_WATCH | api.AM_CONTROL)
        reply.Data = original.Filter(filter)
    } else {
        log.Println("::: FILTER FAILED :::", reply.Service, reply.Action)
    }
    
    if nil != reply.Data {
        res, err := json.Marshal(reply)
        if err != nil {
            panic(err)
        }
        websocket.Message.Send(client.ws, string(res))
    }
    reply.Data = data // restore data
}

func (dispatcher *Dispatcher) broadcast(exclude int64, reply *api.ReplyMessage) {
    // TODO:
    // if data.([]Event) then get automatic actions from Configuration for this event
    // serviceId + deviceState (AND ...) -> targetServiceId+deviceId+commandId
    // so all services should implement #GetDeviceState(id or complex key - string)
    // and #SendCommand(deviceId, commandId, params?)
    // maybe use chanhels for command queue?

    events, _ := reply.Data.(api.EventsList)
    
    if events != nil {
        dispatcher.processEvents(reply.Service, events)
    }
    var list []int64

    dispatcher.RLock()
    for i := range dispatcher.clients {
        if exclude != i {
            list = append(list, i)
        }
    }
    dispatcher.RUnlock()
    for _, cid := range list {
        dispatcher.reply(cid, reply)
    }
    
    if events != nil {// process events if needed
        dispatcher.scanAlgorithms(events)
    }
}

func (dispatcher *Dispatcher) processEvents(serviceId int64, events api.EventsList) {
    dispatcher.RLock()
    title := dispatcher.services[serviceId].GetSettings().Title
    dispatcher.RUnlock()
    for j := range events {
        events[j].ServiceId = serviceId
        events[j].ServiceName = title
        dispatcher.cfg.ProcessEvent(&events[j])
    }
}

// check for automatic algorihms, special events, and so
func (dispatcher *Dispatcher) scanAlgorithms(events api.EventsList) {
    var aEvents api.EventsList
    for j := range events {
        algos := events[j].Algorithms//dispatcher.cfg.CheckEvent(&events[j])
        for i := range algos {
            //log.Println("!!! ALGO:", algos[i])
            aEvents = append(aEvents, api.Event{Class: api.EC_ALGO_STARTED, Text: "Запуск алгоритма " + algos[i].Name})
        }
        
        if len(aEvents) > 0 {
            reply := api.ReplyMessage{Service: 0, Action: "Events", Data: aEvents}
            //log.Println("ALGO EVENTS:", reply)
            dispatcher.broadcast(0, &reply)
        }

        for i := range algos {
            if algos[i].TargetZoneId > 0 {
                dispatcher.doZoneCommand(0, algos[i].TargetZoneId, algos[i].Command)
            } else {
                cmd := api.Command{
                    algos[i].TargetDeviceId,
                    algos[i].Command,
                    algos[i].Argument}
                res, _ := json.Marshal(&cmd)
                /*if err != nil {
                    panic(err)
                }*/

                q := Query{algos[i].TargetServiceId, "ExecCommand", 0, res}
                //log.Println("!!! QUERY:", q)
                //log.Println("!!! CMD:", cmd)
                dispatcher.do(0, &q)
            }
        }
    }
}

// check for special events
/*func (dispatcher *Dispatcher) scanSpecial(events api.EventsList) {
    for i := range events {
        if api.EC_ENTER_ZONE == events[i].Class {
            newEvent := events[i]
            reply := api.ReplyMessage{0, "EnterZone", 0, newEvent}
        }
    }
}*/


func (dispatcher *Dispatcher) preprocessQuery(userId *int64, ws *websocket.Conn, q Query) interface{} {
    if 0 == q.Service {
        //log.Println("!!! Preprocess:", q.Service, q.Action)
        switch q.Action {
            case "ListServices": // services with statuses
                var list api.ServicesList
                dispatcher.RLock()
                for _, service := range dispatcher.services {
                    settings := service.GetSettings()
                    if 0 != settings.Id {
                        list = append(list, *settings)
                    }
                }
                dispatcher.RUnlock()
                return list

            case "ChangeUser":
                var cred Credentials
                json.Unmarshal(q.Data, &cred)
                user, errClass := dispatcher.changeUser(*userId, ws, &cred)
                if 0 == errClass {
                    *userId = user.Id
                    return user
                } else {
                    return Error{errClass, api.DescribeClass(errClass)}
                }
            case "ZoneCommand":
                var zc ZoneCommand
                err := json.Unmarshal(q.Data, &zc)
                if nil == err && zc.ZoneId > 0 && zc.Command > 0 {
                    dispatcher.doZoneCommand(*userId, zc.ZoneId, zc.Command)
                    return true
                } else {
                    return false
                }
        }
    }
    return nil
}

func (dispatcher *Dispatcher) doZoneCommand(userId, zoneId, command int64) {
    services := make(map[int64]ManageableZones)
    dispatcher.RLock()
    for i := range dispatcher.services {
        inter, ok := dispatcher.services[i].(ManageableZones)
        if ok {
            services[i] = inter
        }
    }
    dispatcher.RUnlock()

    events := api.EventsList{{UserId: userId, Class: command, ZoneId: zoneId}}
    reply := api.ReplyMessage{Service: 0, Action: "Events", Data: events}
    dispatcher.broadcast(0, &reply)
    
    list := dispatcher.cfg.LoadLinks(zoneId, "zone-device")
    for i := range services {
        var devices []int64
        for j := range list {
            if list[j][0] == i {
                devices = append(devices, list[j][1])
            }
        }
        if len(devices) > 0 {
            services[i].ZoneCommand(userId, command, devices)
        }
    }

}

func (dispatcher *Dispatcher) updateService(data interface{}) {
    var s *api.Settings
    var ok bool
    var shutdown func()
    if s, ok = data.(*api.Settings); !ok {
        log.Println("[Dispatcher] reconfiguration settings type wrong!")
        return
    }
    //log.Println("[Diaspatcher] updating service", s)
    dispatcher.RLock()
    _, ok = dispatcher.services[s.Id]
    if true == ok {
        // existing service
        shutdown = dispatcher.services[s.Id].Shutdown
    }
    dispatcher.RUnlock()
    if nil != shutdown {
        shutdown()
    }
    // (Re-)Create service
    service := factory(api.NewAPI(s, dispatcher.broadcast, dispatcher.cfg))
    if nil == service {
        log.Println("[Dispatcher] Wrong settings:", s)
    } else {
        dispatcher.useService(service)
    }
}

func (dispatcher *Dispatcher) deleteService(data interface{}) {
    var id int64
    var ok bool
    var shutdown func()
    if id, _ = data.(int64); id == 0 {
        log.Println("[Dispatcher] wrong delete id!", data)
        return
    }

    dispatcher.RLock()
    _, ok = dispatcher.services[id]
    if ok {
        log.Println("[Dispatcher] Deleting service", dispatcher.services[id].GetName())
        shutdown = dispatcher.services[id].Shutdown
    } else {
        log.Println("[Dispatcher] service to delete not found")
    }
    dispatcher.RUnlock()
    //log.Println("[Dispatcher] Shutting service down...")
    if nil != shutdown {
        shutdown()
    }
    dispatcher.Lock()
    delete(dispatcher.services, id)
    dispatcher.Unlock()
    //log.Println("[Dispatcher] Deleting service done!")
}

/*
func heartbeat(ws *websocket.Conn) {
	for i := 0; i < 10; i++ {
		time.Sleep(2000 * time.Millisecond)
		websocket.Message.Send(ws, "heartbeat")
	}		
}*/

// formatRequest generates ascii representation of a request
/*
func formatRequest(r *http.Request) string {
 // Create return string
 var request []string
 // Add the request string
 url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
 request = append(request, url)
 // Add the host
 request = append(request, fmt.Sprintf("Host: %v", r.Host))
 // Loop through headers
 for name, headers := range r.Header {
   name = strings.ToLower(name)
   for _, h := range headers {
     request = append(request, fmt.Sprintf("%v: %v", name, h))
   }
 }
 
 // If this is a POST, add post data
 if r.Method == "POST" {
    r.ParseForm()
    request = append(request, "\n")
    request = append(request, r.Form.Encode())
 } 
  // Return the request as a string
  return strings.Join(request, "\n")
}
*/