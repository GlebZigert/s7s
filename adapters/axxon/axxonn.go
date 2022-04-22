package axxon

import (
	//    "log"
	//    "fmt"
	"context"
	"s7server/adapters/configuration"
	"s7server/api"
	//    "golang.org/x/net/websocket"
	"strings"
	"time"
	//  "strconv"
)

// Return all devices IDs for user filtering
func (svc *Axxon) GetList() []int64 {

	list := make([]int64, 0, len(svc.devList))

	svc.RLock()
	defer svc.RUnlock()

	for _, dev := range svc.devList {
		list = append(list, dev.id)
	}

	return list

}

func waiter(svc *Axxon) {
	svc.waiter_is_running=true
	ticker := time.NewTicker(1 * time.Second)
	var count = 0
	for _ = range ticker.C {

		select {
		case <-svc.quit_waiter:

			svc.Log("waiter done")
			svc.waiter_done <- true
			
			return

		default:

			count++
			if count == 1 {

				if svc.work == false{


			//	svc.Log("waiting for the Axxon")

				if !svc.test_http_connection() {
				//	svc.Log("Не удалось установить соединение с сервером Axxon!! Проверьте настройки!!")
				} else {

					svc.Log("Соединение установлено")
					svc.work = true
					svc.telemetrySessions = make(map[int]telemetrySession)
			
					svc.Api(map[string]api.Action{
						"ListDevices":       svc.listDevices,
						"request_URL":       svc.request_URL,
						"request_intervals": svc.request_intervals,
						"Telemetry_command": svc.Telemetry_command,
						"ExecCommand":       svc.execCommand})
			
					//go svc.SetTCPStatus("online")
					svc.SetServiceStatus(api.EC_SERVICE_ONLINE)
			
					//Обновляем список камер
					svc.devList_update()
			
					svc.websocket_is_connected = svc.websocket_connection()
			
					svc.Send_to_events_websocket()
			
					go svc.Take_axxon_events()
			
					go backgroundTask(svc)
				}	

				}


				count = 0
			}



		}
	}
}


func backgroundTask(svc *Axxon) {
	svc.background_is_running=true;
	ticker := time.NewTicker(1 * time.Second)
	var count = 0
	for _ = range ticker.C {

		select {
		case <-svc.quit_background:

			svc.background_done <- true
			return
		default:

			count++
			if count == 10 {

				svc.devList_update()

				svc.Broadcast("ListDevices", svc.make_devList_for_client())

				svc.Send_to_events_websocket()

				count = 0
			}

			if len(svc.telemetrySessions) > 0 {
				//          svc.Log("Количество сессий: ",len(svc.telemetrySessions))
				for idx, session := range svc.telemetrySessions {
					//              svc.Log("idx: ",idx,"; cid: ",session.cid,"; point: ",session.point,"; key: ",session.key,"; livetime: ",session.livetime)

					if session.livetime > 0 {
						//   session.livetime=session.livetime-1
						//   svc.hold_Session(session.point,string(session.key))
						svc.telemetrySessions[idx] = telemetrySession{cid: session.cid, point: session.point, key: session.key, livetime: session.livetime - 1}

					}

					if session.livetime == 0 {

						delete(svc.telemetrySessions, idx)

					}
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
	svc.username = svc.Settings.Login
	svc.password = svc.Settings.Password
	svc.ipaddr = strings.Split(svc.Settings.Host, ":")[0]
	svc.port = strings.Split(svc.Settings.Host, ":")[1]

	svc.background_is_running = false
	svc.eventHandler_is_running = false
	svc.waiter_is_running = false

	svc.quit_background = make(chan bool)
	svc.background_done = make(chan bool)

	svc.quit_eventHandler = make(chan bool)
	svc.eventHandler_done = make(chan bool)

	svc.quit_waiter = make(chan bool)
	svc.waiter_done = make(chan bool)

	//Проверяем соеднинение с сервером Axxon
	svc.work = false

	go waiter(svc)


	<-ctx.Done()
	svc.Log("<-ctx.Done()")
	//////////////////////////////////////////////////////////////


	if svc.work{
	
	//	svc.Log("1")
	if svc.background_is_running{
		svc.quit_background <- true
	}
	//svc.Log("2")	
	svc.conn.Close()
	//svc.Log("3")
	if svc.eventHandler_is_running{	
	svc.quit_eventHandler <- true
	}

	//svc.Log("4")
	if svc.background_is_running{
	<-svc.background_done
	//svc.Log("background_done")
	}
	//svc.Log("5")
	if svc.eventHandler_is_running{		
	<-svc.eventHandler_done
	//svc.Log("eventHandler_done")
	}

	//svc.Log("6")
	}

	//svc.Log("7")
	if svc.waiter_is_running{	
		svc.quit_waiter <- true
		<-svc.waiter_done
	//	svc.Log("waiter_done")
	}	
	

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
