package parus

import (
    "time"
    "math/rand"
    "context"
    "encoding/xml"
    "encoding/json"
    "bytes"
    "strings"
    
    "golang.org/x/net/html/charset"

    "s7server/api"
    "s7server/adapters/configuration"
)

const (
    pollInterval = 3 // seconds
    errThreshold = 3
)

var core configuration.ConfigAPI

func (svc *Parus) Run(_ configuration.ConfigAPI) (err error) {
    configuration.ExportCore(&core)
    var ctx context.Context
    ctx, svc.Cancel = context.WithCancel(context.Background())
    svc.Stopped = make(chan struct{})
    defer close(svc.Stopped)

    //svc.complaints = make(chan error, 10)
    //go svc.ErrChecker(ctx, svc.complaints, api.EC_SERVICE_READY, api.EC_SERVICE_FAILURE)

    rand.Seed(time.Now().UnixNano())
    err = svc.loadDevices()
    if nil != err {
        return
    }
    
    go svc.pollDevices(ctx)
    
    svc.setupApi()
    svc.SetServiceStatus(api.EC_SERVICE_READY)
    
    <-ctx.Done()
    //////////////////////////////////////////////////////////////////

    svc.Log("Shutting down...")
    svc.SetServiceStatus(api.EC_SERVICE_SHUTDOWN)
    return
}

func (svc *Parus) Shutdown() {
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
func (svc *Parus) GetList() []int64 {
    svc.RLock()
    defer svc.RUnlock()

    list := make([]int64, 0, len(svc.devices))
    
    for id := range svc.devices {
        list = append(list, id)
    }

    return list
}


func (svc *Parus) pollDevices(ctx context.Context) {
    for !svc.Cancelled(ctx) {
        svc.RLock()
        for _, dev := range svc.devices {
            go svc.queryStatus(dev)
        }
        svc.RUnlock()
        svc.Sleep(ctx, time.Duration(pollInterval * time.Second))
    }
}


func (svc *Parus) queryStatus(dev *Device) {
    var stateClass int64
    
    svc.RLock()
    url := "http://" + dev.IP + "/upsstatus.xml"
    svc.RUnlock()
    xmlFile, err := getRequest(url)
    if nil != err {
        //svc.Log("HTTP ERR:", err)
        stateClass = api.EC_LOST
    } else {
        data := new(UPSStatus)
        reader := bytes.NewReader(xmlFile)
        decoder := xml.NewDecoder(reader)
        decoder.CharsetReader = charset.NewReaderLabel
        err = decoder.Decode(&data)
        
		//err = xml.Unmarshal(xmlFile, data)
        if nil != err {
            svc.Log("XML ERR", err)
            stateClass = api.EC_ERROR
        } else {
            //svc.Log("DATA:", data.Input)
            if strings.Index(data.Input, "Normal") >= 0 {
                stateClass = api.EC_UPS_PLUGGED
            } else if strings.Index(data.Input, "On Battery") >= 0 {
                stateClass = api.EC_UPS_UNPLUGGED
            }
            
        }
    }
    svc.analyzeStatus(dev, stateClass)
}

func (svc *Parus) analyzeStatus(dev *Device, stateClass int64) {
    var events api.EventsList
    var newClass int64
    svc.Lock()
    if stateClass != dev.StateClass {
        if stateClass == api.EC_ERROR || stateClass == api.EC_LOST {
            dev.StateCounter += 1
            if errThreshold == dev.StateCounter {
                newClass = stateClass
                dev.StateCounter = 0
            }
        } else {
            newClass = stateClass
        }
    }
    if newClass > 0 {
        dev.StateClass = newClass
        dev.StateText = api.DescribeClass(newClass)
        events = append(events, api.Event{
            Class: newClass,
            DeviceId: dev.Id,
            DeviceName: dev.Name})
    }
    svc.Unlock()
    if nil != events {
        svc.Broadcast("Events", events)
    }
}

func (svc *Parus) loadDevices() (err error) {
    svc.Lock()
    svc.devices = make(map[int64] *Device)
    devices, err := core.LoadDevices(svc.Settings.Id) // TODO: handle err
    if nil != err {
        return
    }
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
    return
}

func (svc *Parus) setupApi() {
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