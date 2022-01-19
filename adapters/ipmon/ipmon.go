package ipmon

import (
    "time"
    "math/rand"
    "context"
    "encoding/json"

    "../../api"
    "../configuration"
)

const (
    pollInterval = 5 // seconds
    errThreshold = 3
)

func (svc *IPMon) Run(cfg configuration.ConfigAPI) (err error) {
    var ctx context.Context
    ctx, svc.Cancel = context.WithCancel(context.Background())
    svc.cfg = cfg
    svc.Stopped = make(chan struct{})
    defer close(svc.Stopped)
    
    
    rand.Seed(time.Now().UnixNano())
      
    svc.loadDevices()
    
    go svc.pollDevices(ctx)
    
    svc.setupApi()
    svc.SetServiceStatus(api.EC_SERVICE_READY)
    
    <-ctx.Done()
    //////////////////////////////////////////////////////////////
    
    svc.Log("Shutting down...")
    svc.SetServiceStatus(api.EC_SERVICE_SHUTDOWN)
    return
}

func (svc *IPMon) Shutdown() {
    svc.RLock()
    ret := nil == svc.Cancel || nil == svc.Stopped
    svc.RUnlock()
    if ret {
        return
    }

    svc.Cancel()
    <-svc.Stopped
}

// Return all devices IDs for user filtering
func (cfg *IPMon) GetList() []int64 {
    return nil
}

func (svc *IPMon) pollDevices(ctx context.Context) {
    var states = []int64{api.EC_CONNECTION_LOST, api.EC_CONNECTION_OK}
    for !svc.Cancelled(ctx) {
        svc.Lock()
        var events api.EventsList
        for _, dev := range svc.devices {
            stateClass := states[rand.Intn(len(states))]
            if dev.StateClass != stateClass {
                dev.StateClass = stateClass
                dev.StateText = api.DescribeClass(stateClass)
                events = append(events, api.Event{
                    Class: stateClass,
                    DeviceId: dev.Id,
                    DeviceName: dev.Name})
            }
        }
        svc.Unlock()
        if nil != events {
            svc.Broadcast("Events", events)
        }
        svc.Sleep(ctx, time.Duration(10 + rand.Intn(10)) * time.Second)
    }
}

func (svc *IPMon) loadDevices() {
    svc.Lock()
    svc.devices = make(map[int64] *Device)
    devices, _ := svc.cfg.LoadDevices(svc.Settings.Id) // TODO: handle err
    for i := range devices {
        dev := Device{Device: devices[i]}
        if "" != dev.Data {
            json.Unmarshal([]byte(dev.Data), &dev.DeviceData) // TODO: handle err
            dev.Data = ""
        }
        dev.StateClass = api.EC_NA
        svc.devices[dev.Id] = &dev
    }
    svc.Unlock()
    //svc.Log(":::::::::::::::::", len(svc.devices), "DEVICES LOADED for service", svc.Settings.Id)
}

func (svc *IPMon) setupApi() {
    svc.Api(map[string] api.Action {
        "ResetAlarm" : svc.resetAlarm,
        
        "ListDevices" : svc.listDevices,
        "DeleteDevice" : svc.deleteDevice,
        "UpdateDevice" : svc.updateDevice})
}

func catch(err error) {
    if nil != err {
        panic(err)
    }
}