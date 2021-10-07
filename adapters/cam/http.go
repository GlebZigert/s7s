package cam

import (
	"log"
	//"net/http"
	//"sort"
	"time"

	"github.com/deepch/vdk/format/mp4f"
//	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)
/*
func serveHTTP() {
	router := gin.Default()
	router.LoadHTMLGlob("web/templates/*")
	router.GET("/", func(c *gin.Context) {
		fi, all := Config.list()
		sort.Strings(all)
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"port":     Config.Server.HTTPPort,
			"suuid":    fi,
			"suuidMap": all,
			"version":  time.Now().String(),
		})
	})
	router.GET("/player/:suuid", func(c *gin.Context) {
		_, all := Config.list()
		sort.Strings(all)
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"port":     Config.Server.HTTPPort,
			"suuid":    c.Param("suuid"),
			"suuidMap": all,
			"version":  time.Now().String(),
		})
	})
	router.GET("/ws/:suuid", func(c *gin.Context) {
		handler := websocket.Handler(ws)
		handler.ServeHTTP(c.Writer, c.Request)
	})
	router.StaticFS("/static", http.Dir("web/static"))
	err := router.Run(Config.Server.HTTPPort)
	if err != nil {
		log.Fatalln(err)
	}
}
*/
func CamSocket(ws *websocket.Conn) {
	defer ws.Close()
	suuid := ws.Request().FormValue("suuid")
	if Config.ext(suuid) {
	   log.Println("Request", suuid)
        ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
		suuid := ws.Request().FormValue("suuid")
		cuuid, ch := Config.addClient(suuid)
		defer Config.delClient(suuid, cuuid)
		codecs := Config.getCodecs(suuid)
		if codecs == nil {
			log.Println("No Codec Info")
			return
		}
		muxer := mp4f.NewMuxer(nil)
		muxer.WriteHeader(codecs)
		meta, init := muxer.GetInit(codecs)
		err := websocket.Message.Send(ws, append([]byte{9}, meta...))
		if err != nil {
			return
		}
		err = websocket.Message.Send(ws, init)
		if err != nil {
			return
		}
		var start bool
packets := 1
start = true
		for {
			select {
			case pck := <-ch:
				if pck.IsKeyFrame {
					start = true
				}
				if !start {
					continue
				}
//				packets = packets + 1
				/*var ready bool
				var buf []uint8
				if 0 != packets % 200 {*/
    				    ready, buf, _ := muxer.WritePacket(pck, false)
/*				} else {
				    log.Println("Drop video packet")
				}*/
				if ready {
				    if packets < 50 || packets > 53{
    					    ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
    					    err := websocket.Message.Send(ws, buf)
    					    if err != nil {
						return
					    }
				    } else {
					log.Println("Drop:", packets)
				    }
				    packets = packets + 1
				}
			}
		}
	log.Println("!!! Playing:", suuid)
    } else {
        log.Println("Unknown:", suuid)
    }
}
