package rif

import (
//	"io"
	"net"
	"bufio"
//	"fmt"
	"log"
    "regexp"
    "context"
	"encoding/xml"
	"strings"
	"time"
    "s7server/api"
//	"golang.org/x/net/html/charset"
)

const reconnectInterval = 5 // seconds
var rifPacketRE = regexp.MustCompile(`<RIFPlusPacket[^>]*?/>`)

func (rif *Rif) connect(ctx context.Context) {
    //var lastTryTime time.Time
    var dialer net.Dialer
    // INFO: settings are considered immutable, so locking is not required
    //keepAlive := rif.Settings.KeepAlive + 2 // + ping time
	host := rif.Settings.Host
	newTry := true
    
    listCommand := `<RIFPlusPacket type="Commands"><Commands><Command id="0"/><Command id="10000"/></Commands></RIFPlusPacket>`

    var err error
    for !rif.Cancelled(ctx) {
        if nil != err { // wait before reconnect
            rif.Warn("connection loop err:", err)
            // TODO: report error?
            // rif.SetServiceStatus(api.EC_SERVICE_OFFLINE, api.EC_DATABASE_UNAVAILABLE)
            rif.Sleep(ctx, reconnectInterval * time.Second)
        }

        if (newTry) {
            rif.Log("Trying to connect to", host)
		}

        dctx, _ := context.WithTimeout(ctx, 500 * time.Millisecond)
        var conn net.Conn
        conn, err = dialer.DialContext(dctx, "tcp", host)
		if err != nil {
            rif.SetServiceStatus(api.EC_SERVICE_OFFLINE, api.EC_DATABASE_UNAVAILABLE)
            newTry = false
			continue
		}

        rif.Lock()
        rif.conn = conn
        rif.Unlock()
        
		newTry = true

        rif.Log("Connected to", host)
		
		netReader := bufio.NewReader(rif.conn)
        rif.SendCommand(listCommand)
        rif.queryEventsChan <-0

        var message, packet string
        for nil == err {
            //conn.SetReadDeadline(time.Now().Add(time.Duration(keepAlive) * time.Second))
            packet, err = netReader.ReadString('>')
            if err == nil { // io.EOF
                message += packet
                //if strings.Index(packet, "</RIFPlusPacket>") >= 0 {
                if strings.Index(packet, "</RIFPlusPacket>") >= 0 || rifPacketRE.MatchString(packet) {
                    message = win2utf8(message)
                    t := time.Now().Format("2006-01-02T15:04:05.999")
                    rif.logXml("\n\n====================== { " + t + " { =====================\n\n" + message)
                    p := RIFPlusPacket{}
                    // TODO: ignore parse error?
                    err = xml.Unmarshal([]byte(message), &p)
                    if nil == err {
                        switch p.Type {
                            case "InitialStatus": err = rif.populate(p.Devices)
                            case "EventsAndStates": rif.update(p.Devices)
                            case "ListJourRecord": rif.scanJourEvents(p.Events)
                            default: rif.Warn("Unknown RIFPlusPacket type:", p.Type)
                        }
                    }
                    message = ""
                }
            }
        }
        
		// listen for reply
		/*netReader, _ := charset.NewReaderLabel("windows-1251", rif.conn)
		d := xml.NewDecoder(netReader)
		//d.CharsetReader = charset.NewReaderLabel
		
		for {
			p := RIFPlusPacket{}
			if err := d.Decode(&p); err != nil {
                rif.Warn(err)
                break
			} else {
                if "InitialStatus" == p.Type {
                    rif.init(p.Devices)
                } else {
                    rif.update(p.Devices)
                }
			}
		}*/
	}
	log.Println(rif.GetName(), "connection closed")
}

func (rif *Rif) logXml(s string) {
    if (rif.xmlLog != nil) {
        rif.xmlLog.Write([]byte(s))
    }
}

func (rif *Rif) closeConnection() {
    conn := rif.getConn()
    if nil != conn {
        conn.Close()
    }
}

func (rif *Rif) keepAlive(ctx context.Context, interval int) {
	keepAliveMsg := `<RIFPlusPacket type="KeepAlive" />`
    // TODO: update keep-alive on settings change
    if interval < 1 {
		interval = 5
	}
    for !rif.Cancelled(ctx) {
		//rif.send(keepAliveMsg)
        rif.SendCommand(keepAliveMsg)
        rif.Sleep(ctx, time.Duration(interval) * time.Second)
	}
    rif.Log("keep-alive stopped")
}



func (rif *Rif) SendCommand(xml string) {
	//fmt.Println("Command: " + xml)
    t := time.Now().Format("2006-01-02T15:04:05.999")
	if rif.send(xml) {
        rif.logXml("\n\n====================== } " + t + " } =====================\n\n" + xml)
	} else {
		rif.Warn("Can't send remote command")
	}
}

func (rif *Rif) send(data string) bool {
	res := false
    conn := rif.getConn()
    if (conn != nil) {
		// TODO: check for write buffer overflow
		n, err := conn.Write([]byte(data)) //n, err := fmt.Fprintf(rif.conn, data)
        if err != nil {
            rif.Warn("TCP send err:", err)
        } else if n == len(data) {
			res = true
		}
    }
	return res
}
func (rif *Rif) getConn() (conn net.Conn) {
    rif.RLock()
    conn = rif.conn
    rif.RUnlock()
    return
}
