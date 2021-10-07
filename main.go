package main
 
import (
    "os"
    "io"
	"log"
    "fmt"
//	"time"
    "strings"
	"net/http"
	"golang.org/x/net/websocket"
	"net/http/httputil"
    
    "./dispatcher"
)

/*var services = map[string] dispatcher.Service {
	"rif-1": {Adapter: rif.Rif{}},
	"rif-2": {Adapter: rif.Rif{}}}*/

/**********************************************************************************/

const host = "0.0.0.0:2973"

func main() {
	var dsp = dispatcher.New()
    server(&dsp)
}

/*
if len(os.Args) == 2 && len(os.Args[1]) > 0 {
		ss := strings.Split(os.Args[1], " ")
		fn = ss[len(ss)-1]
	}
*/


func server(dispatcher *dispatcher.Dispatcher) {
	/*http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World!")
	})*/
    
    var wsConfig *websocket.Config
	var err error
    //if wsConfig, err = websocket.NewConfig("ws://127.0.0.1:6080/", "http://127.0.0.1:6080"); err != nil {
    if wsConfig, err = websocket.NewConfig("ws://" + host, "http://" + host); err != nil {
		log.Fatalf(err.Error())
		return
	}
    
    mux := http.NewServeMux()
    //mux.Handle("/echo", ws.Handler(dispatcher.SocketServer))
    mux.HandleFunc("/", dispatcher.HTTPHandler)
    mux.Handle("/echo", websocket.Server{Handler: dispatcher.SocketServer,
		Config: *wsConfig,
		Handshake: func(ws *websocket.Config, req *http.Request) error {
			//ws.Protocol = []string{"base64"}
			return nil
        }})
    //mux.Handle("/cam", websocket.Handler(cam.CamSocket))

    /*http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		 http.ServeFile(w, r, "websockets.html")
	})*/
    
    //fsHandler := http.FileServer(http.Dir("./html"))
    //mux.Handle("/", proxyHandler(fsHandler))
    //mux.HandleFunc("/post", hello)
    
    log.Println("Running. Listening " + host)
    log.Println(http.ListenAndServe(host, mux))
}

func hello(w http.ResponseWriter, r *http.Request) {
    log.Println("Processs POST", r.URL.Path, r.URL.RawQuery)
    if r.URL.Path != "/post" {
        http.Error(w, "404 not found.", http.StatusNotFound)
        return
    }
    
    fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])

    f, _ := os.OpenFile("./downloaded", os.O_WRONLY|os.O_CREATE, 0666)
    defer f.Close()
    io.Copy(f, r.Body) 

    
    /*
    //file, fileHeader, err := r.FormFile("fileupload")
    file, _, err := r.FormFile("fileupload")
    if ()
    
    defer file.Close()

    // copy example
    f, _ := os.OpenFile("./downloaded", os.O_WRONLY|os.O_CREATE, 0666)
    defer f.Close()
    io.Copy(f, file) 
    
    return*/
    
    // Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
    /*if err := r.ParseForm(); err != nil {
        fmt.Println("ParseForm() err: %v", err)
        return
    }
    fmt.Println("Post from website! r.PostFrom = %v\n", r.PostForm)
    name := r.FormValue("name")
    address := r.FormValue("address")
    fmt.Fprintf(w, "Name = %s\n", name)
    fmt.Fprintf(w, "Address = %s\n", address)*/
}

    

func proxyHandler(next http.Handler) http.Handler {
	proxy1 := &httputil.ReverseProxy{Director: func(req *http.Request) {
        req.SetBasicAuth("admin", "12345")
		req.URL.Scheme = "http"
		req.URL.Host = "192.168.0.90"
	}}
  
	proxy2 := &httputil.ReverseProxy{Director: func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = "c.tile.openstreetmap.org"
	}}

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if "/streaming/channels/1/preview" == r.URL.Path {
            proxy1.ServeHTTP(w, r)
        } else if 0 == strings.Index(r.URL.Path, "/tiles/") {
            r.URL.Path = strings.Replace(r.URL.Path, "/tiles/", "/", -1)
            proxy2.ServeHTTP(w, r)
        } else {
            next.ServeHTTP(w, r)
        }
    })
}
/*
func proxy(mux *http.ServeMux) {
	origin, _ := url.Parse("http://192.168.0.90:80/")

	director := func(req *http.Request) {
        log.Println(req.URL)
		//req.Header.Add("X-Forwarded-Host", req.Host)
		//req.Header.Add("X-Origin-Host", origin.Host)
        //req.Header.Add("Authorization","Basic " + basicAuth("username1","password123")) 
        req.SetBasicAuth("admin", "12345")
		req.URL.Scheme = "http"
		req.URL.Host = origin.Host
	}

	proxy := &httputil.ReverseProxy{Director: director}

	mux.HandleFunc("/streaming/channels/1/preview", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	//log.Fatal(http.ListenAndServe(":9091", nil))
}
*/