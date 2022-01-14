package axxon

import (
//    "log"
   "fmt"
    "../../api"
    "../configuration"
	"strings"
    "time"  
  //  "strconv" 
)


func backgroundTask(svc *Axxon) {
    ticker := time.NewTicker(1 * time.Second)
    var count=0;
    for _ = range ticker.C {

count++;
if count==60{
fmt.Println("======================")
fmt.Println("=")
fmt.Println("=")
fmt.Println("[ОПРОС СПИСОК КАМЕР !!!!!!!!!!111]")

fmt.Println("=")
fmt.Println("=")
fmt.Println("======================")   
 svc.Get_list_devices()
 svc.Send_to_events_websocket()   





    count=0;
}
//fmt.Println("Количество сессий: ",len(svc.telemetrySessions))

    if len(svc.telemetrySessions)>0{
        fmt.Println("Количество сессий: ",len(svc.telemetrySessions))
        for idx, session := range svc.telemetrySessions {
            fmt.Println("idx: ",idx,"; cid: ",session.cid,"; point: ",session.point,"; key: ",session.key,"; livetime: ",session.livetime)

            if session.livetime>0{
         //   session.livetime=session.livetime-1
         //   svc.hold_Session(session.point,string(session.key))
            svc.telemetrySessions[idx]=telemetrySession{cid:session.cid, point:session.point,key : session.key,livetime:session.livetime-1}

            }

            if session.livetime==0{
            fmt.Println("Удаляю сессию idx: ",idx,"; cid: ",session.cid,"; point: ",session.point,"; key: ",session.key,"; livetime: ",session.livetime)   
            delete(svc.telemetrySessions,idx)   

            }
        }


        }

        //    fmt.Println("Tock")
    }



}


// Return all devices IDs for user filtering
func (cfg *Axxon) GetList() []int64 {
    return nil
}

func (svc *Axxon) Run() {
    svc.username = "func (svc *Axxon) Run()"
    svc.username = "User"
	svc.password = "12345"
	svc.username = "root"
	svc.password = "root"
//    svc.ipaddr   = "192.168.0.187"
 //   svc.port     = "8000"
    //svc.devices = make(map[int] Device)
    svc.username=svc.Settings.Login
    svc.password=svc.Settings.Password    
    svc.ipaddr   = strings.Split(svc.Settings.Host,":")[0]
    svc.port     = strings.Split(svc.Settings.Host,":")[1]
     
    fmt.Println("-----------------------------------------svc.username  ",svc.username)
    fmt.Println("-----------------------------------------svc.Settings.Host  ",svc.password)

    fmt.Println("-----------------------------------------svc.ipaddr  ",svc.ipaddr)
    fmt.Println("-----------------------------------------svc.port  ",svc.port)   

 //   fmt.Println("-----------------------------------------svc.Settings.Host  ",svc.Settings.Host)
 //   fmt.Println("-----------------------------------------svc.Settings.Host  ",svc.Settings.Host)


if !svc.test_http_connection(){
     fmt.Println("[Wrong connection settings]")     
    }else{


svc.telemetrySessions=make(map[int]telemetrySession)



   

    svc.Api(map[string] api.Action {
        "ListDevices" : svc.listDevices,
        "DateTime" : svc.DateTime,
        "ResetAlarm": svc.ResetAlarm,


        "request_intervals": svc.request_intervals,

        "request_URL": svc.request_URL,
        "request_URL_for_globalDeviceId": svc.request_URL_for_globalDeviceId,
    //    "Telemetry_capture_session" : svc.Telemetry_capture_session,
    //    "Telemetry_hold_session" : svc.Telemetry_hold_session,
 
    "Telemetry_command" : svc.Telemetry_command,

/*

        "Telemetry_move" : svc.Telemetry_move,

        "Telemetry_stop_moving" : svc.Telemetry_stop_moving, 

        "Telemetry_focus_in" : svc.Telemetry_focus_in,        
        "Telemetry_focus_out" : svc.Telemetry_focus_out,
        "Telemetry_stop_focus" : svc.Telemetry_stop_focus, 


        "Telemetry_zoom_in" : svc.Telemetry_zoom_in,        
        "Telemetry_zoom_out" : svc.Telemetry_zoom_out,
        "Telemetry_stop_zoom" : svc.Telemetry_stop_zoom,       
        "Telemetry_preset_info" : svc.Telemetry_preset_info,    
        "Telemetry_go_to_preset" : svc.Telemetry_go_to_preset,  
        "Telemetry_edit_preset" : svc.Telemetry_edit_preset,  
        "Telemetry_remove_preset" : svc.Telemetry_remove_preset, 
        "Telemetry_add_preset"  : svc.Telemetry_add_preset,   

*/

         "ExecCommand" : svc.execCommand })
    
    svc.cfg = svc.Configuration.(configuration.ConfigAPI)
  //  svc.devices = make([]Device)
    svc.stream_from_storage=make(map[int] Stream_from_storage)

    

 //   svc.getlistDevices()

 //   svc.getDateTime("20210429T081819")

    //go svc.SetTCPStatus("online")

    


svc.Get_list_devices()


svc.signal= make(chan string)


svc.websocket_is_connected=svc.websocket_connection()
svc.Send_to_events_websocket()

  fmt.Println("svc.websocket_is_connected ",svc.websocket_is_connected)  

go svc.Take_axxon_events()  

go backgroundTask(svc)


}
  //  svc.request_to_axxon("archive/list/"+"ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1")
}



func (svc *Axxon) Shutdown() {
    svc.Log("Shutting down...")
}


func catch(err error) {
    if nil != err {
        panic(err)
    }
}
