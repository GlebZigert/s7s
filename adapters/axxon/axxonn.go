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

func backgroundTask(svc *Axxon) {
	ticker := time.NewTicker(1 * time.Second)
	var count = 0
	for _ = range ticker.C {

		select {
		case <-svc.quit:

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

	svc.quit = make(chan bool)
	svc.background_done = make(chan bool)

	svc.quit_eventHandler = make(chan bool)
	svc.eventHandler_done = make(chan bool)

	//Проверяем соеднинение с сервером Axxon
	res:=false
	if !svc.test_http_connection() {
		svc.Log("Не удалось установить соединение с сервером Axxon!! Проверьте настройки!!")
	} else {
		svc.Log("Соединение установлено")
		res=true
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
		
	<-ctx.Done()
	//////////////////////////////////////////////////////////////
	if res{
	
	svc.quit <- true

	
	
		
	svc.conn.Close()

	svc.quit_eventHandler <- true


	<-svc.background_done

	<-svc.eventHandler_done

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
