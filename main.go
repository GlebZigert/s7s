package main
 
import (
    "os"
    "log"
    //"time"
    "context"
    "syscall"
    "os/signal"

    "./dispatcher"
)

const host = "0.0.0.0:2973"

func main() {
    var ctx context.Context
    //log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
    ctx, cancel := context.WithCancel(context.Background())
    
    c := make(chan os.Signal, 1) // use buffer (size = 1), so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
    go func() {
        <-c
        log.Println("Let's stop")
        cancel()
	}()
    
    dispatcher.Run(ctx, host)
    //time.Sleep(1 * time.Second)
}

