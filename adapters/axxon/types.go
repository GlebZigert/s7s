package axxon

import (
    "sync"
    "../../api"
    "../configuration"
  //  "fmt"   
    "golang.org/x/net/websocket"
    "time"
    "fmt"
)

//type Reply func (int, string, string, interface{})

    type Camera_options struct {
    Id    string `json:"id"`
    LiveStream    string `json:"liveStream"`
    StorageStream    string `json:"storageStream"`
    Snapshot    string `json:"snapshot"`  
    Intervals intervals `json:"intervals"`  
   
}

type Stream_from_storage struct {
    Name    string    `json:"name"`
    Stream   string `json:"stream"`
    
}

type Device struct {
	Id 		int64 	  `json:"id"`
    CameraId    string    `json:"cameraId"`    
	Name	string    `json:"name"`
    Stream   stream_settings `json:"stream"`
    TelemetryControlID string `json:"telemetryControlID"`
    Cmd_presets map[int64]string  `json:"cmd_presets"`
    State string `json:"State"`
    Snapshot string `json:"Snapshot"`
    AccessMode int64 `json:"accessMode"`
}

type State struct {
	Id 			int 	`json:"id"`
	DateTime 	uint 	`json:"datetime"`
	Name 		string 	`json:"name"`
}

var m_streams []Videostream

type host_list struct {
    Hostname   string `json:"hostname"`
    Domaininfo struct {
        Domainname         string `json:"domainName"`
        Domainfriendlyname string `json:"domainFriendlyName"`
    } `json:"domainInfo"`
    Platforminfo struct {
        Machine string `json:"machine"`
        Os      string `json:"os"`
    } `json:"platformInfo"`
    Licensestatus string `json:"licenseStatus"`
    Timezone      int    `json:"timeZone"`
}

type Videostream  struct {
    Accesspoint string `json:"accessPoint"`
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
//--------------------------------------------------

type intervals struct {
    Intervals []struct {
        Begin string `json:"begin"`
        End   string `json:"end"`
    } `json:"intervals"`
    More bool `json:"more"`
}

	
type camera_list struct {
	Cameras []Camera `json:"cameras"`
}

var hosts []string



type accesspoint_settings struct{
    Accesspoint string
    Rtsp string

}

type stream_settings struct{
    Accesspoint []accesspoint_settings

}

type stream_settings_list struct{
    settings []stream_settings

}

type telemetrySession struct{
cid int64 `json:"clientId"`
point  string `json:"TelemetryPoint"`
key int64 `json:"SessionId "`
livetime int64 `json:"livetime "`
}

type DevList []Device

type alert struct{
id int64
dt time.Time
}

func (alert alert) show(){
    fmt.Println("alert.id: ",alert.id," alert.dt: ",alert.dt.String())
}

type l_alert []alert

func (svc *Axxon) append_alert() {

 svc.alerts=append(svc.alerts,alert{id:1,dt:time. Now()})
}

//если уже была тревога с камеры в эту же секунду - возыращает тру
func (svc *Axxon) find_alert(id int64,dt time.Time) (bool){
fmt.Println("l_alert.find:")
fmt.Println("len(l_alert): ",len(svc.alerts))
    for i:=0;i<len(svc.alerts);i++{

        current:=svc.alerts[i]

        fmt.Println(current.id," ",id)
        if current.id==id{
            fmt.Println("уже была тревога с этой камеры в ",current.dt.String())

                dur := dt.Sub(current.dt).Seconds()



             fmt.Println("dur ",  dur) 
            svc.alerts[i].dt=dt

             if(dur<1){
fmt.Println("!!! В ТУ ЖЕ СЕКУНДУ !!!") 

                return true
             }else{
                return false
             }


 


            return false

        }
    }



    svc.alerts=append(svc.alerts,alert{id:id,dt:dt})


    fmt.Println("len(l_alert): ",len(svc.alerts))

    return false
   
}


type Axxon struct {

    sync.RWMutex

    api.API

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

    conn *websocket.Conn

    websocket_is_connected bool

    alerts l_alert
}

func (devices DevList) Filter(list map[int64]int64) interface{} {
    var res DevList
    for i := range devices {
        // list[0] > 0 => whole service accessible
        devices[i].AccessMode = list[0]
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


/*func (devices DevList) Filter (list map[int64]int64, mask int64) (res []interface{}) {
    for i := range devices {

    //        dev := devices[i]



//            fmt.Println("Проверю, разрешена ли для пользователя камера с этим globalId: ",dev.Id)   
//            fmt.Println("list[devices[i].Id] : ",list[devices[i].Id])
     am := list[devices[i].Id] & mask // access mode

if nil == list{
//fmt.Println("[!!!] nil == list ")

}


        if nil == list || am > 0 {
            dev := devices[i]
    
            res = append(res, dev)
        }



    }
    return
}*/

	
