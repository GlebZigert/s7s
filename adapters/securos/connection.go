package securos

import (
//	"os"
//	"io"
//	"net"
//	"bufio"
//	"fmt"
	"log"
    "context"
	//"encoding/xml"
	//"strings"
	"time"
)

const reconnectInterval = 3 // seconds

func (svc *Securos) connect(ctx context.Context) {
    var lastTryTime time.Time
    //var dialer net.Dialer
    
	host := svc.Settings.Host
	newTry := true

    for !svc.Cancelled(ctx) {
        svc.Sleep(ctx,
                  time.Duration(reconnectInterval) * time.Second - time.Now().Sub(lastTryTime));
        lastTryTime = time.Now()

        if (newTry) {
            log.Print(svc.GetName(), ": trying to connect to " + host)
		}
        
        /*dctx, _ := context.WithTimeout(ctx, 500*time.Millisecond)
        conn, err := dialer.DialContext(dctx, "tcp", host)
		if err != nil {
			//log.Print("No connection, retry in 5s...")
			if (newTry) {
				newTry = false
                svc.SetTCPStatus("offline")
			}
			time.Sleep(time.Duration(500)*time.Millisecond);
			continue;
		}

        svc.Lock()
        svc.conn = conn
        svc.Unlock()
        
		newTry = true

        log.Println(svc.GetName(), ": connected to ", host)
        svc.SetTCPStatus("online")
        */
	}
	log.Println(svc.GetName(), "connection closed")
}
/*
func (rif *Rif) closeConnection() {
    svc.RLock()
    notNil := nil != svc.conn
    svc.RUnlock()
    
    if notNil {
        svc.conn.Close()
    }
}

func (rif *Rif) keepAlive(ctx context.Context, interval int) {
	keepAliveMsg := `<RIFPlusPacket type="KeepAlive" />`
    // TODO: update keep-alive on settings change
    if interval < 1 {
		interval = 5
	}
    for !svc.Cancelled(ctx) {
		svc.Sleep(ctx, time.Duration(interval) * time.Second)
		svc.send(keepAliveMsg)
	}
    log.Println(svc.GetName(), "keep-alive stopped")
}

func (rif *Rif) SendCommand(xml string) {
	//fmt.Println("Command: " + xml)
	if svc.send(xml) {
		if (svc.log != nil) {
			svc.log.Write([]byte("\n\n====================== >>> =====================\n\n" + xml))
		}
	} else {
		log.Println("Can't send remote command")
	}
}

func (rif *Rif) send(data string) bool {
	res := false
    svc.RLock()
    conn := svc.conn
    svc.RUnlock()
    if (conn != nil) {
		// TODO: check for write buffer overflow
		n, err := conn.Write([]byte(data)) //n, err := fmt.Fprintf(svc.conn, data)
		if err == nil && n == len(data) {
			res = true
		}
	}
	return res;
}
*/