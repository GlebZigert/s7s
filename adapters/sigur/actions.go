package sigur

import (
//    "log"
    "net"
    //"database/sql"
//    "fmt"
//    _ "../../mysql"
)

// TODO: move it to API
func (sigur *Sigur) dbFailure() {
    if r := recover(); r != nil {
        //fmt.Println(r.(error))
        //err0 = r.(error)
        if _, ok := r.(*net.OpError); ok {
            sigur.SetDBStatus("offline")
        } else {
            sigur.SetDBStatus("error")
        }
        //log.Println(sigur.GetName(), r.(error))
    }
}


func (sigur *Sigur) listPersonal(cid int64, data []byte) (interface{}, bool) {
    defer sigur.dbFailure()
    list := sigur.dbListPersonal()
    return list, false
}

func (sigur *Sigur) listDevices(cid int64, data []byte) (interface{}, bool) {
    defer sigur.dbFailure()
    list := sigur.dbListDevices()
    return list, false
}