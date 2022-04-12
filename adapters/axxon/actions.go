package axxon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"s7server/api"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Проверка на соединение с сервером Axxon
//Возвращает true - если соединение установлено; иначе - false
// 1. Отправить на сервер Axxon запрос на получение уникального идентификататора uuid
//    (согласно пункут SDK интеграции п.1.2.1.1. Получение уникального идентификатора)
// 2.Проконтролировать получение ответа
// 2.1. Если овтета нет - вернуть false
// 3. При наличии ответа проконтролировать его содерживое на соответствие ожидамемому.

/*
	{
	"uuid":""

	}
*/

// 3.1. Если соответствует - вернуть true. иначе - false

func (svc *Axxon) test_http_connection() bool {

	fmt.Println("Контроль возможности подключения к серверу : ")

	fmt.Println("логин:  ", svc.username)
	fmt.Println("пароль: ", svc.password)
	fmt.Println("ip:     ", svc.ipaddr)
	fmt.Println("порт:   ", svc.port)

	//Формируем строку запроса
	request := "http://" + svc.username + ":" + svc.password + "@" + svc.ipaddr + ":" + svc.port + "/" + "uuid"

	fmt.Println("Сформирована строка запроса: ",request)

	//Отправляем запрос
	resp, err := http.Get(request)
	if err != nil {
		fmt.Println("err: ", err)
		return false
	}

	defer resp.Body.Close()

	//Читаем ответ
	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("err: ", err)
		return false
	}

	bodyString := string(bodyBytes)
//	fmt.Println("Получен ответ: ", bodyString)

	type uuid_struct struct {
		UUID string `json:"uuid"`
	}

	var m_uuid uuid_struct

	err = json.Unmarshal([]byte(bodyString), &m_uuid)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	return true

}

/*Список камер - обновление твоей внутренней структуры списка камер.
Запрашиваешь у Ахона список камер.
Сверяешь с имеющимся у тебя списком следующим образом.

Если у тебя нет такой камеры - добавляешь ее.

Если у тебя есть такая камера - оставляешь ее.

Если у тебя есть камеры - а в новом списке ее нет - удаляешь ее.
*/

var lock sync.Mutex

func (svc *Axxon) devList_update() {

	//Блокируем запись и чтение из других потоков
	svc.Lock()
	defer svc.Unlock()
	//fmt.Println("BEGIN devList_update")

	lock.Lock()
	defer lock.Unlock()

	err := svc.get_cameraList()

	if err != nil {
		fmt.Println(err)
	} else {
		//		fmt.Println("[PROFIT]")

		for j := 0; j < len(svc.devList); j++ {
			svc.devList[j].actual = false

		}

		for i := 0; i < len(svc.cameraList.List); i++ {

			need_to_add := true
			for j := 0; j < len(svc.devList); j++ {

				if svc.devList[j].VIDEOSOURSEID == svc.get_VIDEOSOURCEID(&svc.cameraList.List[i]) {
					need_to_add = false

					svc.devList[j].pointer = &svc.cameraList.List[i]
					svc.devList[j].state = svc.get_state(&svc.cameraList.List[i])
					svc.devList[j].actual = true
				}
			}

			if need_to_add {
				svc.devList_add(&svc.cameraList.List[i])
			}

		}

		var new_devList []dev
		new_devList = nil
		for j := 0; j < len(svc.devList); j++ {

			if svc.devList[j].actual {
				new_devList = append(new_devList, svc.devList[j])
			}

		}

		svc.devList = nil
		svc.devList = new_devList

		/*
		fmt.Println("Список камер:")
		for j := 0; j < len(svc.devList); j++ {
			fmt.Println(j, ": ", svc.devList[j].VIDEOSOURSEID)

		}
		*/

	}
	//Для тестов
	/*
		ticker := time.NewTicker(1 * time.Second)
		var count=0;
		for _ = range ticker.C {
			count++;
			fmt.Println("count ",count)
			if count==6{
				ticker.Stop()
				break
			}
		}
	*/

	//fmt.Println("END devList_update")

}

func (svc *Axxon) devList_add(camera *Camera) {
	VIDEOSOURCEID := svc.get_VIDEOSOURCEID(camera)
	TelemetryControlID := svc.get_TelemetryControlID_from(camera)
	fmt.Println("добавляю камеру: ", VIDEOSOURCEID)

	xx, _ := svc.cfg.GlobalDeviceId(svc.Settings.Id, VIDEOSOURCEID, camera.DisplayName)

	svc.devList = append(svc.devList, dev{id: xx,
		pointer:            camera,
		VIDEOSOURSEID:      VIDEOSOURCEID,
		TelemetryControlID: TelemetryControlID,
		actual:             true,
		state:              svc.get_state(camera)})
}

func (svc *Axxon) get_VIDEOSOURCEID(camera *Camera) string {

	return camera.VideoStreams[0].AccessPoint

}

//Запрашивает с с сервера Ахон список камер
//Пишет в структуру svc.m_camera_list
func (svc *Axxon) get_cameraList() error {

	src, err := svc.request_to_axxon("camera/list/")

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	err = json.Unmarshal([]byte(src), &svc.cameraList)
	if err != nil {
		fmt.Println(err.Error())

		return err
	}

	return err

}

//Запрос серверу Ахон - строка запроса в аргументе
//Возвращает ответ. и флаг.
//Если нет ответа - выставляешь плохой флаг.
//Должен поменяться цвет значка сервера.
//Должно записаться событие -пропажа связи с сервером Ахон

func (svc *Axxon) control_connection_status(res bool) {

	//	fmt.Println("статус: ",api.ClassText[svc.Settings.Status.TCP])

	if res == false {
		fmt.Println("[ERROR  http.Get(request)]")
		svc.SetServiceStatus(api.EC_SERVICE_OFFLINE)

	} else {

		svc.SetServiceStatus(api.EC_SERVICE_ONLINE)
	}
}

func (svc *Axxon) request_to_axxon(req string) (string, error) {
	request := svc.msg_to_axxon(req)
	//fmt.Println("запрос: ", request)

	resp, err := http.Get(request)

	var res bool

	if err == nil {
		res = true
		svc.control_connection_status(res)
	} else {
		res = false
		svc.control_connection_status(res)
		return "", err
	}

	if err != nil {

		//Вывести актуальный к этому моменту статус видеосервера RIF

		return "", err
	}

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Println("[ioutil.ReadAll(resp.Body)]")
		return "", err
	}
	bodyString := string(bodyBytes)
	//fmt.Println("ответ:  ", bodyString)

	return bodyString, nil

}

func (svc *Axxon) msg_to_axxon(request string) string {

	return "http://" + svc.username + ":" + svc.password + "@" + svc.ipaddr + ":" + svc.port + "/" + request

}

func (svc *Axxon) listDevices(cid int64, data []byte) (interface{}, bool) {

	//	fmt.Println("[listDevices]")
	//из devList формируешь структуру для передачи в  JSON

	res := svc.make_devList_for_client()

	//	fmt.Println("[END][listDevices]")
	return res, false

}

func (svc *Axxon) convert_for_client(dev *dev) Device {

	var device Device

	device.Sid = svc.Settings.Id
	device.Id = dev.id
	device.Name = dev.pointer.DisplayID + dev.pointer.DisplayName
	device.Stream = svc.get_RTSP(dev.pointer)

	device.State = dev.state

	device.Frash_Snapshot = svc.get_frash_snapshot_from(dev.pointer)

	device.TelemetryControlID = dev.TelemetryControlID
	device.Intervals = svc.get_intervals_from(dev.pointer)

	//fmt.Println("convert for client: ",dev.pointer.VideoStreams[0].AccessPoint)

	return device
}

/*
func (svc *Axxon) get_intervals_from(camera *Camera) intervals{

	point:=camera.VideoStreams[0].AccessPoint



	var src string
	src,_ = svc.request_to_axxon("archive/contents/intervals/"+strings.Replace(point,"hosts/","",1)+"/past/future?limit=1000")
	//fmt.Println("src: ",src)

	var m_struct intervals

	err:=json.Unmarshal([]byte(src), &m_struct)
		   if err != nil {

		   }

	 for i:=0;i<len(m_struct.Intervals);i++{

		 m_struct.Intervals[i].Begin=svc.utc_to_local(m_struct.Intervals[i].Begin)
		 m_struct.Intervals[i].End=svc.utc_to_local(m_struct.Intervals[i].End )
	 }





	//fmt.Println("m_struct: ",m_struct)

	return m_struct

}
*/
func (svc *Axxon) get_archive(camera *Camera) string {

	src, _ := svc.request_to_axxon("archive/list/" + strings.Replace(camera.VideoStreams[0].AccessPoint, "hosts/", "", 1))

	type Autogenerated struct {
		Archives []struct {
			Default bool   `json:"default"`
			Name    string `json:"name"`
		} `json:"archives"`
	}

	var m_struct Autogenerated

	err := json.Unmarshal([]byte(src), &m_struct)
	if err != nil {
		//fmt.Println(err.Error())

	}

	return m_struct.Archives[0].Name
}

func (svc *Axxon) get_intervals_from(camera *Camera) intervals {

	point := camera.VideoStreams[0].AccessPoint

	//Получить список архивов для камеры

	//fmt.Println("Получам список архивов для камеры ", camera.DisplayName, ".", camera.DisplayID)
	//   GET http://127.0.0.1:80/archive/list/SERVER1/DeviceIpint.1/SourceEndpoint.video:0:0

	type Autogenerated struct {
		Archives []struct {
			Default bool   `json:"default"`
			Name    string `json:"name"`
		} `json:"archives"`
	}

	var m_archive Autogenerated

	src, err := svc.request_to_axxon("archive/list/" + strings.Replace(camera.VideoStreams[0].AccessPoint, "hosts/", "", 1))

	if err != nil {

		fmt.Println("ERROR 1 ", err.Error)
		var empty intervals
		return empty
	}

	err = json.Unmarshal([]byte(src), &m_archive)
	if err != nil {
		fmt.Println("ERROR 2 ", err.Error)
		var empty intervals
		return empty

	}

	//	fmt.Println(m_archive)

	for i := 0; i < len(m_archive.Archives); i++ {

		archive := m_archive.Archives[i].Name

		var src string
		src, _ = svc.request_to_axxon("archive/contents/intervals/" + strings.Replace(point, "hosts/", "", 1) + "/past/future?archive=" + archive)
		//fmt.Println("src: ",src)

		var m_struct intervals

		err := json.Unmarshal([]byte(src), &m_struct)
		if err != nil {

		}

		for i := 0; i < len(m_struct.Intervals); i++ {

			m_struct.Intervals[i].Begin = svc.utc_to_local(m_struct.Intervals[i].Begin)
			m_struct.Intervals[i].End = svc.utc_to_local(m_struct.Intervals[i].End)
		}
		if len(m_struct.Intervals) > 0 {
			return m_struct
		}

	}

	//fmt.Println("m_struct: ",m_struct)
	var empty intervals
	return empty

}

func (svc *Axxon) utc_to_local(point string) string {

	year, err := strconv.Atoi(point[0:4])
	mouth, err := strconv.Atoi(point[4:6])
	day, err := strconv.Atoi(point[6:8])

	hour, err := strconv.Atoi(point[9:11])
	min, err := strconv.Atoi(point[11:13])
	sec, err := strconv.Atoi(point[13:15])

	msec := "000000"
	if len(point) == 22 {
		msec = point[15:22]
	}

	if len(point) == 15 {
		msec = ".000000"
	}

	if err != nil {

	}

	dt := time.Date(year, time.Month(mouth), day, hour, min, sec, 0, time.UTC)

	year = dt.In(time.Local).Year()
	mouth = int(dt.In(time.Local).Month())
	day = dt.In(time.Local).Day()

	hour, min, sec = dt.In(time.Local).Clock()

	var str_year, str_mouth, str_day, str_hour, str_min, str_sec string

	str_year = strconv.Itoa(year)

	if mouth < 10 {
		str_mouth = "0" + strconv.Itoa(mouth)
	} else {
		str_mouth = strconv.Itoa(mouth)
	}

	if day < 10 {
		str_day = "0" + strconv.Itoa(day)
	} else {
		str_day = strconv.Itoa(day)
	}

	if hour < 10 {
		str_hour = "0" + strconv.Itoa(hour)
	} else {
		str_hour = strconv.Itoa(hour)
	}

	if min < 10 {
		str_min = "0" + strconv.Itoa(min)
	} else {
		str_min = strconv.Itoa(min)
	}

	if sec < 10 {
		str_sec = "0" + strconv.Itoa(sec)
	} else {
		str_sec = strconv.Itoa(sec)
	}

	res := str_year + str_mouth + str_day + "T" + str_hour + str_min + str_sec + msec

	return res
}

func (svc *Axxon) get_TelemetryControlID_from(camera *Camera) string {

	var telemetryId string
	telemetryId = ""

	if len(camera.Ptzs) > 0 {

		telemetryId = camera.Ptzs[0].AccessPoint
	}
	return telemetryId
}

func (svc *Axxon) get_frash_snapshot_from(camera *Camera) string {

	src := strings.Replace(svc.get_VIDEOSOURCEID(camera), "hosts/", "", 1)
	//	fmt.Println("get snapshot  from",src)

	var snapshot string = "http://" + svc.username + ":" + svc.password + "@" + svc.ipaddr + ":" + "8000" + "/live/media/snapshot/" + src

	return snapshot
}

func (svc *Axxon) make_devList_for_client() DeviceList {
	//Блокируем запись из других потоков
	svc.RLock()
	defer svc.RUnlock()
	var cameras DeviceList

	for i := 0; i < len(svc.devList); i++ {
		cameras = append(cameras, svc.convert_for_client(&svc.devList[i]))
	}

	return cameras
}

func (svc *Axxon) get_RTSP(camera *Camera) string {

	accesspoint := strings.Replace(camera.VideoStreams[0].AccessPoint, "hosts/", "", 1)
	//	fmt.Println("get URL livestream for ", accesspoint)

	var src string
	src, _ = svc.request_to_axxon("live/media/" + accesspoint + "?format=rtsp")

	type Autogenerated struct {
		HTTP struct {
			Description string `json:"description"`
			Path        string `json:"path"`
			Port        string `json:"port"`
		} `json:"http"`
		Rtsp struct {
			Description string `json:"description"`
			Path        string `json:"path"`
			Port        string `json:"port"`
		} `json:"rtsp"`
	}

	var m_struct Autogenerated

	err := json.Unmarshal([]byte(src), &m_struct)
	if err != nil {
		//fmt.Println(err.Error())

	}

	//    //fmt.Println("Description",m_struct.Rtsp.Description)
	//   //fmt.Println("Description",m_struct.Rtsp.Path)
	//   //fmt.Println("Description",m_struct.Rtsp.Port)

	var res string

	res = "rtsp://" + svc.username + ":" + svc.password + "@" + svc.ipaddr + ":" + m_struct.Rtsp.Port + "/" + m_struct.Rtsp.Path
	//fmt.Println("URL",res)
	return res

}
func (svc *Axxon) get_state(camera *Camera) string {
	var state string

	if camera.IsActivated {
		state = "ok"
	} else {
		state = "lost"
	}

	return state
}

func (svc *Axxon) request_URL(cid int64, data []byte) (interface{}, bool) {

	/*
	       svc.Log(" ")
	   svc.Log("[Request_URL]")
	       svc.Log(" ")
	*/

	type MyJsonName struct {
		CameraId  int64  `json:"cameraId"`
		Dt        string `json:"dt"`
		Format_dt string `json:"format_dt"`
	}

	var m_struct MyJsonName

	str := string(data[:])

	//	fmt.Println("str: ",str)

	err := json.Unmarshal([]byte(str), &m_struct)
	if err != nil {

	}

	cameraId := m_struct.CameraId
	dt := m_struct.Dt
	format_dt := m_struct.Format_dt

	//	fmt.Println("cameraId : ",cameraId)
	//	fmt.Println("dt       : ",dt)
	//	fmt.Println("format_dt: ",format_dt)

	return svc.request_URL_handler(cameraId, dt, format_dt), false

}

//----------------------------------------------------------
func (svc *Axxon) request_URL_handler(cameraId int64, dt, format_dt string) interface{} {

	//fmt.Println("Запрос URL на камеру ", cameraId, " время", dt)
	/*
		//fmt.Println("----------------------------")
		//fmt.Println("")
		//fmt.Println("")

		//fmt.Println("[request_URL_handler]")
		//fmt.Println("")
		//fmt.Println("cameraId ",cameraId)
		//fmt.Println("dt ",dt)
		//fmt.Println("format_dt ",format_dt)
		//fmt.Println("")
		//fmt.Println("")
		//fmt.Println("----------------------------")
	*/
	//fmt.Println("cameraId: ", cameraId)

	//Найти у себя камеру с таким же Id
	//fmt.Println("Количество камер: ",len(svc.devices))

	var needed_camera Camera

	for i := 0; i < len(svc.devList); i++ {
		//	    fmt.Println(i)

		var m_camera = *svc.devList[i].pointer

		if svc.devList[i].id == cameraId {
			//	     fmt.Println("Найдена камера ",m_camera.DisplayName, ";ID: ",m_camera.DisplayID)
			needed_camera = m_camera

		}
	}

	var needed_point string
	needed_point = ""

	//	    fmt.Println("количество точек: ",len(needed_camera.VideoStreams))
	var width int64
	width = 0
	//	    fmt.Println(width)
	for i := 0; i < len(needed_camera.VideoStreams); i++ {

		var point string = needed_camera.VideoStreams[i].AccessPoint
		//	    fmt.Println(needed_camera.VideoStreams[i].AccessPoint)

		var this_width int64
		this_width = svc.get_width(point)
		if width < this_width {
			width = this_width
			needed_point = point
		}

	}
	if needed_point == "" {
		needed_point = needed_camera.VideoStreams[0].AccessPoint

	}
	//  //fmt.Println("Нужный стрим 2: ",needed_point)

	var liveStream string = ""
	var storageStream string = ""
	//ar snapshot string = "http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+"8000"+"/live/media/snapshot/"+strings.Replace(needed_point,"hosts/","",1)

	snapshot := ""
	liveStream = "rtsp://" + svc.username + ":" + svc.password + "@" + svc.ipaddr + ":" + "50554" + "/" + needed_point

	//  //fmt.Println("liveStream: ",liveStream)

	//   //fmt.Println("dt: ",dt)

	format_dt = "local"
	//fmt.Println("[01]")

	//fmt.Println("len(dt) ",len(dt))

	var my_intervals intervals
	my_intervals = svc.get_intervals_from(&needed_camera)
	//fmt.Println("len(dt) ",len(dt))
	if dt != "" && dt != "undefined" {

		// //fmt.Println("[02]")
		//  storageStream=svc.get_storage_rtsp_stream(needed_point,dt)

		var string_dt string

		res := svc.compare_dt_with_intervals(dt, my_intervals)

		if format_dt == "local" {
			string_dt = svc.local_to_utc(dt)
			//fmt.Println("[03]")
		}
		if format_dt == "utc" {
			string_dt = dt
			//fmt.Println("[04]")
		}

		if res == true {

			storageStream = "rtsp://" + svc.username + ":" + svc.password + "@" + svc.ipaddr + ":" + "50554" + "/archive/hosts/" + strings.Replace(needed_point, "hosts/", "", 1) + "/" + string_dt + "?speed=1"

			snapshot = "http://" + svc.username + ":" + svc.password + "@" + svc.ipaddr + ":" + "8000" + "/archive/media/" + strings.Replace(needed_point, "hosts/", "", 1) + "/" + string_dt
		}
		//else{

		//}

	}
	//fmt.Println("[05]")

	//fmt.Println("storageStream: ",storageStream)

	type MyJsonName struct {
		Id                 int64     `json:"id"`
		LiveStream         string    `json:"liveStream"`
		StorageStream      string    `json:"storageStream"`
		Snapshot           string    `json:"snapshot"`
		TelemetryControlID string    `json:"telemetryControlID"`
		Intervals          intervals `json:"intervals"`
	}

	var m_struct []MyJsonName

	var mTelemetryControlID string

	if len(needed_camera.Ptzs) > 0 {
		mTelemetryControlID = strings.Replace(needed_camera.Ptzs[0].AccessPoint, "hosts", "", 1)
	}
	//fmt.Println("TelemetryControlID: ",mTelemetryControlID)

	m_struct = append(m_struct, MyJsonName{Id: cameraId, LiveStream: liveStream,
		StorageStream:      storageStream,
		Snapshot:           snapshot,
		TelemetryControlID: mTelemetryControlID,
		Intervals:          my_intervals})

	//storageStream="http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+svc.port+"/archive/hosts/"+needed_point+
	//Добываем для этого point  ссылки на живой поток, на архивный поток и стоп кадр по данному времени

	return m_struct
}

func dt_to_float64(point string) float64 {

	//fmt.Println("point ",point)
	//fmt.Println("point ",strings.Replace(point, "T", "", 1))
	res, _ := strconv.ParseFloat(strings.Replace(point, "T", "", 1), 64)
	//fmt.Println("value ",res)
	return res
}

func (svc *Axxon) compare_dt_with_intervals(string_dt string, my_intervals intervals) bool {

	for i := 0; i < len(my_intervals.Intervals); i++ {

		val_dt := dt_to_float64(string_dt)
		val_begin := dt_to_float64(my_intervals.Intervals[i].Begin)
		val_end := dt_to_float64(my_intervals.Intervals[i].End)

		//		fmt.Println("val_begin ",val_begin)
		//		fmt.Println("val_dt    ",val_dt)
		//		fmt.Println("val_end   ",val_end)

		if (val_dt > val_begin) && (val_dt < val_end) {

			//		fmt.Println("PROFIT !!!!!")
			return true
		}
	}

	return false
}

func (svc *Axxon) get_width(point string) int64 {

	var src string

	src, _ = svc.request_to_axxon("statistics/" + strings.Replace(point, "hosts/", "", 1))

	type MyJsonName struct {
		Bitrate    int64   `json:"bitrate"`
		Fps        float64 `json:"fps"`
		Height     int64   `json:"height"`
		MediaType  int64   `json:"mediaType"`
		StreamType int64   `json:"streamType"`
		Width      int64   `json:"width"`
	}

	var m_struct MyJsonName

	err := json.Unmarshal([]byte(src), &m_struct)
	if err != nil {
		//fmt.Println(err.Error())
		return -1
	}

	//fmt.Println("Ширина кадра: ",m_struct.Width)

	return m_struct.Width
}

func (svc *Axxon) local_to_utc(point string) string {
	/*
	  //fmt.Println("")
	  //fmt.Println("")
	  //fmt.Println("")
	  //fmt.Println("[utc_to_local]")
	  //fmt.Println("")

	  //fmt.Println("point", point)
	*/
	var timestamp int = time.Now().In(time.UTC).Hour() - time.Now().In(time.Local).Hour()

	//fmt.Println("временная задержка: ",timestamp)

	year, err := strconv.Atoi(point[0:4])
	mouth, err := strconv.Atoi(point[4:6])
	day, err := strconv.Atoi(point[6:8])

	hour, err := strconv.Atoi(point[9:11])
	min, err := strconv.Atoi(point[11:13])
	sec, err := strconv.Atoi(point[13:15])

	//fmt.Println("year ", year)
	//fmt.Println("mouth ", mouth)
	//fmt.Println("day ",day)

	//fmt.Println("hour ",hour)
	//fmt.Println("min ",min)
	//fmt.Println("sec ",sec)

	if err != nil {
		//fmt.Println("err",err)
	}
	//fmt.Println("")

	//var dt string=year+"-"+mouth+"-"+day+" "+hour+":"+min+":"+sec

	// timeT, _ := time.Parse("2006-01-02 03:04:05", dt)
	dt := time.Date(year, time.Month(mouth), day, hour, min, sec, 0, time.UTC)

	//dt:=time.Date(2021,7,19,1,2,3,0,time.UTC)

	//fmt.Println(dt)

	//fmt.Println("добавляем временную задержку: ",time.Duration(timestamp)*time.Hour)
	dt = dt.Add(time.Duration(timestamp) * time.Hour)

	//fmt.Println(dt)
	/*
	  //fmt.Println("In(time,UTC) ",dt.In(time.UTC))
	  //fmt.Println("In(time,local) ",dt.In(time.Local))
	  //fmt.Println("dt.In(time.Local).Format(2999-01-02 23:59:59) ",dt.In(time.Local).Format("2006-01-02 15:04:05"))
	*/
	//timeT, _ := time.Parse("2006-01-02 03:04:05", dt.In(time.Local).Format("2006-01-02 15:04:05") )

	//fmt.Println("timeT",timeT)
	//fmt.Println("timeT",timeT)
	/*
	  year=dt.In(time.Local).Year()
	  mouth=int(dt.In(time.Local).Month())
	  day=dt.In(time.Local).Day()


	  hour,min,sec=dt.In(time.Local).Clo

	  //fmt.Println("year ", year)
	  //fmt.Println("mouth ", mouth)
	  //fmt.Println("day ",day)
	  //fmt.Println("hour ",hour)
	  //fmt.Println("min ",min)
	  //fmt.Println("sec ",sec)
	*/
	year = dt.In(time.UTC).Year()
	mouth = int(dt.In(time.UTC).Month())
	day = dt.In(time.UTC).Day()

	hour, min, sec = dt.In(time.UTC).Clock()

	/*
	  //fmt.Println("len(point) ", len(point))

	  //fmt.Println("year ", year)
	  //fmt.Println("mouth ", mouth)
	  //fmt.Println("day ",day)
	  //fmt.Println("hour ",hour)
	  //fmt.Println("min ",min)
	  //fmt.Println("sec ",sec)
	*/
	var str_year, str_mouth, str_day, str_hour, str_min, str_sec string

	str_year = strconv.Itoa(year)

	if mouth < 10 {
		str_mouth = "0" + strconv.Itoa(mouth)
	} else {
		str_mouth = strconv.Itoa(mouth)
	}

	if day < 10 {
		str_day = "0" + strconv.Itoa(day)
	} else {
		str_day = strconv.Itoa(day)
	}

	if hour < 10 {
		str_hour = "0" + strconv.Itoa(hour)
	} else {
		str_hour = strconv.Itoa(hour)
	}

	if min < 10 {
		str_min = "0" + strconv.Itoa(min)
	} else {
		str_min = strconv.Itoa(min)
	}

	if sec < 10 {
		str_sec = "0" + strconv.Itoa(sec)
	} else {
		str_sec = strconv.Itoa(sec)
	}

	str_msec := "000000"
	if len(point) > 21 {
		str_msec = point[15:22]
	}

	if len(point) < 16 {
		str_msec = ".000000"
	}
	/*
	  //fmt.Println("str_year ", str_year)
	  //fmt.Println("str_mouth ", str_mouth)
	  //fmt.Println("str_day ", str_day)
	  //fmt.Println("str_hour ", str_hour)
	  //fmt.Println("str_min ", str_min)
	  //fmt.Println("str_sec ", str_sec)
	*/
	//fmt.Println("err",err)

	//    t := time.Now()

	// For a time t, offset in seconds east of UTC (GMT)
	//    _, offset := t.Local().Zone()
	//    //fmt.Println(offset)

	// For a time t, format and display as UTC (GMT) and local times.
	//   //fmt.Println(t.In(time.UTC))
	//   //fmt.Println(t.In(time.Local))

	var res string = str_year + str_mouth + str_day + "T" + str_hour + str_min + str_sec + str_msec

	//fmt.Println("res:", res)

	return res
}

func (svc *Axxon) request_intervals(cid int64, data []byte) (interface{}, bool) {
	//  //fmt.Println("")
	var id int64
	id, _ = strconv.ParseInt(string(data[:]), 10, 64)
	var res intervals
	for i := 0; i < len(svc.devList); i++ {

		if id == svc.devList[i].id {
			res = svc.get_intervals_from(svc.devList[i].pointer)
		}

	}
	type MyJsonName struct {
		Intervals intervals `json:"intervals"`
	}
	var m_struct []MyJsonName
	m_struct = append(m_struct, MyJsonName{Intervals: res})

	return m_struct, false

}

func (svc *Axxon) execCommand(cid int64, data []byte) (interface{}, bool) {
	//   var xml string
	command := new(api.Command)
	json.Unmarshal(data, command) // TODO: handle err

	/*
	   //fmt.Println("[Axxon Exec Command]", *command )

	   //fmt.Println("command.DeviceId: ",command.DeviceId)
	   //fmt.Println("command.Command: ",command.Command)
	   //fmt.Println("command.Argument: ",command.Argument)
	 //fmt.Println("len( svc.devices): ",len( svc.devices))
	*/

	for i := 0; i < len(svc.devList); i++ {
		/*
		   //fmt.Println("---")
		   //fmt.Println("id: ",svc.devices[i].Id)
		   //fmt.Println("cameraId: ",svc.devices[i].CameraId)
		  //fmt.Println("name: ",svc.devices[i].Name)
		*/
		if command.DeviceId == svc.devList[i].id {
			d := svc.devList[i]

			switch command.Command {
			case 100:

				//fmt.Println("переход по пресету")

				//Получить телеметри Айди
				var tmtr_Id = strings.Replace(d.TelemetryControlID, "hosts/", "", 1)
				var session_id = svc.get_SessionID_value_from_Axxon(tmtr_Id)

				svc.request_to_axxon("control/telemetry/preset/go/" + tmtr_Id + "?pos=" + strconv.FormatInt(command.Argument, 10) + "&session_id=" + strconv.FormatInt(session_id, 10))

				//Остановить текущий сеанс управления телеметрией если он есть

				//Перейти по пресету с заданным айдишником

				//Остановить сеанс управления телеметрией

				return svc.request_URL_handler(d.id, "undefined", "local"), true

			

			case 101:

				return svc.request_URL_handler(d.id, "undefined", "local"), true

			default:				
			}

		}

	}

	return "", false
}
