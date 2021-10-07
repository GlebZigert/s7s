package sigur

import (
//	"os"
//	"io"
	"net"
	"bufio"
//	"fmt"
	"log"
	"time"
    "regexp"
	"strings"
    "strconv"
    "context"
    "errors"
    //"text/scanner"
)

// 192.168.0.237:3312

const reconnectInterval = 3 // seconds
const dtLayout = "2006-01-02 15:04:05"

func (api *Sigur) connect(ctx context.Context) {
	var lastTryTime time.Time
    var dialer net.Dialer
    var err error
    var conn net.Conn

    host := api.Settings.Host
	newTry := true

    for !api.Cancelled(ctx) {
        api.Sleep(ctx, time.Duration(reconnectInterval) * time.Second - time.Now().Sub(lastTryTime));
        lastTryTime = time.Now()

        if (newTry) {
            log.Println(api.GetName(), "trying to connect to", host)
		}

        if nil != conn {
	       conn.Close()
        }
        dctx, _ := context.WithTimeout(ctx, 500*time.Millisecond)
        conn, err = dialer.DialContext(dctx, "tcp", host)
        if err != nil {
            if (newTry) {
                log.Println(api.GetName(), err)
                api.SetTCPStatus("offline")
            }
            newTry = false
            continue;
        }

        if (newTry) {
            log.Println(api.GetName(), "connected to", host)
        }

		//scanner := bufio.NewScanner(conn)

        // timeout for init: login, subscribe and get all APINFO
        deadline := time.Now().Add(time.Second * 3)
        conn.SetReadDeadline(deadline)

        
        var devToWait int
        err = api.loginAndSubscribe(conn)
        if nil == err {
            devToWait, err = api.apList(conn)
        }
        if nil != err {
            if (newTry) {
                log.Println(api.GetName(), err)
                api.SetTCPStatus("error")
                newTry = false
            }
            continue
        }
        api.readLoop(conn, devToWait)
        newTry = true
	}
    log.Println(api.GetName(), "TCP stream stopped")
}

func (api *Sigur) loginAndSubscribe(conn net.Conn) error {
    commands := []string {
        `LOGIN 1.8 "` + api.Settings.Login + `" "` + api.Settings.Password + `"`,
        "SUBSCRIBE CE"}

    scanner := bufio.NewScanner(conn)

    ok := true
    for i:= 0; i < len(commands) && ok; i++ {
         ok = send(conn, commands[i]) && scanner.Scan() && scanner.Text() == "OK"
    }
    if !ok {
        return errors.New("Authentication failed")
    } else {
        return nil
    }
}

func (api *Sigur) apList(conn net.Conn) (int, error) {
    var data []string
    scanner := bufio.NewScanner(conn)
    
    send(conn, "GETAPLIST")

    for scanner.Scan() {
        data = strings.Split(scanner.Text(), " ")
        if len(data) > 0 && "APLIST" == data[0] {
            break
        }
    }
    if len(data) > 1 && "APLIST" == data[0] && "EMPTY" != data[1] {
        for i := 1; i < len(data); i++ {
            send(conn, "GETAPINFO " + data[i])
        }
        return len(data) - 1, nil
    } else {
        return 0, errors.New("Can't query AP list")
    }
    
}


func (api *Sigur) readLoop(conn net.Conn, devToWait int) {
    polled := false // all devices are polled?
    devs := make(map[int]struct{})
    
    //messageRE := regexp.MustCompile(`[^\s"']+|"(?:[^"\\]|\\.)*"`)
    messageRE := regexp.MustCompile(`[^\s"']+|"([^"\\]|\\.)*"`)
    //devToWait := len(api.devices)
    c := make(chan []string, 2)
    defer close(c)
    go api.messageServer(c)
    
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        data := messageRE.FindAllString(scanner.Text(), -1)
        for i, _ := range data {
            //data[i] = strings.ReplaceAll(strings.Trim(data[i], "\""), "\\", "")
            data[i] = strings.Replace(strings.Trim(data[i], "\""), "\\", "", -1)
        }
        log.Println(api.GetName(), data)

        switch data[0] {
            case "EVENT_CE", "OBJECTINFO": c <- data
                
            case "ERROR":
                log.Println(api.GetName(), scanner.Text())

            case "APINFO":
                // APINFO ID 1 NAME "╨в1" ZONEA 1 ZONEB 0 STATE ONLINE_UNLOCKED CLOSED?
                if !polled {
                    id, err := strconv.Atoi(data[2])
                    if nil == err {
                        devs[id] = struct{}{}
                    }

                    if len(devs) >= devToWait {
                        polled = true
                        devs = nil
                        conn.SetReadDeadline(time.Time{})
                        api.Lock()
                        api.conn = conn // now connection can be used for commands
                        api.Unlock()
                        log.Println("All", devToWait, "device(s) polled")
                        api.SetTCPStatus("online")
                    }
                }
                c <- data
        }
    }

    err := scanner.Err()
    if nil != err {
        log.Println(api.GetName(), err)
        api.SetTCPStatus("error")
    }
}

func (api *Sigur) resetConnection() {
    api.RLock()
    conn := api.conn
    api.RUnlock()
    
    if nil != conn {
	   conn.Close()
    }
}

func (api *Sigur) send(data string) bool {
    api.RLock()
    conn := api.conn
    api.RUnlock()
    return send(conn, data)
}

/********************************************************************************/

func send(conn net.Conn, data string) bool {
    log.Println("> > >", data)
	res := false
	if (conn != nil) {
		n, err := conn.Write([]byte(data + "\r\n"))
        if err == nil && n == len(data) + 2 { // TODO: Check n? Really?
			res = true
		}
	}
	return res;
}

func time2num(dt string) int64 {
    dtn, err := time.Parse(dtLayout, strings.Trim(dt, `"`))
    if nil != err {
        // TODO: log err?
        dtn = time.Now()
    }
    return dtn.Unix()
}
