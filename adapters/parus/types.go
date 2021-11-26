package parus

import (
    "sync"
    "encoding/xml"
    "../../api"
    "../configuration"
)

//type Reply func (int, string, string, interface{})

type Parus struct {
    sync.RWMutex
    api.API
    cfg     configuration.ConfigAPI
    
    devices         map[int64] *Device
}


type DeviceData struct {
    IP    string       `json:"ip"`
}

type Device struct {
    configuration.Device
    DeviceData

    AccessMode      int64       `json:"accessMode"` // hint for GUI
    StateClass      int64       `json:"stateClass"`
    StateCounter    int64
    StateText       string      `json:"stateText"`
}

type UPSStatus struct {
    XMLName     xml.Name `xml:"upsstatus"`
    Input       string    `xml:"input"`
}


type DevList []Device // for filtering

func (devices DevList) Filter(list map[int64]int64) interface{} {
    var res DevList
    for i := range devices {
        // list[0] > 0 => whole service accessible
        devices[i].AccessMode = list[0] // TODO: || list[devices[i].Id]
        if 0 == devices[i].AccessMode {
            devices[i].AccessMode = list[devices[i].Id]
        }
        if devices[i].AccessMode > 0 {
            res = append(res, devices[i])
        }
    }
    if len(res) > 0 {
        return res
    } else {
        return nil
    }
}

