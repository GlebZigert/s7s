package main
 
import (
	"log"
	"time"
	"net/http"
	"golang.org/x/net/websocket"
)
 
func main() {
	/*http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World!")
	})*/
	http.Handle("/echo", websocket.Handler(WSServer))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		 http.ServeFile(w, r, "websockets.html")
	})
	log.Println("Running.")
	http.ListenAndServe("localhost:8081", nil)
}

func WSServer(ws *websocket.Conn) {

	log.Println("Connected")

	go heartbeat(ws)

	var msg string
	var err error
	
	for {
		if err = websocket.Message.Receive(ws, &msg); err != nil {
	    	
	    	log.Fatal("exit! ", err)
	    } else {
	    	log.Println(msg)
	    	websocket.Message.Send(ws, msg)
	    }
	}
}

func heartbeat(ws *websocket.Conn) {
	for i := 0; i < 10; i++ {
		time.Sleep(2000 * time.Millisecond)
		websocket.Message.Send(ws, "heartbeat")
	}		
}