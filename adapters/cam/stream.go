package cam

import (
    //"fmt"
	"log"
	"time"
//    "strconv"

	"github.com/deepch/vdk/format/rtsp"
)

func ServeStreams() {
	for k, v := range Config.Streams {
		go func(name, url string) {
			for {
				log.Println(name, "connect", url)
				//rtsp.DebugRtsp = true
				session, err := rtsp.Dial(url)
				if err != nil {
					log.Println(name, err)
					time.Sleep(5 * time.Second)
					continue
				}
				session.RtpKeepAliveTimeout = time.Duration(10 * time.Second)
				codecs, err := session.Streams()
				if err != nil {
					log.Println(name, err)
					time.Sleep(5 * time.Second)
					continue
				}
				Config.addCodecs(name, codecs)
                //go speedup(session, 4)
				for {
					pkt, err := session.ReadPacket()
					if err != nil {
						log.Println(name, err)
						break
					}
                    //fmt.Printf("rtp: time for stream #%d: ...%v\n", pkt.Idx, pkt.Time)
					Config.cast(name, pkt)
				}
				session.Close()
				log.Println(name, "reconnect wait 5s")
				time.Sleep(5 * time.Second)
			}
		}(k, v.URL)
	}
}

/*
func speedup(client *rtsp.Client, scale int) {
	req := rtsp.Request{
		Method: "PLAY",
		Uri:    client.requestUri,
	}
	req.Header = append(req.Header, "Session: " + client.session)
    req.Header = append(req.Header, "Scale: " + strconv.Itoa(scale))
    if err := client.WriteRequest(req); err != nil {
		return
	}
	return
}*/