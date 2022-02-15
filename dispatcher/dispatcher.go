package dispatcher

import (
    "io"
	"log"
	"fmt"
	"time"
    "sync"
//    "io/ioutil"
    "errors"
    "context"
    "encoding/json"
//    "encoding/base64"
	"golang.org/x/net/websocket"
)

import (
    "../api"
	"../adapters/rif"
    "../adapters/axxon"
    "../adapters/z5rweb"
    "../adapters/configuration"
)

const (
    serviceRestartDelay = 5 // seconds
    maxClients = 100
    loginTimeout = 3 // seconds
    keepAliveInterval = 10 + 2 // seconds (time + ping)
    shutdownTimeout = 10 // seconds
)

var core configuration.ConfigAPI

func factory(api *api.API) Service {
    var service Service
    switch (*api).Settings.Type {
        case "configuration": service = &configuration.Configuration{API: *api}
        case "rif": service = &rif.Rif{API: *api}
        //case "sigur": service = &sigur.Sigur{API: *api}
        case "axxon": service = &axxon.Axxon{API: *api}
        case "z5rweb": service = &z5rweb.Z5RWeb{API: *api}
    }
    return service
}

func Run(ctx context.Context, host string) (err error) {
    //seedFilter()
    var d = Dispatcher{
        ctx: ctx,
        queue: make(chan string, 10),
        services: make(map[int64] Service),
        clients: make(map[int64] Client)}

    outbox = make(chan api.ReplyMessage, maxQueueSize / 10) // buffered replies
    go d.queueServer(ctx)
    
    cfg := factory(api.NewAPI(&api.Settings{Id: 0, Type: "configuration"}, d.broadcast))
    core = cfg.(configuration.ConfigAPI)
    d.services[0] = cfg
    err = d.services[0].Run(core)
    if nil != err {
        return
    }
    
    settings := core.Get()
    for _, s := range settings {
        service := factory(api.NewAPI(s, d.broadcast))
        if nil == service {
            log.Println("[Dispatcher] Unknown service type:", s)
        } else {
            go d.runService(service)
        }
    }
    
    log.Println("Dispatcher startup completed")
    err = d.httpServer(ctx, host)
    if nil != err && nil == ctx.Err() {
        err = fmt.Errorf("HTTP server failure: %w", err)
    } else {
        err = nil        
        log.Println("HTTP server stopped")
    }

    d.shutdown()
    
    return
}

func (dispatcher *Dispatcher) shutdown() {
    var wg sync.WaitGroup
    var id int64
    
    // 1. shutdown services
    dispatcher.RLock()
    for id = range dispatcher.services {
        if id > 0 {
            serviceId := id
            wg.Add(1)
            go func() {
                defer wg.Done()
                dispatcher.shutdownService(serviceId)
            }()
        }
    }
    dispatcher.RUnlock()
    
    // 2. wait for services to shutdown properly
    c := make(chan struct{})
    go func() {
        defer close(c)
        wg.Wait()
    }()    
    select {
        case <-c:
            log.Println("All services stopped")
        case <-time.After(shutdownTimeout * time.Second):
            log.Println("Some services are hang")
    }
    
    // 3. wait for core shutdown
    log.Println("Stopping core...")
    c = make(chan struct{})
    go func() {
        defer close(c)
        dispatcher.shutdownService(0)
    }()    
    select {
        case <-c:
            log.Println("Core stopped")
        case <-time.After(shutdownTimeout * time.Second):
            log.Println("Core hang")
    }
}

func (dispatcher *Dispatcher) shutdownService(id int64) {
    var shutdown func()
    dispatcher.RLock()
    if _, ok := dispatcher.services[id]; ok {
        // existing service
        shutdown = dispatcher.services[id].Shutdown
    }
    dispatcher.RUnlock()
    if nil != shutdown {
        log.Println("Stopping #", id)
        shutdown()
        log.Println("Finished #", id)
    } else {
        log.Println("It's a new service? Can't shutdown unknown #", id)
    }
}

// should be called async! don't call twice for the same service
func (dispatcher *Dispatcher) runService(service Service) {
    settings := service.GetSettings()
    id := settings.Id
    dispatcher.Lock()
    if nil == dispatcher.services[id] {
        dispatcher.services[id] = service
    } else {
        id = 0
    }
    dispatcher.Unlock()
    
    if 0 == id {
        return
    }
    
    for nil == dispatcher.ctx.Err() {
        // TODO: dispatcher shutdown can happens here!
        // service should exit with NOT NIL error only in case of real failure
        err := serviceWrapper(service, core)
        if nil == err {
            break
        }
        log.Println("Service", service.GetName(), "crashed, restart in", serviceRestartDelay, "seconds:", err)
        time.Sleep(serviceRestartDelay * time.Second)
    }
    dispatcher.Lock()
    delete(dispatcher.services, id)
    dispatcher.Unlock()
    log.Println("runService: service stopped", service.GetName())
}

func serviceWrapper(service Service, cfg configuration.ConfigAPI) (err error) {
    defer func() {
        if r := recover(); r != nil {
            switch x := r.(type) {
                case error: err = x
                default: err = errors.New(fmt.Sprint(r))
            }
        }
    }()
    return service.Run(cfg)
}

func (dispatcher *Dispatcher) loggedIn(userId int64) (really bool) {
    _, really = dispatcher.clients[userId]
    return
}

func (dispatcher *Dispatcher) loginError(ws *websocket.Conn, class int64) {
    ev := api.Event{Event: class, Class: api.EC_ERROR}
    dispatcher.broadcastEvent(&ev)
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
        dispatcher.broadcastEvent(&api.Event{
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
    clientId, role := core.Authenticate(cred.Login, cred.Token)

    if clientId == 0 {
        dispatcher.broadcastEvent(&api.Event{
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
        dispatcher.broadcastEvent(&api.Event{
            Class: errClass,
            UserId: clientId})

        return nil, errClass
    }
    if userId > 0 {
        core.CompleteShift(userId)
        dispatcher.broadcastEvent(&api.Event{
            Class: api.EC_USER_LOGGED_OUT,
            UserId: userId})
    }

    dispatcher.broadcastEvent(&api.Event{
        Class: api.EC_USER_LOGGED_IN,
        UserId: clientId})
    
    core.StartNewShift(clientId)
    
    user, _ := core.GetUser(clientId) // TODO: handle err
    return user, 0
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

    /////////////// POST-PROCESSING ////////////////
    if 0 == q.Service {
        switch q.Action {
            case "UpdateService": dispatcher.updateService(res) // TODO: control wrong settings
            case "DeleteService": dispatcher.deleteService(res)
        }
    }

    // prepare Service.Status for marshall
    if s, ok := res.(*api.Settings); ok {
        s.Status.RLock()
        res = *s
        s.Status.RUnlock()
    }

    /// REPLY
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

}

func (dispatcher *Dispatcher) broadcastEvent(event *api.Event) {
    reply := api.ReplyMessage{Service: 0, Action: "Events", Task: 0, Data: api.EventsList{*event}}
    dispatcher.broadcast(0, &reply)
}

func (dispatcher *Dispatcher) reply(cid int64, reply *api.ReplyMessage) {
    reply.UserId = cid
    outbox <-*reply
}


func (dispatcher *Dispatcher) broadcast(exclude int64, reply *api.ReplyMessage) {
    // TODO:
    // if data.([]Event) then get automatic actions from Configuration for this event
    // serviceId + deviceState (AND ...) -> targetServiceId+deviceId+commandId
    // so all services should implement #GetDeviceState(id or complex key - string)
    // and #SendCommand(deviceId, commandId, params?)
    // maybe use channels for command queue?

    //var err error
    var list []int64
    //log.Println("BroadC", reply)
    /*events, _ := reply.Data.(api.EventsList)
    
    if events != nil {
        err = dispatcher.processEvents(reply.Service, events)
    }
    if nil != err {
        return // dont't broadcast failed events
    }*/

    dispatcher.RLock()
    for i := range dispatcher.clients {
        if exclude != i {
            list = append(list, i)
        }
    }
    dispatcher.RUnlock()
    
    if 0 == len(list) { // if no clients connected
        dispatcher.reply(0, reply)
    }
    
    for _, cid := range list {
        dispatcher.reply(cid, reply)
    }
    
    /*if events != nil {// process events if needed
        dispatcher.scanAlgorithms(events)
    }*/
}

func (dispatcher *Dispatcher) preprocessQuery(userId *int64, ws *websocket.Conn, q Query) interface{} {
    if 0 == q.Service {
        //log.Println("!!! Preprocess:", q.Service, q.Action)
        switch q.Action {
            case "ListServices": // services with statuses
                var list []api.Settings
                dispatcher.RLock()
                for _, service := range dispatcher.services {
                    settings := service.GetSettings()
                    if 0 != settings.Id {
                        idList := service.GetList()
                        //log.Println("ListAllDevices", idList)
                        filter, err := core.Authorize(*userId, idList)
                        //log.Println("FILTER", filter)
                        // TODO: handle err (report db failure)
                        if nil == err && len(filter) > 0 {
                            settings.Status.RLock()
                            list = append(list, *settings)
                            settings.Status.RUnlock()
                        }
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
    
    // TODO: handle err
    list, _ := core.LoadLinks(zoneId, "zone-device")

    var devices []int64
    for i := range list {
        devices = append(devices, list[i][1])
    }

    if 0 == len(devices) {
        return // Zone is empty
    }

    // TODO: handle err
    filter, _ := core.Authorize(userId, devices)
    devices = make([]int64, len(filter))
    for id, flags := range filter {
        // filter[0] > 0 => all id are acceptable
        if filter[0] > 0 || flags & api.AM_CONTROL > 0 {
             devices = append(devices, id)
        }
    }

    if 0 == len(devices) {
        return // No permitted devices
    }

    for i := range services {
        // WARN: devices should be read-only
        go services[i].ZoneCommand(userId, command, devices)
    }
}

func (dispatcher *Dispatcher) updateService(data interface{}) {
    var s *api.Settings
    var ok bool
    //var shutdown func()
    if s, ok = data.(*api.Settings); !ok {
        log.Println("[Dispatcher] reconfiguration settings type wrong!")
        return
    }
    //log.Println("[Diaspatcher] updating service", s)
    dispatcher.shutdownService(s.Id)
    // (Re-)Create service
    service := factory(api.NewAPI(s, dispatcher.broadcast))
    if nil == service {
        log.Println("[Dispatcher] Wrong settings:", s)
    } else {
        go dispatcher.runService(service) // run async!
    }
}

func (dispatcher *Dispatcher) deleteService(data interface{}) {
    var id int64
    //var ok bool
    //var shutdown func()
    if id, _ = data.(int64); id == 0 {
        log.Println("[Dispatcher] wrong delete id!", data)
        return
    }

    dispatcher.shutdownService(id)
}

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