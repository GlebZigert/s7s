package axxon

import (
	//   "sync"
	"fmt"
	"golang.org/x/net/websocket"
	"s7server/adapters/configuration"
	"s7server/api"
	"sync"
	"time"
)

type Axxon struct {

	//Настройки соединения с сервером Axxon
	username, password, ipaddr, port string

	background_done   chan bool
	eventHandler_done chan bool

	quit              chan bool
	quit_eventHandler chan bool
	sync.RWMutex
	api.API
	cameraList             cameraList
	devList                devList
	cfg                    configuration.ConfigAPI
	telemetrySessions      map[int]telemetrySession
	current_speed_x        float64
	current_speed_y        float64
	websocket_is_connected bool
	conn                   *websocket.Conn
	alerts                 l_alert
	/*
	    //   sync.RWMutex

	    cfg     configuration.ConfigAPI

	    telemetrySessions map[int]telemetrySession

		devices DevList
		m_camera_list camera_list

	    m_settings_list  stream_settings_list

	    username,password,ipaddr,port string

	    stream_from_storage map[int] Stream_from_storage

	    current_telemetry_point string

	    current_telemetry_id int64

	    current_pun float64

	    current_titl float64

	    current_speed_x float64

	    current_speed_y float64

	    signal chan string
	*/
}

//Структура которую возвращает Ахон по запросу /camera/list
type cameraList struct {
	List []Camera `json:"cameras"`
}

type Camera struct {
	Archives []struct {
		AccessPoint        string `json:"accessPoint"`
		Default            bool   `json:"default"`
		IsEmbedded         bool   `json:"isEmbedded"`
		Storage            string `json:"storage"`
		StorageDisplayName string `json:"storageDisplayName"`
	} `json:"archives"`
	AudioStreams []struct {
		AccessPoint string `json:"accessPoint"`
		IsActivated bool   `json:"isActivated"`
	} `json:"audioStreams"`
	Comment          string        `json:"comment"`
	Detectors        []interface{} `json:"detectors"`
	DisplayID        string        `json:"displayId"`
	DisplayName      string        `json:"displayName"`
	Groups           []string      `json:"groups"`
	IPAddress        string        `json:"ipAddress"`
	IsActivated      bool          `json:"isActivated"`
	Latitude         string        `json:"latitude"`
	Longitude        string        `json:"longitude"`
	Model            string        `json:"model"`
	OfflineDetectors []interface{} `json:"offlineDetectors"`
	Ptzs             []struct {
		AccessPoint string `json:"accessPoint"`
		AreaZoom    bool   `json:"areaZoom"`
		Focus       struct {
			IsAbsolute  bool `json:"isAbsolute"`
			IsAuto      bool `json:"isAuto"`
			IsContinous bool `json:"isContinous"`
			IsRelative  bool `json:"isRelative"`
		} `json:"focus"`
		Iris struct {
			IsAbsolute  bool `json:"isAbsolute"`
			IsAuto      bool `json:"isAuto"`
			IsContinous bool `json:"isContinous"`
			IsRelative  bool `json:"isRelative"`
		} `json:"iris"`
		IsActive bool `json:"is_active"`
		Move     struct {
			IsAbsolute  bool `json:"isAbsolute"`
			IsAuto      bool `json:"isAuto"`
			IsContinous bool `json:"isContinous"`
			IsRelative  bool `json:"isRelative"`
		} `json:"move"`
		PointMove bool `json:"pointMove"`
		Zoom      struct {
			IsAbsolute  bool `json:"isAbsolute"`
			IsAuto      bool `json:"isAuto"`
			IsContinous bool `json:"isContinous"`
			IsRelative  bool `json:"isRelative"`
		} `json:"zoom"`
	} `json:"ptzs"`
	TextSources  []interface{} `json:"textSources"`
	Vendor       string        `json:"vendor"`
	VideoStreams []struct {
		AccessPoint string `json:"accessPoint"`
	} `json:"videoStreams"`
}

type devList []dev

type DeviceList []Device

func (devices DeviceList) GetList() []int64 {
	/*
	   fmt.Println(" ")
	   fmt.Println(" ")
	   fmt.Println(" ")
	   fmt.Println("axxon (devices DeviceList) GetList() ")
	   fmt.Println("len ",len(devices))
	   fmt.Println(" ")
	   fmt.Println(" ")
	   fmt.Println(" ")
	*/

	list := make([]int64, 0, len(devices))

	for _, dev := range devices {
		list = append(list, dev.Id)
	}

	return list
}

func (devices DeviceList) Filter(list map[int64]int64) interface{} {
	/*
	   fmt.Println(" ")
	   fmt.Println(" ")
	   fmt.Println(" ")
	   fmt.Println("axxon (devices DeviceList) Filter")
	   fmt.Println("len ",len(devices))
	   fmt.Println(" ")
	   fmt.Println(" ")
	   fmt.Println(" ")
	*/
	var res DeviceList

	//fmt.Println("devices: ",devices)
	//fmt.Println("list: ",list)

	for i := range devices {

		// list[0] > 0 => whole service accessible
		devices[i].AccessMode = list[0]
		if 0 == devices[i].AccessMode {
			//      fmt.Println("[1] ")
			//      fmt.Println("devices[i].Id: ",devices[i].Id)
			devices[i].AccessMode = list[devices[i].Id]
		}
		if devices[i].AccessMode > 0 {
			//     fmt.Println("[2] ")
			res = append(res, devices[i])

		}

		//fmt.Println(devices[i].Name, "; access: ", devices[i].AccessMode)

	}
	if len(res) > 0 {
		return res
	} else {
		return nil
	}
}

//
type dev struct {
	id                 int64
	VIDEOSOURSEID      string
	TelemetryControlID string
	AccessMode         int64
	pointer            *Camera
	actual             bool
	state              string
}

//struct to send camera options to client
type Device struct {
	Sid		   int64  `json:"sid"`
	Id                 int64  `json:"id"`
	Name               string `json:"name"`
	Stream             string `json:"stream"`
	State              string `json:"state"`
	TelemetryControlID string `json:"telemetryControlID"`

	//   Cmd_presets map[int64]string  `json:"cmd_presets"`
	//   State string `json:"State"`
	Intervals      intervals
	Frash_Snapshot string `json:"frash_snapshot"`
	AccessMode     int64  `json:"accessMode"`
}

type intervals struct {
	Intervals []struct {
		Begin string `json:"begin"`
		End   string `json:"end"`
	} `json:"intervals"`
	More bool `json:"more"`
}

type telemetrySession struct {
	cid      int64  `json:"clientId"`
	point    string `json:"TelemetryPoint"`
	key      int64  `json:"SessionId "`
	livetime int64  `json:"livetime "`
}

type alert struct {
	id int64
	dt time.Time
}

func (alert alert) show() {
	fmt.Println("alert.id: ", alert.id, " alert.dt: ", alert.dt.String())
}

type l_alert []alert

func (svc *Axxon) append_alert() {

	svc.alerts = append(svc.alerts, alert{id: 1, dt: time.Now()})
}

//если уже была тревога с камеры в эту же секунду - возыращает тру
func (svc *Axxon) find_alert(id int64, dt time.Time) bool {
	fmt.Println("l_alert.find:")
	fmt.Println("len(l_alert): ", len(svc.alerts))
	for i := 0; i < len(svc.alerts); i++ {

		current := svc.alerts[i]

		fmt.Println(current.id, " ", id)
		if current.id == id {
			fmt.Println("уже была тревога с этой камеры в ", current.dt.String())

			dur := dt.Sub(current.dt).Seconds()

			fmt.Println("dur ", dur)
			svc.alerts[i].dt = dt

			if dur < 1 {
				fmt.Println("!!! В ТУ ЖЕ СЕКУНДУ !!!")

				return true
			} else {
				return false
			}

			return false

		}
	}

	svc.alerts = append(svc.alerts, alert{id: id, dt: dt})

	fmt.Println("len(l_alert): ", len(svc.alerts))

	return false

}
