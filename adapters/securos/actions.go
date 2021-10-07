package securos

import (
//    "log"
    //"encoding/json"
)

/****************** RIF+  Actions ******************/

func (svc *Securos) listDevices(cid int64, data []byte) (interface{}, bool) {
    /*res, err := json.Marshal(rif.devices)
    if err != nil {
        //log.Println(err)
        return err.Error()
    } else {
        return "{\"" + rif.Name + "\": " + string(res) + "}"
    }*/
    //log.Println(rif.Name, " - ", len(rif.devices), "devices")
    return svc.devices, false
}