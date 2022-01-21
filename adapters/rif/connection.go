package rif

import (
//	"io"
	"net"
	"bufio"
//	"fmt"
	"log"
    "context"
	"encoding/xml"
	"strings"
	"time"
    "../../api"
//	"golang.org/x/net/html/charset"
)

const reconnectInterval = 3 // seconds

func (rif *Rif) connect(ctx context.Context) {
    var lastTryTime time.Time
    var dialer net.Dialer
    
	host := rif.Settings.Host
	newTry := true

    listCommand := `<RIFPlusPacket type="Commands"><Commands><Command id="0"/><Command id="10000"/></Commands></RIFPlusPacket>`

    for !rif.Cancelled(ctx) {
        rif.Sleep(ctx,
                  time.Duration(reconnectInterval) * time.Second - time.Now().Sub(lastTryTime));
        lastTryTime = time.Now()

        if (newTry) {
            log.Print(rif.GetName(), ": trying to connect to " + host)
		}
        
        dctx, _ := context.WithTimeout(ctx, 500 * time.Millisecond)
        conn, err := dialer.DialContext(dctx, "tcp", host)
		if err != nil {
			//log.Print("No connection, retry in 5s...")
			if (newTry) {
                rif.SetServiceStatus(api.EC_SERVICE_OFFLINE, api.EC_DATABASE_UNAVAILABLE)
                newTry = false
			}
			//time.Sleep(time.Duration(500)*time.Millisecond);
			continue;
		}

        rif.Lock()
        rif.conn = conn
        rif.Unlock()
        
		newTry = true

        log.Println(rif.GetName(), ": connected to ", host)
        // rif.SetTCPStatus("online")  => moved to populate()
		
		netReader := bufio.NewReader(rif.conn)
        rif.SendCommand(listCommand)
        rif.queryEventsChan <-0

        EOF := false
		for EOF == false {
            message := ""
			for {
				packet, err := netReader.ReadString('>')
				if err != nil { // io.EOF
					EOF = true
					rif.Warn(err)
					break
				}
				message += packet
				if (strings.Index(packet, "</RIFPlusPacket>") >= 0) {
                    message = win2utf8(message)
                    rif.logXml("\n\n====================== { { { =====================\n\n" + message)
                    p := RIFPlusPacket{}
                    if err := xml.Unmarshal([]byte(message), &p); err != nil {
                        EOF = true // reset connection
                        rif.Warn(err)
                        break
                    } else {
                        switch p.Type {
                            case "InitialStatus": rif.populate(p.Devices)
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
		rif.send(keepAliveMsg)
        rif.Sleep(ctx, time.Duration(interval) * time.Second)
	}
    rif.Log("keep-alive stopped")
}



func (rif *Rif) SendCommand(xml string) {
	//fmt.Println("Command: " + xml)
	if rif.send(xml) {
        rif.logXml("\n\n====================== } } } =====================\n\n" + xml)
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
