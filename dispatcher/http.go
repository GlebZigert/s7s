package dispatcher

import (
    "io"
    "log"
    "net"
    "time"
    "strings"
    "strconv"
    "context"
	"net/http"
    "io/ioutil"
    "crypto/md5"
    "encoding/hex"
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
    defer func () {
        if nil != r.Body {
            // should not only to close the body, but also to drain it too!
            // https://pkg.go.dev/net/http#Client.Do
            io.Copy(ioutil.Discard, r.Body)
            r.Body.Close()
        }
    }()
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
    
    if 0 == id {
        if code := dispatcher.checkAuth(w, r); code > 0 {
            http.Error(w, http.StatusText(code), code)
            return
        }
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

func (dispatcher *Dispatcher) checkAuth(w http.ResponseWriter, r *http.Request) (httpErrCode int) {
    var ok bool
    var client Client

    u, p, ok := r.BasicAuth()
    if !ok {
        w.Header().Set("WWW-Authenticate", `Basic realm="Restricted", charset="UTF-8"`)
		log.Println("No credentials for", r.RemoteAddr)
        httpErrCode = http.StatusUnauthorized
        return
	}
    
    id, err := strconv.ParseInt(u, 10, 64)
    if nil == err && id > 0 {
        dispatcher.RLock()
        client, ok = dispatcher.clients[id]
        dispatcher.RUnlock()
    }
    
    if !ok {
		httpErrCode = http.StatusUnauthorized
        log.Println("Wrong username for", r.RemoteAddr)
        return
    }
    
    // use short-time session keys, +- 5 seconds
    var check string
    start := time.Now().Unix()
    for i := start - maxTimeMismatch; i <= start + maxTimeMismatch; i++ {
        check = md5hex(client.token + strconv.FormatInt(i, 10))
        if p == check {
            break
        }
    }    
    if p != check {
		httpErrCode = http.StatusUnauthorized
        log.Println("Wrong password for", r.RemoteAddr)
	}

    return
}



func (dispatcher *Dispatcher) socketServer(ws *websocket.Conn) {
    defer time.Sleep(100 * time.Millisecond)
    var cred Credentials
    if nil == core {
        log.Println("not ready, try later")
        return // not ready, try later
    }
    
    log.Println("New client:", ws.Request().RemoteAddr)
    
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
        log.Println("Serving #", user.Id, "(" + cred.Login + ")")
        dispatcher.serveClient(user.Id, ws)
    } else if api.EC_TIME_MISMATCH == errClass {
        log.Println("Client/server time offset is more than", maxTimeMismatch, "seconds")
    } else {
        log.Println("Wrong password or unknown user:", cred.Login)
    }
}

func md5hex(text string) string {
   hash := md5.Sum([]byte(text))
   return hex.EncodeToString(hash[:])
}
