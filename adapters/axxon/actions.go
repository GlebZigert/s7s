package axxon

import (
    "encoding/json"
    //    "log"
    //"encoding/json"
    "fmt"
    "time"    
    "io/ioutil"
    "net/http"
    "strings"
     "../../api"  
    "strconv" 
    "sync"

    //   "reflect"
)




/****************** RIF+  Actions ******************/
func (svc *Axxon)  DateTime(cid int64, data []byte) (interface{}, bool){

/*
var dt string
dt=string(data[:])
//fmt.Println("[DateTime]--------------------------------------------------------")  
//fmt.Println(dt)
dt=strings.Replace(dt,"\"","",2)

//fmt.Println(len(svc.devices))

for i:=0;i<len(svc.devices);i++{
var dev=svc.devices[i] 
//fmt.Println("Камера ",dev.Id,":",dev.Name)

for j:=0;j<len(dev.Stream.Accesspoint);j++{

var point=dev.Stream.Accesspoint[j]
//fmt.Println("Камера ",dev.Id," Стрим ",j,": ",point.Accesspoint)
var vid=point.Accesspoint

//И для каждого стрима получаем: список архивов

//список времен записи в каждом архиве

//выбираем архив где стрим с бОльшим качеством

//возвращаем пару - название камеры и ссылку на этот стрим

var stream string
stream = svc.get_storage_stream(vid, dt)



if stream !="no_stream" {
    //fmt.Println("[URL] ",stream)    
    svc.stream_from_storage[i]=Stream_from_storage{Name: dev.Name, Stream: stream}
}
//svc.devices[i] = Device{Id: i,Name: m_camera.Displayname,Stream: svc.m_settings_list.settings[i]}    




}    

}
*/
return svc.stream_from_storage, false
}

func (svc *Axxon) test_http_connection() bool{

//fmt.Println("[test_http_connection]")  
//var res bool


var request string
    request="http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+svc.port+"/"+"uuid"
//request="http://"+"wrong_user"+":"+svc.password+"@"+svc.ipaddr+":"+svc.port+"/"+"uuid"
    //fmt.Println("req: ",request)


resp, err := http.Get(request)  
    if err != nil { 
        //fmt.Println("err")    
        //fmt.Println(err) 
        return false
    }


    defer resp.Body.Close()


  bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        //fmt.Println(err)
        return false
    }

    bodyString := string(bodyBytes)


    if bodyString==""{
      return false
    }

    //fmt.Println("bodyString: ",bodyString)    

return true

}

func (svc *Axxon)  getDateTime(dt string){

//fmt.Println("[DateTime]")  
//fmt.Println(string(dt))
//fmt.Println(len(svc.devices))
/*
for i:=0;i<len(svc.devices);i++{
var dev=svc.devices[i] 
//fmt.Println(dev.Id," ",dev.Name)

}*/
}
var lock sync.Mutex
func (svc *Axxon) Get_list_devices(){
 lock.Lock()
 defer lock.Unlock()
fmt.Println("[Get_list_devices]") 
svc.m_settings_list.settings=nil
    svc.get_camera_list()

//    var streams [][]string
//fmt.Println("len(svc.m_camera_list.Cameras)  ",len(svc.m_camera_list.Cameras)) 
    for i:=0;i<len(svc.m_camera_list.Cameras);i++{
        var m_camera *Camera
        m_camera=&svc.m_camera_list.Cameras[i]
        //fmt.Println(m_camera.DisplayName)   
//        //fmt.Println("[1]")   
        //fmt.Println("Ptzs AccessPoint:",m_camera.Ptzs[0].AccessPoint )
        var sss stream_settings
        var settings *stream_settings


        svc.m_settings_list.settings=append(svc.m_settings_list.settings,sss)
        settings=&svc.m_settings_list.settings[i]
              

        for j:=0;j<len(m_camera.VideoStreams);j++{

//         //fmt.Println("[2]")   
            var stream string
            stream=m_camera.VideoStreams[j].AccessPoint 
            //fmt.Println(stream)             

//            //fmt.Println("[3]")  
            var fff accesspoint_settings
            settings.Accesspoint=append(settings.Accesspoint,fff)
            var accesspoint *accesspoint_settings

            accesspoint=&settings.Accesspoint[j]
//            //fmt.Println("[4]")  
            var str_access string

            str_access=strings.Replace(stream,"hosts/","",1)
            
            //fmt.Println(str_access)   

            //fmt.Println(svc.get_RTSP(str_access) )
//            //fmt.Println("[5]")  
            accesspoint.Accesspoint=str_access
            accesspoint.Rtsp=svc.get_RTSP(str_access)
//        //fmt.Println("[6]")  


//Получаем TelemetryControlID


        //    svc.camera_RTSP_streams[i]=svc.get_RTSP(stream.Accesspoint) 
        //    svc.camera_RTSP_streams[i].stream[j]=svc.get_RTSP(stream.Accesspoint) 

        }  
//        //fmt.Println(streams[i])  
    }

    //fmt.Println(svc.camera_RTSP_streams[0].stream[0])


//svc.devices=nil

//fmt.Println("!!! len(svc.m_camera_list.Cameras)  ",len(svc.m_camera_list.Cameras)) 
//fmt.Println("!!! svc.devices  ",len(svc.devices)) 


for i:=0;i<len( svc.m_camera_list.Cameras);i++{
  //fmt.Println("..")
    var m_camera Camera
    m_camera=svc.m_camera_list.Cameras[i]

    //fmt.Println("m_camera.DisplayName ",m_camera.DisplayName)  

    var globalDeviceId int64


    globalDeviceId=svc.cfg.GlobalDeviceId(svc.Settings.Id,m_camera.DisplayID,m_camera.DisplayName)


//Проверяем есть ли в системе устройство с этим глобал айди
res:=false
for j:=0;j<len(svc.devices);j++{

if globalDeviceId==svc.devices[j].Id{
res=true

}

}

if !res{
 fmt.Println("[Добавляем] ",m_camera.DisplayName)  
dt:=svc.local_to_utc(svc.current_dt())
snapshot:=svc.request_Snapshot_handler(m_camera.DisplayID,dt,"local")
state:="ok"
if !m_camera.IsActivated{
state="lost"
}

var telemetryId string
telemetryId=""
if len(m_camera.Ptzs)>0{
telemetryId=strings.Replace(m_camera.Ptzs[0].AccessPoint,"hosts","",1)
}

svc.devices = append (svc.devices,Device{Id: globalDeviceId ,
                            CameraId: m_camera.DisplayID ,
                            Name: m_camera.DisplayName,
                            Stream: svc.m_settings_list.settings[i],
                            TelemetryControlID :telemetryId,
                            State:state, 
                            Snapshot:snapshot,
                        //    AccessMode: accessMode 
                          }) 

}else{
   fmt.Println("[Не добавляем ничего]")  
}


}
}



func (svc *Axxon) listDevices(cid int64, data []byte) (interface{}, bool) {


  //fmt.Println("[listDevices !!!]")  
    svc.Get_list_devices()
// svc.Send_to_events_websocket()

/* 
 //fmt.Println("======================")
//fmt.Println("=")
//fmt.Println("=")
//fmt.Println("[HERE]]")
//fmt.Println(svc.devices)
//fmt.Println("=")
//fmt.Println("=")
//fmt.Println("======================")  
*/
    return svc.devices, false
    
}



func (svc *Axxon) msg_to_axxon(request string) string{


    var result string
    result="http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+svc.port+"/"+request
    //fmt.Println("req: ",result)
	return result
}

//rtsp://root:root@192.168.0.187:50554/hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1
//rtsp://root:root@192.168.0.187:50554/archive/hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1/20210427T143416.870000?speed=1

//rtsp://root:root@192.168.0.187:50554/archive/hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0/20210429T183819.650000?archive=hosts/ASTRAAXXON/MultimediaStorage.AntiqueWhite/MultimediaStorage?speed=1
//http://root:root@192.168.0.187:8000/archive/contents/intervals/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0/future/past/



//http://192.168.0.187:8000/archive/media/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1/20210427T143416.870000?speed=1

//http://192.168.0.187:8000/archive/media/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1/past?speed=1


//http://root:root@192.168.0.187:8000/archive/media/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1/20210426T195046.831000?speed=1


/*
gleb@astra:~$ curl http://root:root@192.168.0.187:8000/archive/list/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0/
{
   "archives" : [
      {
         "default" : false,
         "name" : "hosts/ASTRAAXXON/DeviceIpint.2/MultimediaStorage.0"
      },
      {
         "default" : false,
         "name" : "hosts/ASTRAAXXON/MultimediaStorage.AntiqueWhite/MultimediaStorage"
      }
   ]
}
gleb@astra:~$ curl http://root:root@192.168.0.187:8000/archive/contents/intervals/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0/future/past/
{
   "intervals" : [
      {
         "begin" : "20210429T181212.407000",
         "end" : "20210429T181228.407000"
      },
      {
         "begin" : "20210429T185927.685000",
         "end" : "20210430T073758.281000"
      }
   ],
   "more" : false
}

*/



//GET http://root:root@192.168.0.187:8000/archive/media/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1/20210426T195046.831000?format=rtsp&speed=1

//GET http://root:root@192.168.0.187:8000/archive/media/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0/20210426T195046.831000?format=rtsp&speed=1&archive=hosts/ASTRAAXXON/MultimediaStorage.AntiqueWhite/MultimediaStorage

//curl http://IP-адрес:порт/префикс/archive/media/HOSTNAME/DeviceIpint.23/SourceEndpoint.video:0:0/20110608T060141.375?format=rtsp&speed=1&w=640&h=480

//"rtsp://root:root@192.168.0.187:50554/hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1",
//"rtsp://root:root@192.168.0.187:50554/hosts/ASTRAAXXON/DeviceIpint.1/SourceEndpoint.video:0:1",

//http://root:root@192.168.0.187:8000/live/media/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0

//http://root:root@192.168.0.187:8054/hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0

//rtsp://root:root@192.168.0.187:8054/hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0

//src:  {"http":{"description":"RTP/RTSP/HTTP/TCP","path":"hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0","port":"8554"},
//"rtsp":{"description":"RTP/UDP or RTP/RTSP/TCP", "path":"hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0","port":"50554"}}

/*
gleb@astra:~/QML_player$ curl http://root:root@192.168.0.187:8000/archive/list/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:0/
{
   "archives" : [
      {
         "default" : false,
         "name" : "hosts/ASTRAAXXON/DeviceIpint.2/MultimediaStorage.0"
      },
      {
         "default" : false,
         "name" : "hosts/ASTRAAXXON/MultimediaStorage.AntiqueWhite/MultimediaStorage"
      }
   ]
}
*/

//Получение живого потока от видеокамеры
//GET http://IP-адрес:порт/префикс/live/media/{VIDEOSOURCEID}
func (svc *Axxon) stream(VIDEOSOURCEID string){
    svc.request_to_axxon("live/media/"+VIDEOSOURCEID)     


}


//get list videostreams
func (svc *Axxon) get_videostreams() []string{
    var res []string
    svc.m_camera_list=svc.get_camera_list()
//    //fmt.Println(len(svc.m_camera_list.Cameras))
   
   //Для каждой камеры
        for i:=0;i<len(svc.m_camera_list.Cameras);i++{
        var m_camera Camera
        m_camera=svc.m_camera_list.Cameras[i]
     //   //fmt.Println(m_camera.DisplayName)   
        //Для каждого потока этой видеокамеры
        for i:=0;i<len(m_camera.VideoStreams);i++{
            var stream string
            stream=m_camera.VideoStreams[i].AccessPoint
//            //fmt.Println(stream.Accesspoint) 
            res=append(res,stream)
            
        }    
    }
    return res
}

//GET http://IP-адрес:порт/префикс/archive/media/VIDEOSOURCEID/STARTTIME?параметры,

//rtsp://root:root@192.168.0.187:50554/archive/hosts/ASTRAAXXON/DeviceIpint.2/SourceEndpoint.video:0:1/20210427T143416.870000?speed=1
func (svc *Axxon) get_storage_stream(Accesspoint, data string) string{

    var src string  
    
    src = svc.request_to_axxon("archive/media/"+Accesspoint+"/"+data+"?format=rtsp&speed=1") 

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

    err:=json.Unmarshal([]byte(src), &m_struct)
    if err != nil {
        //fmt.Println(err.Error()) 
        return "no_stream"          
        
    }

 //   //fmt.Println("Description",m_struct.Rtsp.Description)
 //   //fmt.Println("Path",m_struct.Rtsp.Path)
 //   //fmt.Println("Port",m_struct.Rtsp.Port)

//     //fmt.Println(strings.Split(m_struct.Rtsp.Path,"&id")[0])

    var res string
    
    res="rtsp://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+m_struct.Rtsp.Port+"/"+strings.Split(m_struct.Rtsp.Path,"&id")[0]
    //fmt.Println("URL",res)
    return res
}

func (svc *Axxon) get_RTSP(Accesspoint string) string{
    var src string  
    src=svc.request_to_axxon("live/media/"+Accesspoint+"?format=rtsp") 

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

    err:=json.Unmarshal([]byte(src), &m_struct)
    if err != nil {
        //fmt.Println(err.Error())            
        
    }

 //    //fmt.Println("Description",m_struct.Rtsp.Description)
 //   //fmt.Println("Description",m_struct.Rtsp.Path)
 //   //fmt.Println("Description",m_struct.Rtsp.Port)



    var res string
    
    res="rtsp://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+m_struct.Rtsp.Port+"/"+m_struct.Rtsp.Path
    //fmt.Println("URL",res)
    return res
}

//Получение списка источников видео (камер)
//GET http://IP-адрес:порт/префикс/camera/list
func (svc *Axxon) get_camera_list() camera_list{
    var src string  
    src=svc.request_to_axxon("camera/list/")  
    
    
    err:=json.Unmarshal([]byte(src), &svc.m_camera_list)
    if err != nil {
        //fmt.Println(err.Error())            
    //    err=json.Unmarshal([]byte(src), &svc.m_camera_list)
    }



    return svc.m_camera_list
}    

//Информация о конкретном сервере
//GET http://IP-адрес:порт/префикс/hosts/
func (svc *Axxon) get_host_settings(host string) host_list{

//    //fmt.Println(host) 
    var src string  
    src=svc.request_to_axxon("hosts/"+host) 



    var m_struct host_list

    err:=json.Unmarshal([]byte(src), &m_struct)
    if err != nil {
        //fmt.Println(err.Error())            
        
    }

    return m_struct
               
}



//Получение списка серверов
//GET http://IP-адрес:порт/префикс/uuid
func (svc *Axxon) get_hosts() []string{

    var src string
    src=svc.request_to_axxon("hosts")    

       var m_struct []string
     
       err:=json.Unmarshal([]byte(src), m_struct)
       if err != nil {
           //fmt.Println(err.Error())            
           
       }

       return m_struct
}


// Получение уникального идентификатора
//GET http://IP-адрес:порт/префикс/uuid
func (svc *Axxon) get_uui() string{
    var src string
    src=svc.request_to_axxon("uuid")
    type uuid_struct struct {
        Uuid string `json:"uuid"`
        }
       
       var m_uuid uuid_struct
        
       
        err:=json.Unmarshal([]byte(src), &m_uuid)
       if err != nil {
           //fmt.Println(err.Error())            
           return ""
       }
       return m_uuid.Uuid
    }








/**/


func (svc *Axxon) request_to_axxon(request string) string{

/*
    resp, err := http.Get(svc.msg_to_axxon(request))  
    if err != nil {
        //fmt.Println("Error: " + err.Error())
    }   
    defer resp.Body.Close()

var result map[string]interface{}

    json.NewDecoder(resp.Body).Decode(&result)

    //fmt.Println(result)
    //fmt.Println("----------------------------------------------------------------------------")   
    var i=0
    for i<result.size{
     //fmt.Println(result[i])
     i++       

    }
    



    return ""
*/


resp, err := http.Get(svc.msg_to_axxon(request))  
    if err != nil { 
        //fmt.Println("err") 		
        //fmt.Println(err) 
        return ""
    }


    defer resp.Body.Close()


  bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        //fmt.Println(err)
    }
    bodyString := string(bodyBytes)
 //   //fmt.Println(bodyString)

    return bodyString
/*
    for true {
        //fmt.Println("size ",resp.ContentLength)
        bs := make([]byte, 10000)
     
        n, err := resp.Body.Read(bs)
//        //fmt.Println(bs[:n])
        str:=string(bs[:n])

        if n == 0 || err != nil{
            //fmt.Println(err.Error())  
           
        }
        //fmt.Println("src: ",string(str))  
  
        return str


    }
    return ""
    */
   
}


func (svc *Axxon) request_to(username,password,ipaddr,port,request string){
    request="http://"+username+":"+password+"@"+ipaddr+":"+port+"/"+request
    resp, err := http.Get(request)  

    if err != nil { 
        //fmt.Println("err") 		
        //fmt.Println(err) 
        return
    } 
    defer resp.Body.Close()
    for true {
        bs := make([]byte, 1014)
        n, err := resp.Body.Read(bs)
//        //fmt.Println(string(bs[:n]))
         
        if n == 0 || err != nil{
            break
        }


        


    }
}
//----------------------------------------------------------
func (svc *Axxon) current_dt() string{ 
    dt := time. Now().String()
    fmt. Println("Current date and time is: ", dt)


year, _ := strconv.Atoi(dt[0:4])
mouth, _ := strconv.Atoi(dt[5:7])
day, _ := strconv.Atoi(dt[8:10])

hour, _ := strconv.Atoi(dt[11:13])
min, _ := strconv.Atoi(dt[14:16])
sec, _ := strconv.Atoi(dt[17:19])

msec, _ := strconv.Atoi(dt[20:len(dt)])




//hour,min,sec=dt.In(time.Local).Clock()


var str_year,str_mouth,str_day,str_hour,str_min,str_sec string

str_year=strconv.Itoa(year)

if(mouth<10){
str_mouth="0"+strconv.Itoa(mouth)
}else{
str_mouth=strconv.Itoa(mouth)
}


if(day<10){
str_day="0"+strconv.Itoa(day)
}else{
str_day=strconv.Itoa(day)
}


if(hour<10){
str_hour="0"+strconv.Itoa(hour)
}else{
str_hour=strconv.Itoa(hour)
}


if(min<10){
str_min="0"+strconv.Itoa(min)
}else{
str_min=strconv.Itoa(min)
}


if(sec<10){
str_sec="0"+strconv.Itoa(sec)
}else{
str_sec=strconv.Itoa(sec)
}
str_msec:=strconv.Itoa(msec)

//fmt.Println("[6]")
/*
//fmt.Println("str_year ", str_year)
//fmt.Println("str_mouth ", str_mouth)
//fmt.Println("str_day ", str_day)
//fmt.Println("str_hour ", str_hour)
//fmt.Println("str_min ", str_min)
//fmt.Println("str_sec ", str_sec)
//fmt.Println("str_msec ", str_msec)
*/
//    t := time.Now()

    // For a time t, offset in seconds east of UTC (GMT)
//    _, offset := t.Local().Zone()
//    //fmt.Println(offset)

    // For a time t, format and display as UTC (GMT) and local times.
//   //fmt.Println(t.In(time.UTC))
//   //fmt.Println(t.In(time.Local))
//fmt.Println("point[13:22]: ",point[15:22])



res:=str_year+str_mouth+str_day+"T"+str_hour+str_min+str_sec+"."+str_msec   

    return res
}
//----------------------------------------------------------
func (svc *Axxon) request_Snapshot_handler(cameraId string, dt, format_dt string) string{ 
fmt.Println("----------------------------")
fmt.Println("")
fmt.Println("")
fmt.Println("[request_Snapshot_handler]")
fmt.Println("")
fmt.Println("cameraId ",cameraId)
fmt.Println("dt ",dt)
fmt.Println("format_dt ",format_dt)
fmt.Println("")
fmt.Println("")
fmt.Println("----------------------------")
fmt.Println("cameraId: ", cameraId)   

    //Найти у себя камеру с таким же Id
    //fmt.Println("Количество камер: ",len(svc.devices))   

    var needed_camera Camera



    for i:=0;i<len(svc.m_camera_list.Cameras);i++{
//    //fmt.Println(i) 

    var m_camera=svc.m_camera_list.Cameras[i]

    if  m_camera.DisplayID==cameraId{
     //fmt.Println("Найдена камера ",m_camera.DisplayName, ";ID: ",m_camera.DisplayID)
     needed_camera=m_camera                

    }    
    }

    var needed_point string
    needed_point=""

    //fmt.Println("количество точек: ",len(needed_camera.VideoStreams))
    var width int64
    width=0
    //fmt.Println(width)
    for i:=0;i<len(needed_camera.VideoStreams);i++{

    var point string=needed_camera.VideoStreams[i].AccessPoint
    //fmt.Println(needed_camera.VideoStreams[i].AccessPoint)


    var this_width int64
    this_width=svc.get_width(point)
    if width<this_width{
    width=this_width  
    needed_point=point      
    }

    }

     if needed_point==""{
      needed_point=needed_camera.VideoStreams[0].AccessPoint

    }

    //fmt.Println("Нужный стрим 1: ",needed_point)

   var snapshot string = "http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+"8000"+"/live/media/snapshot/"+strings.Replace(needed_point,"hosts/","",1)
  

    //проверку на наличие эиого времени в архиве
/*
    var my_intervals intervals
    my_intervals=svc.get_storage_intervals(needed_point)
    //fmt.Println("len(dt) ",len(dt))
    if dt!=""&&dt!="undefined"{


// //fmt.Println("[02]")
  //  storageStream=svc.get_storage_rtsp_stream(needed_point,dt)

    //var string_dt string

    res:=svc.compare_dt_with_intervals(dt,my_intervals)


    if res==true{
    */
    snapshot="http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+"8000"+"/archive/media/"+strings.Replace(needed_point,"hosts/","",1)+"/"+dt
/*
  }else{
    snapshot=""
  }
  }
  */





    
     //fmt.Println("[05]")




    return snapshot
}

//----------------------------------------------------------
func (svc *Axxon) request_URL_handler(cameraId string, dt, format_dt string) interface{}{ 
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



    for i:=0;i<len(svc.m_camera_list.Cameras);i++{
   // //fmt.Println(i) 

    var m_camera=svc.m_camera_list.Cameras[i]

    if  m_camera.DisplayID==cameraId{
  //   //fmt.Println("Найдена камера ",m_camera.DisplayName, ";ID: ",m_camera.DisplayID)
     needed_camera=m_camera                

    }    
    }

    var needed_point string
    needed_point=""

  //  //fmt.Println("количество точек: ",len(needed_camera.VideoStreams))
    var width int64
    width=0
  //  //fmt.Println(width)
    for i:=0;i<len(needed_camera.VideoStreams);i++{

    var point string=needed_camera.VideoStreams[i].AccessPoint
  //  //fmt.Println(needed_camera.VideoStreams[i].AccessPoint)


    var this_width int64
    this_width=svc.get_width(point)
    if width<this_width{
    width=this_width  
    needed_point=point      
    }


    }
 if needed_point==""{
      needed_point=needed_camera.VideoStreams[0].AccessPoint

    }
  //  //fmt.Println("Нужный стрим 2: ",needed_point)

    var liveStream string = ""
    var storageStream string = ""
   //ar snapshot string = "http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+"8000"+"/live/media/snapshot/"+strings.Replace(needed_point,"hosts/","",1)

   snapshot:=""
    liveStream="rtsp://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+"50554"+"/"+needed_point

  //  //fmt.Println("liveStream: ",liveStream)

 //   //fmt.Println("dt: ",dt)


    
    format_dt="local"
 //fmt.Println("[01]")

 //fmt.Println("len(dt) ",len(dt))

 var my_intervals intervals
    my_intervals=svc.get_storage_intervals(needed_point)
    //fmt.Println("len(dt) ",len(dt))
    if dt!=""&&dt!="undefined"{


// //fmt.Println("[02]")
  //  storageStream=svc.get_storage_rtsp_stream(needed_point,dt)

    var string_dt string

    res:=svc.compare_dt_with_intervals(dt,my_intervals)

    if format_dt=="local"{
        string_dt=svc.local_to_utc(dt)
 //fmt.Println("[03]")        
    }
    if format_dt=="utc"{
        string_dt=dt
 //fmt.Println("[04]")        
    }

    

    

    if res==true{
    

    storageStream="rtsp://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+"50554"+"/archive/hosts/"+strings.Replace(needed_point,"hosts/","",1)+"/"+string_dt+"?speed=1"

    
    
    snapshot="http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+"8000"+"/archive/media/"+strings.Replace(needed_point,"hosts/","",1)+"/"+string_dt
  }else{
    

  }



    }
     //fmt.Println("[05]")
    

    //fmt.Println("storageStream: ",storageStream)  
    
    type MyJsonName struct {
    Id    string `json:"id"`
    LiveStream    string `json:"liveStream"`
    StorageStream    string `json:"storageStream"`
    Snapshot    string `json:"snapshot"`  
    TelemetryControlID string `json:"telemetryControlID"`   
    Intervals intervals `json:"intervals"`  
}

var m_struct []MyJsonName 

var mTelemetryControlID string

if len(needed_camera.Ptzs)>0{
mTelemetryControlID=strings.Replace(needed_camera.Ptzs[0].AccessPoint,"hosts","",1)
}
//fmt.Println("TelemetryControlID: ",mTelemetryControlID)


m_struct=append(m_struct,MyJsonName{Id:cameraId,LiveStream:liveStream,StorageStream:storageStream,Snapshot:snapshot,TelemetryControlID:mTelemetryControlID,Intervals:my_intervals})



   //storageStream="http://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+svc.port+"/archive/hosts/"+needed_point+
    //Добываем для этого point  ссылки на живой поток, на архивный поток и стоп кадр по данному времени




    return m_struct
}

func check_if_first_more_then_second(first,second string)(bool){
  return false
}

func dt_to_float64(point string)(float64){

//fmt.Println("point ",point)
//fmt.Println("point ",strings.Replace(point, "T", "", 1))
res,_:=strconv.ParseFloat(strings.Replace(point, "T", "", 1),64)
//fmt.Println("value ",res)
return res
}



func (svc *Axxon) compare_dt_with_intervals(string_dt string, my_intervals intervals)(bool){
//fmt.Println("[compare_dt_with_intervals] ")


for i:=0;i<len(my_intervals.Intervals);i++{

 fmt.Println("[----] ") 
 //fmt.Println("dt ",string_dt)  
 
 //fmt.Println("dt_to_float ",dt_to_float64(string_dt))  


 //fmt.Println("Begin ",my_intervals.Intervals[i].Begin)  

//fmt.Println("End ",my_intervals.Intervals[i].End) 

val_dt:=dt_to_float64(string_dt)
val_begin:=dt_to_float64(my_intervals.Intervals[i].Begin)
val_end:=dt_to_float64(my_intervals.Intervals[i].End)


fmt.Println("val_begin ",val_begin) 
fmt.Println("val_dt    ",val_dt) 
fmt.Println("val_end   ",val_end) 


   if (val_dt>val_begin)&&(val_dt<val_end){

    fmt.Println("PROFIT !!!!!")
    return true
   }
}


  return false
}




//----------------------------------------------------------
func (svc *Axxon) request_URL(cid int64, data []byte) (interface{}, bool){ 



        svc.Log("")
        svc.Log("")
        svc.Log("")
    svc.Log("[Request_URL]")
        svc.Log("")
        svc.Log("")
        svc.Log("")



type MyJsonName struct {
    CameraId    string `json:"cameraId"`
    Dt        string `json:"dt"`
    Format_dt     string `json:"format_dt"`
   
}

var m_struct MyJsonName


str:=string(data[:])

fmt.Println("str: ",str)


err:=json.Unmarshal([]byte(str), &m_struct)
       if err != nil {
           //fmt.Println(err.Error())            
           
       }





     /*   
 str:=string(data[:])

    str=strings.Replace(str,"\"","",2)
    //fmt.Println("str: ",str)
    */

 /*
    var str string


    str=string(data[:])

    str=strings.Replace(str,"\"","",2)
    //fmt.Println("str: ",str)

     words := strings.Fields(str)
    for idx, word := range words {
        fmt.Printf("Word %d is: %s\n", idx, word)
    }

    var dt string
    dt=""
    if len(words)>1{
    dt=words[1]
    }

    var format_dt string
    format_dt="local"

    if len(words)>2{
    format_dt=words[2]
    }

    var cameraId string
    cameraId=words[0]
*/


cameraId:=m_struct.CameraId
dt:=m_struct.Dt
format_dt:=m_struct.Format_dt

fmt.Println("cameraId : ",cameraId)
fmt.Println("dt       : ",dt)
fmt.Println("format_dt: ",format_dt)




    return svc.request_URL_handler(cameraId,dt,format_dt),false
   
}


func (svc *Axxon) request_URL_for_globalDeviceId(cid int64, data []byte) (interface{}, bool){ 
    svc.Log("[request_URL_for_globalDeviceId]")
    var str string
    str=string(data[:])
    str=strings.Replace(str,"\"","",2)
    //fmt.Println("str: ",str)

     words := strings.Fields(str)
   // for idx, word := range words {
  //      fmt.Printf("Word %d is: %s\n", idx, word)
  //  }

 //   var dt string
 //   dt=""
 //   if len(words)>1{
//    dt=words[1]
  //}
   
    var globalDeviceId_string string
    globalDeviceId_string=words[0]



    //var globalDeviceId int64
    strconv.ParseInt(globalDeviceId_string, 10, 64)

     //fmt.Println(err)   

     //fmt.Println(globalDeviceId)   

    //ar cameraId string
    //cameraId,xxx:=svc.cfg.get_for_globalDeviceId(globalDeviceId)    
//cameraId,xxx,res:=svc.cfg.Get_for_globalDeviceId(globalDeviceId)
//if res==true{
//fmt.Println(xxx)

 //   cameraId_string:=strconv.FormatInt(cameraId,10)

 //   return nil, false
//}

return 0, false
 
}
func (svc *Axxon) get_storage_intervals(point string) intervals{

 //   //fmt.Println("[get_storage_depth]")    
 //   //fmt.Println("point: ",point)   

   // GET http://127.0.0.1:80/archive/statistics/depth/SERVER1/DeviceIpint.23/SourceEndpoint.video:0:0?threshold=2 


  // GET http://127.0.0.1:80/archive/contents/intervals/SERVER1/DeviceIpint.1/SourceEndpoint.video:0:0/past/future
   var src string
    
src = svc.request_to_axxon("archive/contents/intervals/"+strings.Replace(point,"hosts/","",1)+"/past/future?limit=1000")
//fmt.Println("src: ",src)




var m_struct intervals

err:=json.Unmarshal([]byte(src), &m_struct)
       if err != nil {
           //fmt.Println(err.Error())            
           
       }


//       //fmt.Println("количество интервалов: ",len(m_struct.Intervals))

       for i:=0;i<len(m_struct.Intervals);i++{
//fmt.Println(i," из ",len(m_struct.Intervals))
     m_struct.Intervals[i].Begin=svc.utc_to_local(m_struct.Intervals[i].Begin)
     m_struct.Intervals[i].End=svc.utc_to_local(m_struct.Intervals[i].End )    
}
       


      

//fmt.Println("m_struct: ",m_struct)

return m_struct

}

func (svc *Axxon) utc_to_local(point string) string{
//fmt.Println("")
//fmt.Println("")
//fmt.Println("[utc_to_local]")
//fmt.Println("")





//fmt.Println("point", point," len=",len(point))


//fmt.Println("[1]")
year, err := strconv.Atoi(point[0:4])
mouth, err := strconv.Atoi(point[4:6])
day, err := strconv.Atoi(point[6:8])

hour, err := strconv.Atoi(point[9:11])
min, err := strconv.Atoi(point[11:13])
sec, err := strconv.Atoi(point[13:15])

msec:="000000"
if len(point)==22{
msec=point[15:22]
}

if len(point)==15{
msec=".000000"
}

//fmt.Println("[2]")
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

//fmt.Println("[3]")
//var dt string=year+"-"+mouth+"-"+day+" "+hour+":"+min+":"+sec


// timeT, _ := time.Parse("2006-01-02 03:04:05", dt)
dt:=time.Date(year,time.Month(mouth),day,hour,min,sec,0,time.UTC)


year=dt.In(time.Local).Year()
mouth=int(dt.In(time.Local).Month())
day=dt.In(time.Local).Day()

hour,min,sec=dt.In(time.Local).Clock()
//fmt.Println("[5]")
//fmt.Println("year ", year)
//fmt.Println("mouth ", mouth)
//fmt.Println("day ",day)
//fmt.Println("hour ",hour)
//fmt.Println("min ",min)
//fmt.Println("sec ",sec)

var str_year,str_mouth,str_day,str_hour,str_min,str_sec string

str_year=strconv.Itoa(year)

if(mouth<10){
str_mouth="0"+strconv.Itoa(mouth)
}else{
str_mouth=strconv.Itoa(mouth)
}


if(day<10){
str_day="0"+strconv.Itoa(day)
}else{
str_day=strconv.Itoa(day)
}


if(hour<10){
str_hour="0"+strconv.Itoa(hour)
}else{
str_hour=strconv.Itoa(hour)
}


if(min<10){
str_min="0"+strconv.Itoa(min)
}else{
str_min=strconv.Itoa(min)
}


if(sec<10){
str_sec="0"+strconv.Itoa(sec)
}else{
str_sec=strconv.Itoa(sec)
}


//fmt.Println("[6]")

//fmt.Println("str_year ", str_year)
//fmt.Println("str_mouth ", str_mouth)
//fmt.Println("str_day ", str_day)
//fmt.Println("str_hour ", str_hour)
//fmt.Println("str_min ", str_min)
//fmt.Println("str_sec ", str_sec)
//fmt.Println("err",err)

//    t := time.Now()

    // For a time t, offset in seconds east of UTC (GMT)
//    _, offset := t.Local().Zone()
//    //fmt.Println(offset)

    // For a time t, format and display as UTC (GMT) and local times.
//   //fmt.Println(t.In(time.UTC))
//   //fmt.Println(t.In(time.Local))
//fmt.Println("point[13:22]: ",point[15:22])



res:=str_year+str_mouth+str_day+"T"+str_hour+str_min+str_sec+msec   

//fmt.Println("[7]")
//fmt.Println("res:", res)

return res
}   





//========================================================================= 

func (svc *Axxon) local_to_utc(point string) string{
  /*
//fmt.Println("")
//fmt.Println("")
//fmt.Println("")
//fmt.Println("[utc_to_local]")
//fmt.Println("")

//fmt.Println("point", point)
*/
var timestamp int=time.Now().In(time.UTC).Hour()-time.Now().In(time.Local).Hour()

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
dt:=time.Date(year,time.Month(mouth),day,hour,min,sec,0,time.UTC)


//dt:=time.Date(2021,7,19,1,2,3,0,time.UTC)


//fmt.Println(dt)

//fmt.Println("добавляем временную задержку: ",time.Duration(timestamp)*time.Hour)
dt=dt.Add(time.Duration(timestamp)*time.Hour)

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
year=dt.In(time.UTC).Year()
mouth=int(dt.In(time.UTC).Month())
day=dt.In(time.UTC).Day()

hour,min,sec=dt.In(time.UTC).Clock()

/*
//fmt.Println("len(point) ", len(point))

//fmt.Println("year ", year)
//fmt.Println("mouth ", mouth)
//fmt.Println("day ",day)
//fmt.Println("hour ",hour)
//fmt.Println("min ",min)
//fmt.Println("sec ",sec)
*/
var str_year,str_mouth,str_day,str_hour,str_min,str_sec string

str_year=strconv.Itoa(year)

if(mouth<10){
str_mouth="0"+strconv.Itoa(mouth)
}else{
str_mouth=strconv.Itoa(mouth)
}


if(day<10){
str_day="0"+strconv.Itoa(day)
}else{
str_day=strconv.Itoa(day)
}


if(hour<10){
str_hour="0"+strconv.Itoa(hour)
}else{
str_hour=strconv.Itoa(hour)
}


if(min<10){
str_min="0"+strconv.Itoa(min)
}else{
str_min=strconv.Itoa(min)
}


if(sec<10){
str_sec="0"+strconv.Itoa(sec)
}else{
str_sec=strconv.Itoa(sec)
}

str_msec:="000000"
if len(point)>21{
str_msec=point[15:22]
}

if len(point)<16{
str_msec=".000000"
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




var res string=str_year+str_mouth+str_day+"T"+str_hour+str_min+str_sec+str_msec

//fmt.Println("res:", res)

return res
}    

func (svc *Axxon) get_width(point string) int64{

var src string
    
src=svc.request_to_axxon("statistics/"+strings.Replace(point,"hosts/","",1)) 

type MyJsonName struct {
    Bitrate    int64 `json:"bitrate"`
    Fps        float64 `json:"fps"`
    Height     int64 `json:"height"`
    MediaType  int64 `json:"mediaType"`
    StreamType int64 `json:"streamType"`
    Width      int64 `json:"width"`
}

var m_struct MyJsonName
        
       
        err:=json.Unmarshal([]byte(src), &m_struct)
       if err != nil {
           //fmt.Println(err.Error())            
           return -1
       }

       //fmt.Println("Ширина кадра: ",m_struct.Width)





return m_struct.Width
}

func (svc *Axxon) get_storage_rtsp_stream(point string,dt string) string{

var src string
    
src = svc.request_to_axxon("archive/media/"+strings.Replace(point,"hosts/","",1)+"/"+dt+"?format=rtsp&speed=1") 

//fmt.Println("src: ",src)

type MyJsonName struct {
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

var m_struct MyJsonName

err:=json.Unmarshal([]byte(src), &m_struct)
       if err != nil {
           //fmt.Println(err.Error())            
           return ""
       }

//fmt.Println("path: ",m_struct.Rtsp.Path)
//fmt.Println("Port: ",m_struct.Rtsp.Port)       

var res string = "rtsp://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+m_struct.Rtsp.Port+"/"+m_struct.Rtsp.Path
//fmt.Println("res: ",res) 

return res   
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


    for i:=0;i<len( svc.devices);i++{
      /*
    //fmt.Println("---")
    //fmt.Println("id: ",svc.devices[i].Id)   
    //fmt.Println("cameraId: ",svc.devices[i].CameraId)  
   //fmt.Println("name: ",svc.devices[i].Name) 
*/
   if command.DeviceId==svc.devices[i].Id{
    d := svc.devices[i]
   

      switch command.Command {
            case 100:

     //fmt.Println("переход по пресету")  

//Получить телеметри Айди
var tmtr_Id = d.TelemetryControlID
var session_id=svc.get_SessionID_value_from_Axxon(tmtr_Id)


svc.request_to_axxon("control/telemetry/preset/go"+tmtr_Id+"?pos="+strconv.FormatInt(command.Argument,10)+"&session_id="+strconv.FormatInt(session_id,10)) 

//Остановить текущий сеанс управления телеметрией если он есть

//Перейти по пресету с заданным айдишником

//Остановить сеанс управления телеметрией

     return svc.request_URL_handler(d.CameraId,"undefined","local"),true     


         
            default:
         }   




 
    
    }

   }


    






    return "", false   
}

func (svc *Axxon)  request_intervals(cid int64, data []byte) (interface{}, bool){
  //  //fmt.Println("") 

    cameraId:=strings.Replace(string(data[:]) ,"\"","",2)  
    //fmt.Println("request intervals for camera id: ",cameraId)
    var needed_camera Camera
for i:=0;i<len(svc.m_camera_list.Cameras);i++{


    var m_camera=svc.m_camera_list.Cameras[i]
    //fmt.Println(i, "); id: ",m_camera.DisplayID) 
    if  m_camera.DisplayID==cameraId{
     //fmt.Println("Найдена камера ",m_camera.DisplayName, ";ID: ",m_camera.DisplayID)
     needed_camera=m_camera                

    }    
    }

    var needed_point string
    needed_point=""

    //fmt.Println("количество точек: ",len(needed_camera.VideoStreams))
    var width int64
    width=0
    //fmt.Println(width)
    for i:=0;i<len(needed_camera.VideoStreams);i++{

    var point string=needed_camera.VideoStreams[i].AccessPoint
    //fmt.Println(needed_camera.VideoStreams[i].AccessPoint)


    var this_width int64
    this_width=svc.get_width(point)
    if width<this_width{
    width=this_width  
    needed_point=point      
    }


    }
    if needed_point==""{
      needed_point=needed_camera.VideoStreams[0].AccessPoint

    }

    //fmt.Println("Нужный стрим 3: ",needed_point)

//    my_intervals=svc.get_storage_intervals(needed_point)
    my_intervals:=svc.get_storage_intervals(needed_point)

        type MyJsonName struct {
   
    Intervals intervals `json:"intervals"`  
}
var m_struct []MyJsonName 
m_struct=append(m_struct,MyJsonName{Intervals:my_intervals})


    return m_struct, false
} 

func (svc *Axxon) ResetAlarm(cid int64, data []byte) (interface{}, bool) {

  fmt.Println("[ResetAlarm]") 
    var id int64
    json.Unmarshal(data, &id)
    //svc.cfg.DeleteDevice(id)

    svc.RLock()
    defer svc.RUnlock()


    for i:=0;i<len(svc.devices);i++{


        fmt.Println("id: ",id," svc.devices[i].id: ",svc.devices[i].Id )

      if(svc.devices[i].Id==id){

      events := api.EventsList{{
            ServiceId: svc.Settings.Id,
            ServiceName: svc.Settings.Title,
            DeviceId: svc.devices[i].Id,
            DeviceName: svc.devices[i].Name,
            UserId: cid,
            Class: api.EC_INFO_ALARM_RESET}}
        svc.Broadcast("Events", events)
        
        return true, true // broadcast

      }
    }
    

    

   
    return false, false // don't broadcast
}   



/*
Получить список видеокамер.

Для камеры

Получить список видеопотоков

1 Сравнить видеопотоки по разрешению
Упорядочить по убыванию качетсва



*/

/*
По завершению потока отправлять запрос на поиск нового
по этому же времени

Перед отправкой запроса проверяй есть ли это время в архиве
Если нет то не  шли ничего
*/

/*

По запросу потока из архива
Искать поток начиная с архива ывидеопотока высшего качества


*/

/*



*/
