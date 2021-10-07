package securos

import (
//    "log"
//    "fmt"
    "context"
    "../../api"
//	"strings"
)

func (svc *Securos) Run() {
    //svc.devices = make(map[int] Device)

    svc.Api(map[string] api.Action {
        "ListDevices" : svc.listDevices})
    
//    go svc.SetTCPStatus("online")
    var ctx context.Context
    ctx, svc.Cancel = context.WithCancel(context.Background())
    go svc.connect(ctx)
}


func (svc *Securos) Shutdown() {
    svc.Log("Shutting down...")
}


func catch(err error) {
    if nil != err {
        panic(err)
    }
}