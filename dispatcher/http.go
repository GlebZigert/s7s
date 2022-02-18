package dispatcher

import (
    "log"
    "net"
    "time"
    "strings"
    "strconv"
    "context"
	"net/http"
	"golang.org/x/net/websocket"
)

import (
    "s7server/api"    
)

func (dispatcher *Dispatcher) httpServer(ctx context.Context, host string) (err error) {
    var wsConfig *websocket.Config
    //if wsConfig, err = websocket.NewConfig("ws://127.0.0.1:6080/", "http://127.0.0.1:6080"); err != nil {
    if wsConfig, err = websocket.NewConfig("ws://" + host, "http://" + host); err != nil {
		return
	}
    
    mux := http.NewServeMux()
    httpServer := &http.Server{
        Addr: host,
        Handler: mux,
        BaseContext: func (net.Listener) context.Context {return ctx},
    }
    
    //mux.Handle("/echo", ws.Handler(dispatcher.SocketServer))
    mux.HandleFunc("/", dispatcher.httpHandler)
    mux.Handle("/echo", websocket.Server{
        Handler: dispatcher.socketServer,
		Config: *wsConfig,
		Handshake: func(ws *websocket.Config, req *http.Request) error {
			//ws.Protocol = []string{"base64"}
			return nil
        }})
    
    go func() {
        <-ctx.Done()
        //log.Println("HTTP shutting down...")
        sctx, _ := context.WithTimeout(context.Background(), shutdownTimeout * time.Second)
        httpServer.Shutdown(sctx)
    }()

    log.Println("HTTP listening " + host)
    return httpServer.ListenAndServe()
}


func (dispatcher *Dispatcher) httpHandler(w http.ResponseWriter, r *http.Request) {
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


func (dispatcher *Dispatcher) socketServer(ws *websocket.Conn) {
    defer time.Sleep(100 * time.Millisecond)
    var cred Credentials
    if nil == core {
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
