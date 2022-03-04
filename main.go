package main
 
import (
    "os"
    "log"
    "fmt"
    //"time"
    "flag"
    "runtime"
    "context"
    "syscall"
    "os/signal"

    "s7server/api"
    "s7server/dispatcher"
)

const (
    defaultHost = "0.0.0.0:2973"
    winStoragePath = "./storage/"
    linStoragePath = "/var/lib/s7server/"
)

var Version = "v0.0"
var Commit = "012345"
var BuildTime = "01.01.01 00:00"


func getPath() (path string) {
    if runtime.GOOS == "windows" {
        path = winStoragePath
    } else {
        path = linStoragePath
    }
    return
}

func main() {
    version := flag.Bool("v", false, "print version")
    host := flag.String("host", defaultHost, "http host (ip:port)")
    dataDir := flag.String("data", getPath(), "data files path")
    flag.Parse()
    
    vText := "s7server " + Version + "-" + Commit + " (built on " + BuildTime + ")"
    
    if *version {
        fmt.Println(vText)
        return
    }
    
    if (*dataDir)[len(*dataDir) - 1] != '/' {
        *dataDir += "/"
    }
    api.DataStoragePath = *dataDir
    
    log.Println("Starting", vText, "on data", *dataDir)
    
    //log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
    ctx, cancel := context.WithCancel(context.Background())
    c := make(chan os.Signal, 1) // use buffer (size = 1), so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
    go func() {
        <-c
        log.Println("TERM signal recieved, let's stop")
        cancel()
	}()
    
    err := dispatcher.Run(ctx, *host)
    if nil != err {
        log.Println(err)
    } else {
        log.Println("Bye!")
    }
    //time.Sleep(1 * time.Second)
}