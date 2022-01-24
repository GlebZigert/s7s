package axxon

import (
//    "log"
    "fmt"
    "context"
     "../../api"
    "../configuration"
	  "strings"
    "time"  
    //  "strconv" 
)


// Return all devices IDs for user filtering
func (svc *Axxon) GetList() []int64 {
  
  list := make([]int64, 0, len(svc.devList))
    
  svc.RLock()
  defer svc.RUnlock()
  
  for _,dev := range svc.devList {
      list = append(list, dev.id)
  }

  return list

}

func backgroundTask(svc *Axxon) {
    ticker := time.NewTicker(1 * time.Second)
    var count=0;
    for _ = range ticker.C {

        count++;
        if count==10{

        

          fmt.Println(" ")
          fmt.Println(" ")
          fmt.Println("ОПРОС СПИСОК КАМЕР ")

          fmt.Println(" ")
          fmt.Println(" ")

        
          svc.devList_update()

          svc.Broadcast("ListDevices", svc.make_devList_for_client())   

          svc.Send_to_events_websocket()

          count=0;
        }

        if len(svc.telemetrySessions)>0{
//          fmt.Println("Количество сессий: ",len(svc.telemetrySessions))
          for idx, session := range svc.telemetrySessions {
//              fmt.Println("idx: ",idx,"; cid: ",session.cid,"; point: ",session.point,"; key: ",session.key,"; livetime: ",session.livetime)
  
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








    }
}



func (svc *Axxon) Run(cfg configuration.ConfigAPI) (err error) {
  var ctx context.Context
  ctx, svc.Cancel = context.WithCancel(context.Background())
  svc.cfg = cfg
  svc.Stopped = make(chan struct{})
  defer close(svc.Stopped)



  //Задаем логин пароль апи порт сервера Aххоn  
  svc.username=svc.Settings.Login
  svc.password=svc.Settings.Password    
  svc.ipaddr   = strings.Split(svc.Settings.Host,":")[0]
  svc.port     = strings.Split(svc.Settings.Host,":")[1]


//Проверяем соеднинение с сервером Axxon
  if !svc.test_http_connection(){
    fmt.Println("Не удалось установить соединение с сервером Axxon!! Проверьте настройки!!")     
  }else{
    fmt.Println("Соединение установлено") 

    svc.telemetrySessions=make(map[int]telemetrySession)

    svc.Api(map[string] api.Action {
      "ListDevices" : svc.listDevices,
      "request_URL": svc.request_URL, 
      "request_intervals": svc.request_intervals ,
      "Telemetry_command" : svc.Telemetry_command,
      "ExecCommand" : svc.execCommand})

    

    //go svc.SetTCPStatus("online")
    svc.SetServiceStatus(api.EC_SERVICE_ONLINE)

    //Обновляем список камер
    svc.devList_update()


    svc.websocket_is_connected=svc.websocket_connection()

    svc.Send_to_events_websocket()
    

    go svc.Take_axxon_events() 

    go backgroundTask(svc)


  }

  <-ctx.Done()
  //////////////////////////////////////////////////////////////
  
  svc.Log("Shutting down...")
  svc.SetServiceStatus(api.EC_SERVICE_SHUTDOWN)
  return

  return
}



func (svc *Axxon) Shutdown() {
  svc.RLock()
  ret := nil == svc.Cancel || nil == svc.Stopped
  svc.RUnlock()
  if ret {
      return
  }

  svc.Cancel()
  <-svc.Stopped
}


func catch(err error) {
 if nil != err {
  panic(err)
 }
}
