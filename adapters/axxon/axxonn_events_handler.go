package axxon

import (
	"fmt"
	"net/http"
//	"log"
	//"net/url"
    "../../api"
    "time"      
	"golang.org/x/net/websocket"
//    "strconv"    
    "os"
    "net"
)

func (svc *Axxon)  websocket_connection() (res bool) {
	fmt.Println("Устанавливаем соединение по вебсокету с видеосервером",)
	var axxonn, myaddr string
	
	axxonn=  "ws://"+svc.ipaddr+":"+svc.port+"/events"
	
	//localHost, _ := os.Hostname()
	
	
	name, err := os.Hostname()
	if err != nil {
		 fmt.Printf("Oops: %v\n", err)
		 return
	}
	fmt.Println("name:", name)
	addrs, err := net.LookupHost(name)
	if err != nil {
		fmt.Printf("Oops: %v\n", err)
		return
	}
	fmt.Println("addrs:", addrs)
	
	for _, a := range addrs {
		fmt.Println(a)
	}  
	
	
	 tt, err := net.Interfaces()
	  if err != nil { 
		panic(err)  
	  }     
	  for _, t := range tt {
		aa, err := t.Addrs()
		if err != nil {
		  panic(err)        
		}           
		for _, a := range aa {
		  ipnet, ok := a.(*net.IPNet) 
		  if !ok {          
			continue                
		  }                 
		  v4 := ipnet.IP.To4() 
		  if v4 == nil || v4[0] == 127 { // loopback address
			continue                
		  }                 
		  fmt.Printf("%v\n", v4)
		  myaddr="http://"+v4.String()
		  res:=svc.Init_event_websocket(axxonn,myaddr)
		  if res==true{
			return true
		  }
	
	
		}           
		 
	  }  
	  return false   
	}

func (svc *Axxon)  Init_event_websocket(axxonn,myaddr string) (res bool) {

		fmt.Println("[Init_event_websocket]")
		fmt.Println("axxonn: ",axxonn)
		fmt.Println("myaddr: ",myaddr)
		
		config, err := websocket.NewConfig(axxonn, myaddr)
		if err != nil {
		fmt.Println(err)
		}
		
		req, _ := http.NewRequest("", "", nil)
		req.SetBasicAuth(svc.username, svc.password)
		username,password,_:=req.BasicAuth()
		fmt.Println("username: ",username)
		fmt.Println("password: ",password)
		
		  config.Header = http.Header {
			"Authorization" : req.Header["Authorization"],
		}
		
		
		
		fmt.Println("req.Header[Authorization]: ",req.Header["Authorization"])
		fmt.Println("config.Header: ",config.Header)
		
		svc.conn, err= websocket.DialConfig(config)
		if err != nil {
		fmt.Println(err)
		}else{

			fmt.Println("Соединение по вебсокету установлено")
		
		return true
		}
		
		return false
}	


func (svc *Axxon)  Take_axxon_events() {

			if svc.websocket_is_connected{
		  
		  
			  type Receive struct {
			  Objects []struct {
			  Name  string `json:"name"`
			  State string `json:"state"`
			  Type  string `json:"type"`
			  } `json:"objects"`
			  }
		  
			  //http.ListenAndServe("ws://192.168.0.187:8000/events",  websocket.Handler(My_Handler))
		  
		  
			  var rcv Receive
			  var event_class int64
		  
			  //цикл
			  for {
		  
				if err:=websocket.JSON.Receive(svc.conn,&rcv);err!=nil{
				  fmt.Println(err)  
				}else{

					/*
				  fmt.Println(" \n")
				  fmt.Println(" \n")
				  fmt.Println("[1] ")
				  fmt.Println(" ")
				  fmt.Println("[!!]rcv: ",rcv)
				  fmt.Println(" ")
		  */
					//если есть принятые сообщения от Axxon
					if len(rcv.Objects)>0{
		  
					  //создаем структуру лист событий в которую будем их собирать
					  ee := make(api.EventsList, 0, 1)
		  
					  //для каждого из наших устройств
					  for j:=0;j<len(svc.devList);j++    {
		  
						//берем fccesspoint устройства по которому будем сверять событие с сервера Axxon
						point:="hosts/"+svc.devList[j].pointer.VideoStreams[0].AccessPoint
						  //    fmt.Println("point: ",point)
		  
						//gперебираем все полученные события
						for i := 0; i < len(rcv.Objects); i++ {
		  
						  //По accesspoint проверяем, для этой ли камеры это событие
						  if rcv.Objects[i].Name==point{
						//	fmt.Println("[2] ")
		  
							//Далее в зависимости от типа события
							if rcv.Objects[i].Type=="devicestatechanged"{
							//  fmt.Println("[2][1] ")
		  
							  //     fmt.Println("обьект: ",rcv.Objects[i].Name,"; тип: ",rcv.Objects[i].Type,"; состояние: ",rcv.Objects[i].State)
							  //Добавь изменения в listDevices
							  //    fmt.Println(" ")  
							  //    fmt.Println("Камера ",point," Изменила состояние на ",rcv.Objects[i].State)  
							  //    fmt.Println(" ") 
		  
							  prev_dev_state:=svc.devList[j].state
		  
							  var current_dev_state string
							  var text string
		  
		  
							  if rcv.Objects[i].State=="signal restored"{
								current_dev_state="ok"
								event_class=api.EC_OK
								text="Сигнал восстановлен"
							  } 
							  if rcv.Objects[i].State=="signal lost"{
								current_dev_state="lost"
								event_class=api.EC_LOST
								text="Сигнал потерян"                
							  } 
							  var id int64
							  id=svc.devList[j].id
							  fmt.Println("id: ",id)
		  

	//	  fmt.Printf("[current_dev_state] ",current_dev_state)
									  
	//									  fmt.Printf("[prev_dev_state]", prev_dev_state)
							  if current_dev_state!=prev_dev_state{
		  
							
								
								fmt.Printf("Изменение состояния камера  ",svc.devList[j].pointer.DisplayName)		  
										  
							//				  fmt.Printf("[APPEND]")
								//Оформляем событие нужным нам образом и добавляем в лист событий
								ee = append(ee, api.Event {
											ExternalId: 1,
											Event:      event_class,
											Class:      event_class,
											Text:       text,
											ServiceId:  svc.Settings.Id,
											DeviceId:   svc.devList[j].id,
											Reason:     "NO REASON",
											DeviceName: svc.devList[j].pointer.DisplayName,
											ServiceName: svc.Settings.Title,
											Reaction:   "REACTION",
											Time:       parseTime(time. Now().String()).Unix()})
							  }
		  
							  if svc.devList[j].state=="alarm"{
		  
							//	fmt.Println("[Меняем состояние alarm - то есть ненароком сбрасываем тревогу] ")  
		  
		  
							  }
		  
							  svc.devList[j].state=current_dev_state
		  
							}
		  
							//если это сигнала - тревога (c видеоаналитики сервера по этой камере)
							//if rcv.Objects[i].Type=="alert"||rcv.Objects[i].Type=="detector_event"{
							//if rcv.Objects[i].Type=="alert"&&rcv.Objects[i].State=="on"{
							if rcv.Objects[i].Type=="alert"{  
		  
		  
							svc.devList[j].state="alarm"                  
							// if rcv.Objects[i].Type=="alert"||rcv.Objects[i].Type=="alert_state"{  
							  fmt.Println("[2][2] ")      
							  fmt.Println(" \n")
		  
		  //Берем айди камеры и текущее время                    
							  dt := time. Now()
							  fmt.Printf("dt.type:  %T\n", dt)
							  fmt.Println("Name ",rcv.Objects[i].Name," id: ",svc.devList[j].id," dt: ",dt.String())
		  
						 
		  
						   //   svc.alerts.find(svc.devices[j].Id,dt)
							//
						  
		  
						  //  svc.alerts=append(svc.alerts,alert{id:1,dt:dt})
		  
		  
						//    fmt.Println("svc.alerts: ",len(svc.alerts))
		  
		  
							  fmt.Println(" \n") 
							  fmt.Println("Name ",rcv.Objects[i].Name)                                       
							  fmt.Println("State ",rcv.Objects[i].State)
							  fmt.Println("Type ",rcv.Objects[i].Type)
							  fmt.Println(" \n")
		  
		  //Проверка: если на эту секунду уже была тревога с этой камеры - не пишем.
		  //list с айдишниками камер и временм последней тревоги.
			var result bool
			result=svc.find_alert(svc.devList[j].id,dt)
		  
			fmt.Printf("result: %t",result)
		  
			if result==false{
						  fmt.Printf("[APPEND]")
							  ee = append(ee, api.Event {
										  ExternalId: 1,
										  Event:      2,
										  Class:      api.EC_ALARM,
										  Text:       "ТРЕВОГА",
										  ServiceId:  svc.Settings.Id,
										  DeviceId:   svc.devList[j].id,
										  Reason:     "NO REASON",
										  DeviceName: svc.devList[j].pointer.DisplayName,
										  ServiceName: svc.Settings.Title,
										  Reaction:   "REACTION",
										  Time:       parseTime(time. Now().String()).Unix()})
										}
		  
							}
		  
							/*
									Name  string `json:"name"`
								State string `json:"state"`
								Type  string `json:"type"`
							*/
						  }
						}
						//   fmt.Println(" ")
		  
						if err !=nil{
						fmt.Println("ERROR: ",err)
						}
						  
					  }
		  
				//	 fmt.Println("len(ee) ",len(ee))
					  if len(ee)>0{
		  
				   
							 svc.cfg.ImportEvents(ee)
					  svc.Broadcast("Events", ee)
					  }
						  
					}
		  
					//А нужна ли вот эта строка???
		  
					//Не шли лист дева йс - шли только тот элемент и то свойство которое менялось. ???
			//		svc.Broadcast("ListDevices", svc.devList)   
			svc.devList_update()

			svc.Broadcast("ListDevices", svc.make_devList_for_client())   
  
			//	fmt.Println(" \n")
			//	fmt.Println(" \n")
				}
		  
		  
		  
		  
				//---------
			  }
			}
			  
}
var username,password,ipaddr,port string
const (

    dtLayout = "2006-01-02 15:04:05"
)
func parseTime(s string) time.Time {
    loc := time.Now().Location()
    dt, err := time.ParseInLocation(dtLayout, s, loc)
    if nil != err {
        // TODO: log err?
        dt = time.Now()
    }
    return dt
}

func (svc *Axxon)  Send_to_events_websocket() {

	fmt.Println("[Send_to_events_websocket]")
	if svc.websocket_is_connected{
	
	type Message struct {
	  Include []string `json:"include"`
	  Exclude []string `json:"exclude"`
	
	}
	var msg Message
	for i:=0;i<len( svc.devList);i++{
	point:=svc.devList[i].pointer.VideoStreams[0].AccessPoint
	
	msg.Include=append(msg.Include,point)
	}
	fmt.Println("msg.Include: ",msg.Include)
	
	  //  msg.Include  = []string{"hosts/ASTRAAXXON/DeviceIpint.15/SourceEndpoint.video:0:0",
	  //              "hosts/ASTRAAXXON/DeviceIpint.16/SourceEndpoint.video:0:0"}
	fmt.Println("Message",msg)
	if err:=websocket.JSON.Send(svc.conn,msg);err!=nil{
	fmt.Println(err)  
	}
	
	}
	} 