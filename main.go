package main
 
import (
    "os"
    "log"
    //"time"
    "flag"
    "runtime"
    "context"
    "syscall"
    "os/signal"

    "./api"
    "./dispatcher"
)

const (
    defaultHost = "0.0.0.0:2973"
    winStoragePath = "./storage/"
    linStoragePath = "/var/lib/s7server/"
)

func getPath() (path string) {
    if runtime.GOOS == "windows" {
        path = winStoragePath
    } else {
        path = linStoragePath
    }
    return
}

func main() {
    host := flag.String("host", defaultHost, "http host (ip:port)")
    dataDir := flag.String("data", getPath(), "data files path")
    flag.Parse()
    api.DataStoragePath = *dataDir
    
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