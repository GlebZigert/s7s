package sigur

import (
    "log"
    "net"
    "time"
    "context"
//	"strings"
//    "strconv"
    "../../api"
    "../configuration"
//    "../../mysql"
)

const dbPingInterval = 10 // seconds


func (sigur *Sigur) Run() {
    log.Println(sigur.GetName(), "Starting...")
    
    //sigur.Reply = reply
    sigur.initDB()
    
    var ctx context.Context
    ctx, sigur.Cancel = context.WithCancel(context.Background())
    go sigur.checkDB(ctx)
    go sigur.connect(ctx)
    
    sigur.Api(map[string] api.Action {
        "ListPersonal": sigur.listPersonal,
        "ListDevices": sigur.listDevices})
    
    //sigur.cfg = sigur.Subscribe()
    sigur.cfg = sigur.Configuration.(configuration.ConfigAPI)
    sigur.cfg.Subscribe()
    /*go func() {
        for msg := range c {
            log.Println("CHAN>", msg)
        }
    }()*/
}


func (sigur *Sigur) checkDB(ctx context.Context) {
    status := ""
    newStatus := ""

    for !sigur.Cancelled(ctx) {
        err := sigur.DB.PingContext(ctx)
        if nil != err {
            // TODO: change status after several failures?
            // TODO: errors.Is(err, os.ErrDeadlineExceeded)
            if _, ok := err.(*net.OpError); ok {
                newStatus = "offline"
            } else {
                newStatus = "error"
            }
        } else {
            newStatus = "online"
        }
        if status != newStatus {
            sigur.SetDBStatus(newStatus)
            if "online" == newStatus {
                sigur.syncDB()
                break
            } else {
                status = newStatus                
                log.Println(sigur.GetName(), err)
            }
        }
        sigur.Sleep(ctx, time.Duration(dbPingInterval) * time.Second)
    }
    log.Println(sigur.GetName(), "DB connected.")
}

// if trees aren't equal - reconnect tcp data stream
// TODO: don't reconnect, just get APINFO again?
/*func (sigur *Sigur) dbConnected() {
    devices := sigur.dbListDevices()
    if sigur.updateRequired(devices) {
        log.Println(sigur.GetName(), "DB connected. Devices update required")
        sigur.devices = devices
        sigur.resetConnection()
    } else {
        log.Println(sigur.GetName(), "DB connected. Devices are up-to-date")
    }
}*/

// compare current and new devices list 
/*
func (sigur *Sigur) updateRequired(newDevices map[int] Device) bool {
    var equal bool
    if len(sigur.devices) != len(newDevices) {
        return true
    }
    for k, _ := range newDevices {
        if _, ok := sigur.devices[k]; ok {
            // let's compare field-by-field
            equal = newDevices[k].Id == sigur.devices[k].Id && 
                    newDevices[k].ParentId == sigur.devices[k].ParentId &&
                    newDevices[k].Type == sigur.devices[k].Type
            if false == equal {
                break
            }
        }
    }
    return !equal
}
*/
func (sigur *Sigur) syncDB() {
    // get list of users with related devices and access rules
    //sigur.cfg <- "listUsers"
        
    // 1. Get User-Device pairs
    // 2. Get User-Rule pairs
    // 3. Load Users from DB and compare against Sigur, update Sigur if needed
    // 4. Load Rules from DB and compare against Sigur, update Sigur if needed
    // 5. Checkout user-devices relations, update Sigur if needed
    // 6. Checkout user-rules relations, update Sigur if needed
    
    
    sigur.cfg.UsersWithLinks(2) // TODO: sigur.cfg.ListUsers(sigur.Settings.Id)
    /*rules := */sigur.loadRules()
    //log.Println("::: SR>", rules.Name)
}

func (sigur *Sigur) Shutdown() {
    log.Println(sigur.GetName(), "Shutting down...")
    //sigur.cfg <- "unsubscribe"
    sigur.Cancel() // use it before reset connection!
    sigur.resetConnection()
}
