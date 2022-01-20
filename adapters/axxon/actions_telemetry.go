package axxon

import (
//    "log"
    "fmt"
   // "../../api"
	"strings"
    "encoding/json" 
    "strconv"	
)



func (svc *Axxon) get_sessionId(cid int64, point string) (int64, int64, int64){ 

const (
	YOU_HAVE_ACCESS = 0
	YOU_YAVE_NO_ACCESS = 1
	ANOTHER_USER_USE_CAMERA = 2
	
)
//fmt.Println("cid               ",cid)
//filter := core.Authorize(cid, svc.Settings.Id, api.AM_WATCH | api.AM_CONTROL)
    // TODO: real filter
    filter, _ := core.Authorize(cid, []int64{})


//Проверка на доступ
//В devices найти элемент с данным point
//fmt.Println("...")
 for i:=0;i<len( svc.devices);i++{

//fmt.Println("Name               ",svc.devices[i].Name)
//fmt.Println("TelemetryControlID ",svc.devices[i].TelemetryControlID)


//fmt.Println(".")

if svc.devices[i].TelemetryControlID==point{
//fmt.Println("!!! THIS !!!")

svc.devices[i].AccessMode = filter[svc.devices[i].Id]
//fmt.Println("AccessMode         ",svc.devices[i].AccessMode)

if svc.devices[i].AccessMode<2&&filter[0]<2{

//fmt.Println("!!! NO ACCESS !!!")
return -1,YOU_YAVE_NO_ACCESS, -1
}

}

}
//fmt.Println("...")
    
	//fmt.Println("Клиент: ",cid," запрашивает ключ на точку ",point)
	var sessionId int64
	sessionId=-1
 	//fmt.Println("Количество сессий: ",len(svc.telemetrySessions))

 	free_camera:=true


    for _, session := range svc.telemetrySessions {
 	//fmt.Println(idx)

	 	if session.point==point{
	 		free_camera=false

			//fmt.Println("Найдена точка: ",session.point)

			if session.cid==cid{
			//fmt.Println("И она занята нами")
			//Берем ее sessionId из списка, а пока так:

			//sessionId=session.key
			sessionId=svc.get_SessionID_value_from_Axxon(point)
				

			}else{
			//fmt.Println("И она занята ДРУГИМ клиентом")	



			return -1, ANOTHER_USER_USE_CAMERA, session.cid
			}	
	 	}	
 	}

 	if free_camera==true{
			//fmt.Println("Камера свободна. Берем управление на себя:")
			sessionId=svc.get_SessionID_value_from_Axxon(point)
svc.telemetrySessions[len(svc.telemetrySessions)]=telemetrySession{cid:cid, point:point,key : sessionId,livetime:8}


 	}


 			


 



 	
 	////fmt.Println("SessionID: ",telemetry_id)	


 	return sessionId, YOU_HAVE_ACCESS, -1
 	

}

func (svc *Axxon) Telemetry_command(cid int64, data []byte) (interface{}, bool){ 




    //fmt.Println("\n")	
    //fmt.Println("[Telemetry_command]")
    //fmt.Println("\n")    
    var str string
    str=string(data[:])
    str=strings.Replace(str,"\"","",2)
    ////fmt.Println("data: ",str)

    words := strings.Fields(str)

    	type feedback struct{
		Name string `json:"name"`
		Data interface{} `json:"data"`
	}

	var answer feedback
/*
    for idx, word := range words {
		//fmt.Println("Word %d is: %s\n", idx, word)
	}
	*/
	command:=words[0]	
	point:=words[1]
	sessionId,res,another_cid:=svc.get_sessionId(cid,point)	

	//fmt.Println("Камера point: ",point)

	if res==2{

		user:=core.GetUser_for_Axxon(another_cid)

		fmt.Println("Камерой управляет другой пользователь:\n")
		fmt.Println(user.Name," ",user.Surename)
 		answer.Name="Another_user"

 		type another_user struct{
     	Name  string      `json:"name"`	  
    	Surename  string      `json:"surename"`
 		}



 		answer.Data=another_user{Name:user.Name,Surename:user.Surename}
	}

	if res==1{

		core.GetUser_for_Axxon(cid)

		//fmt.Println("Камерой управляет другой пользователь:\n")
		//fmt.Println(user.Name," ",user.Surename)
 		answer.Name="No_access"





 		answer.Data=""
	}	



 switch command {
 
    case "Telemetry_move":
    	svc.Telemetry_move(strconv.FormatInt(sessionId,10),point,words[2],words[3],words[4])	
     


    case "Telemetry_focus_in":
    	svc.request_to_axxon("control/telemetry/focus"+point+"?mode=continuous&value=0.5&session_id="+strconv.FormatInt(sessionId,10)) 	

    case "Telemetry_focus_out":
    	svc.request_to_axxon("control/telemetry/focus"+point+"?mode=continuous&value=-0.5&session_id="+strconv.FormatInt(sessionId,10)) 	
           
    case "Telemetry_stop_focus":
    	svc.request_to_axxon("control/telemetry/focus"+point+"?mode=continuous&value=0&session_id="+strconv.FormatInt(sessionId,10)) 


     case "Telemetry_zoom_in":
    	svc.request_to_axxon("control/telemetry/zoom"+point+"?mode=continuous&value=0.5&session_id="+strconv.FormatInt(sessionId,10)) 	

    case "Telemetry_zoom_out":
    	svc.request_to_axxon("control/telemetry/zoom"+point+"?mode=continuous&value=-0.5&session_id="+strconv.FormatInt(sessionId,10)) 	
           
     case "Telemetry_stop_zoom":
    	svc.request_to_axxon("control/telemetry/zoom"+point+"?mode=continuous&value=0&session_id="+strconv.FormatInt(sessionId,10)) 	       
   

 	case "Telemetry_preset_info":
 		preset_info:=svc.Telemetry_preset_info(point)
 		//fmt.Println(preset_info)
 		answer.Name="preset_info"
 		answer.Data=preset_info
    	


     case "Telemetry_go_to_preset":
    	svc.Telemetry_go_to_preset(point,words[2],strconv.FormatInt(sessionId,10)) 	  


  	case "Telemetry_add_preset":
 		preset_info:=svc.Telemetry_add_preset(point,words[2],strconv.FormatInt(sessionId,10))
 		//fmt.Println(preset_info)
 		answer.Name="preset_info"
 		answer.Data=preset_info   

  	case "Telemetry_remove_preset":
 		preset_info:=svc.Telemetry_remove_preset(point,words[2],strconv.FormatInt(sessionId,10))
 		//fmt.Println(preset_info)
 		answer.Name="preset_info"
 		answer.Data=preset_info    		

  	case "Telemetry_edit_preset":
 		preset_info:=svc.Telemetry_edit_preset(point,words[2],words[3],strconv.FormatInt(sessionId,10))
 		//fmt.Println(preset_info)
 		answer.Name="preset_info"
 		answer.Data=preset_info    			     
    


    }			


 return answer,false
}


func (svc *Axxon) Telemetry_move(sessionId string,point string,_x string, _y string, _val string) (interface{}, bool){ 




  

    /*
	for idx, word := range words {
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
 */	
	//var mx float32 = 0

	var res=1
	var mx float64=0
	var my float64=0
	var mval float64=0	


	x, _ := strconv.ParseFloat(_x, 64); 

    mx=x
	////fmt.Println("err: ",err) 

    ////fmt.Println("mx: ",mx) 

    

    if y, err := strconv.ParseFloat(_y, 64); err == nil {
    //fmt.Println("y: ",y) 
    my=y
	}else{
    	res=0
    }


    if val, err := strconv.ParseFloat(_val, 64); err == nil {
    //fmt.Println("val: ",val) 
    mval=val
	}else{
    	res=0
    }

 

//    //fmt.Println("Клиент ",cid,"; Ключ ",sessionId,"; Точка ",words[1])

    if res==1{

    	
    svc.tlmtr_move(sessionId,point,mx, my, mval) 

   // tlmtr_move(session string, point string,x float64, y float64, value float64){
	
    }
	   	





 /* var x=data[0]
  var y=data[1]
  var val=data[2]
  ////fmt.Println(x)
  ////fmt.Println(y)
  ////fmt.Println(val)  
*/


svc.hold_Session(point, sessionId)
    return svc.stream_from_storage,false
}

func (svc *Axxon) Telemetry_hold_session(cid int64, data []byte) (interface{}, bool){ 
	//fmt.Println("\n")
    //fmt.Println("[Telemetry_hold_session]")
    //fmt.Println("\n ")

    var src string
    src=string(data[:])
    ////fmt.Println("data: ",src)	
 	src=strings.Replace(src,"\"","",2)   
    words := strings.Split(src," ")


    svc.hold_Session(words[1], words[0])

    return svc.stream_from_storage,false
}





func (svc *Axxon) Telemetry_stop_moving(cid int64, data []byte) (interface{}, bool){ 
    svc.Log("[Telemetry_stop_moving]")

    var src string
    src=string(data[:])
    ////fmt.Println("data: ",src)	
 	src=strings.Replace(src,"\"","",2)   
    words := strings.Split(src," ")
svc.tlmtr_move(words[0],words[1],0, 0, 0)    

    return svc.stream_from_storage,false
}

func (svc *Axxon) Telemetry_go_to_preset(point, preset, sessionId string) {
svc.Log("[Telemetry_go_to_preset]")



    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
*/

 //   var point string
 //   point=string(data[:])
    ////fmt.Println(point)	
//ET http://IP-адрес:порт/префикс/control/telemetry/preset/remove/HOSTNAME/DeviceIpint.23/TelemetryControl.0?pos=2&session_id=0
	svc.request_to_axxon("control/telemetry/preset/go"+point+"?pos="+preset+"&session_id="+sessionId)

	svc.hold_Session(point, sessionId)
   
}  


func (svc *Axxon) Telemetry_add_preset(point,preset_name,sessionId string) interface{}{


svc.Log("[Telemetry_add_preset]")



    



 

    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
*/
  
  var ind int
ind=svc.get_free_preset_index(point)     

////fmt.Println("free_index: ",ind)	

//ET http://IP-адрес:порт/префикс/control/telemetry/preset/remove/HOSTNAME/DeviceIpint.23/TelemetryControl.0?pos=2&session_id=0
svc.request_to_axxon("control/telemetry/preset/set"+point+"?pos="+strconv.Itoa(ind)+"&label="+preset_name+"&session_id="+sessionId) 

//------------------
    var src string

src=svc.request_to_axxon("control/telemetry/preset/info"+point) 

    src=strings.Replace(src,"{","",1)
    src=strings.Replace(src,"}","",1)  
	
	words := strings.Split(src,",")

//	var count=len(words)
	////fmt.Println(count)

type preset struct{
    Id int 
    Name string

}
//var m_presets []preset
var s []preset

	for _, word := range words {

		var str=word
		var str1 string

		str1=strings.Split(str,":")[0]
		str1=strings.Replace(str1,"\n","",1)
		str1=strings.Replace(str1,"\"","",2)	
	    str1=strings.Replace(str1," ","",-1)	
		////fmt.Println("str1 0: ",str1)				

		var str2 string

		value, _ := strconv.Atoi(str1)
	
		////fmt.Println("err: ",err)
		str2=strings.Replace((strings.Split(str,":")[1]),"\"","",2)

//dt=strings.Replace(dt,"\"","",2)

s = append(s,preset{Id: value, Name: str2})

		//s.append(s,preset{id: idx, name: str})
	////fmt.Println(idx)
}

////fmt.Println(s)


//////fmt.Println("[presets] ",m_presets)


    return s

    

   
}    


func (svc *Axxon) Telemetry_remove_preset(point,preset_index,sessionId string) interface{}{
svc.Log("[Telemetry_go_to_preset]")



    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
*/
     

    

//ET http://IP-адрес:порт/префикс/control/telemetry/preset/remove/HOSTNAME/DeviceIpint.23/TelemetryControl.0?pos=2&session_id=0
svc.request_to_axxon("control/telemetry/preset/remove"+point+"?pos="+preset_index+"&session_id="+sessionId) 

//------------------
    var src string

src=svc.request_to_axxon("control/telemetry/preset/info"+point) 

    src=strings.Replace(src,"{","",1)
    src=strings.Replace(src,"}","",1)  
	
////fmt.Println("src: ",src)

	////fmt.Println("len: ",len(src))	
	if(len(src)>3){

	words := strings.Split(src,",")

	//var count=len(words)
	////fmt.Println(count)

type preset struct{
    Id int 
    Name string

}
//var m_presets []preset
var s []preset

	for _, word := range words {

		var str=word
		var str1 string

		str1=strings.Split(str,":")[0]
		str1=strings.Replace(str1,"\n","",1)
		str1=strings.Replace(str1,"\"","",2)	
	    str1=strings.Replace(str1," ","",-1)	
		////fmt.Println("str1 1: ",str1)				

		var str2 string

		value, _ := strconv.Atoi(str1)
	
		////fmt.Println("err: ",err)
		str2=strings.Replace((strings.Split(str,":")[1]),"\"","",2)

//dt=strings.Replace(dt,"\"","",2)

s = append(s,preset{Id: value, Name: str2})

		//s.append(s,preset{id: idx, name: str})
	////fmt.Println(idx)
}

////fmt.Println(s)


//////fmt.Println("[presets] ",m_presets)


    return s
}
    return -1

   
} 

func (svc *Axxon) Telemetry_edit_preset(point, index, name, sessionId string) interface{}{
svc.Log("[Telemetry_go_to_preset]")


//ET http://IP-адрес:порт/префикс/control/telemetry/preset/remove/HOSTNAME/DeviceIpint.23/TelemetryControl.0?pos=2&session_id=0
svc.request_to_axxon("control/telemetry/preset/set"+point+"?pos="+index+"&label="+name+"&session_id="+sessionId) 

//------------------
    var src string

src=svc.request_to_axxon("control/telemetry/preset/info"+point) 

    src=strings.Replace(src,"{","",1)
    src=strings.Replace(src,"}","",1)  
	
	words := strings.Split(src,",")

	//var count=len(words)
	////fmt.Println(count)

type preset struct{
    Id int 
    Name string

}
//var m_presets []preset
var s []preset

	for _, word := range words {

		var str=word
		var str1 string

		str1=strings.Split(str,":")[0]
		str1=strings.Replace(str1,"\n","",1)
		str1=strings.Replace(str1,"\"","",2)	
	    str1=strings.Replace(str1," ","",-1)	
		////fmt.Println("str1 2: ",str1)				

		var str2 string

		value, _ := strconv.Atoi(str1)
	
		////fmt.Println("err: ",err)
		str2=strings.Replace((strings.Split(str,":")[1]),"\"","",2)

//dt=strings.Replace(dt,"\"","",2)

s = append(s,preset{Id: value, Name: str2})

		//s.append(s,preset{id: idx, name: str})
	////fmt.Println(idx)
}

////fmt.Println(s)


//////fmt.Println("[presets] ",m_presets)


    return s
 


 
}    

func (svc *Axxon) Telemetry_preset_info(point string) interface{}{
svc.Log("[Telemetry_preset_info")

/*
 var str string
    str=string(data[:])
    ////fmt.Println("data: ",str)	
 	str=strings.Replace(str,"\"","",2)   
    strs := strings.Split(str," ")

    
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}

	point:=strs[1]

		
*/



type preset struct{
    Id int 
    Name string

}



	var s []preset
	//printSlice(s)

	// append works on nil slices.
	//s = append(s,preset{id: 1, name: "str"})
	////fmt.Println(s)

	

/*

    point=strings.Replace(point,"\"","",2)
    point=strings.Replace(point,"{","",1)
    point=strings.Replace(point,"}","",1)        
   ////fmt.Println("[1]")

*/
src:=svc.request_to_axxon("control/telemetry/preset/info"+point) 
   ////fmt.Println("[2]")
	////fmt.Println("src: ",src)

	////fmt.Println("len: ",len(src))	
	if(len(src)>3){
	////fmt.Println("[1]")	
    src=strings.Replace(src,"{","",1)
    src=strings.Replace(src,"}","",1)  
	
	words := strings.Split(src,",")

	//var count=len(words)
	////fmt.Println(count)


//var m_presets []preset


	for _, word := range words {

		var str=word
		var str1 string

		str1=strings.Split(str,":")[0]
		str1=strings.Replace(str1,"\n","",1)
		str1=strings.Replace(str1,"\"","",2)	
	    str1=strings.Replace(str1," ","",-1)	
		////fmt.Println("str1 3: ",str1)				

		var str2 string

		value, _ := strconv.Atoi(str1)
	
		////fmt.Println("err: ",err)
		str2=strings.Replace((strings.Split(str,":")[1]),"\"","",2)

//dt=strings.Replace(dt,"\"","",2)
s = append(s,preset{Id: value, Name: str2})

		//s.append(s,preset{id: idx, name: str})
	////fmt.Println(idx)
}

//fmt.Println(s)


//////fmt.Println("[presets] ",m_presets)


    return s
}
	////fmt.Println("[2]")	

	
return svc.stream_from_storage
}


func (svc *Axxon) tlmtr_move(session string, point string,x float64, y float64, value float64){

//	svc.Log("move "+strconv.FormatInt(x,10)+" "+strconv.FormatInt(y,10))
	
	var mx float64
	var my float64

	mx=0
	my=0




	if value>1{
	value=0
	}

	if value<0{
	value=0
	}
	

	if x>0	{
	mx=value
	}

	if x<0	{
	mx=-1*value
	}



	if y>0	{
	my=value
	}

	if y<0	{
	my=-1*value	
	}
	
	svc.Log("      mx "+fmt.Sprint(mx))
	svc.Log("speed_x "+fmt.Sprint(svc.current_speed_x))

	svc.Log("      my "+fmt.Sprint(my))
	svc.Log("speed_y "+fmt.Sprint(svc.current_speed_y))	

	if (mx!=svc.current_speed_x)||(my!=svc.current_speed_y)	{
	svc.current_speed_x=mx
	svc.current_speed_y=my		
svc.request_to_axxon("control/telemetry/move"+point+"?mode=continuous&pan="+fmt.Sprint(mx)+"&tilt="+fmt.Sprint(my)+"&session_id="+session) 


	
	}
	
}
//----------

func (svc *Axxon) Telemetry_focus_in(cid int64, data []byte) (interface{}, bool){ 
    svc.Log("[Telemetry_focus_in]")
    var str string
    str=string(data[:])
    ////fmt.Println("data: ",str)	
 	str=strings.Replace(str,"\"","",2)   
    strs := strings.Split(str," ")

    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
*/

//GET http://127.0.0.1:80/control/telemetry/focus/HOSTNAME/DeviceIpint.25/TelemetryControl.0??mode=continuous&value=1&session_id=1       
svc.request_to_axxon("control/telemetry/focus"+strs[1]+"?mode=continuous&value=0.5&session_id="+strs[0]) 

    return svc.stream_from_storage,false
}

func (svc *Axxon) Telemetry_focus_out(cid int64, data []byte) (interface{}, bool){ 
    svc.Log("[Telemetry_focus_out]")
    var str string
    str=string(data[:])
    ////fmt.Println("data: ",str)	
 	str=strings.Replace(str,"\"","",2)   
    strs := strings.Split(str," ")

    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
	*/

svc.request_to_axxon("control/telemetry/focus"+strs[1]+"?mode=continuous&value=-0.5&session_id="+strs[0]) 
    return svc.stream_from_storage,false
}

func (svc *Axxon) Telemetry_stop_focus(cid int64, data []byte) (interface{}, bool){ 
    svc.Log("[Telemetry_stop_focus]")
    var str string
    str=string(data[:])
    ////fmt.Println("data: ",str)	
 	str=strings.Replace(str,"\"","",2)   
    strs := strings.Split(str," ")

    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
	*/

svc.request_to_axxon("control/telemetry/focus"+strs[1]+"?mode=continuous&value=0&session_id="+strs[0]) 

    return svc.stream_from_storage,false
}

//--------------------------------------------------------------


//----------

func (svc *Axxon) Telemetry_zoom_in(cid int64, data []byte) (interface{}, bool){ 
    svc.Log("[Telemetry_zoom_in]")

var str string
    str=string(data[:])
    ////fmt.Println("data: ",str)	
 	str=strings.Replace(str,"\"","",2)   
    strs := strings.Split(str," ")


    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
*/

//GET http://127.0.0.1:80/control/telemetry/zoom/HOSTNAME/DeviceIpint.25/TelemetryControl.0??mode=continuous&value=1&session_id=1       
svc.request_to_axxon("control/telemetry/zoom"+strs[1]+"?mode=continuous&value=0.5&session_id="+strs[0]) 

    return svc.stream_from_storage,false
}

func (svc *Axxon) Telemetry_zoom_out(cid int64, data []byte) (interface{}, bool){ 
    svc.Log("[Telemetry_zoom_out]")

    var str string
    str=string(data[:])
    ////fmt.Println("data: ",str)	
 	str=strings.Replace(str,"\"","",2)   
    strs := strings.Split(str," ")

    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	}
	*/


svc.request_to_axxon("control/telemetry/zoom"+strs[1]+"?mode=continuous&value=-0.5&session_id="+strs[0]) 
    return svc.stream_from_storage,false
}

func (svc *Axxon) Telemetry_stop_zoom(cid int64, data []byte) (interface{}, bool){ 


    svc.Log("[Telemetry_stop_zoom]")

    var str string
    str=string(data[:])
    ////fmt.Println("data: ",str)	
 	str=strings.Replace(str,"\"","",2)   
    strs := strings.Split(str," ")

    /*
	for idx, word := range strs{
		//fmt.Printf("Word %d is: %s\n", idx, word)
	} 
	*/

svc.request_to_axxon("control/telemetry/zoom"+strs[1]+"?mode=continuous&value=0&session_id="+strs[0]) 

    return svc.stream_from_storage,false
}

//--------------------------------------------------------------
//"http://192.168.0.187:8000/control/telemetry/session/acquire/ASTRAAXXON/DeviceIpint.1/TelemetryControl.0?session_priority=2"

func (svc *Axxon) get_SessionID_value_from_Axxon(point string) int64{

var src string  
    
 //  src = svc.request_to_axxon("control/telemetry/session/acquire"+point+"/TelemetryControl.0?session_priority=2") 

src =  svc.request_to_axxon("control/telemetry/session/acquire"+point+"?session_priority=2") 

type Autogenerated struct {
	SessionID int64 `json:"session_id"`
}

var m_struct Autogenerated

err:=json.Unmarshal([]byte(src), &m_struct)
    if err != nil {
        ////fmt.Println(err.Error()) 
        return -1         
    }

////fmt.Println("SessionID: ",m_struct.SessionID)

/*

 //   ////fmt.Println("Description",m_struct.Rtsp.Description)
 //   ////fmt.Println("Path",m_struct.Rtsp.Path)
 //   ////fmt.Println("Port",m_struct.Rtsp.Port)

     ////fmt.Println(strings.Split(m_struct.Rtsp.Path,"&id")[0])
*/
    var res int64

    res=m_struct.SessionID

    
    
  //  res="rtsp://"+svc.username+":"+svc.password+"@"+svc.ipaddr+":"+m_struct.Rtsp.Port+"/"+strings.Split(m_struct.Rtsp.Path,"&id")[0]
    //////fmt.Println("URL",res)
    return res
}

//Получение информации о степенях свободы
//GET http://127.0.0.1:80/control/telemetry/info/Server1/DeviceIpint.2/TelemetryControl.0
func (svc *Axxon) get_tmtr_settings(point string) {

//var src string  
    
 
//svc.request_to_axxon("control/telemetry/info/hosts"+point) 
svc.request_to_axxon("control/telemetry/info"+point) 

}


//Поддержание актуальности сессии
//GET http://127.0.0.1:80/control/telemetry/session/keepalive/Server1/DeviceIpint.2/TelemetryControl.0?session_id=1

func (svc *Axxon) hold_Session(point string ,key string ) {
 svc.Log(point)
  svc.Log(key)	

      if len(svc.telemetrySessions)>0{
        //fmt.Println("Количество сессий: ",len(svc.telemetrySessions))
        for idx, session := range svc.telemetrySessions {
            //fmt.Println("idx: ",idx,"; cid: ",session.cid,"; point: ",session.point,"; key: ",session.key,"; livetime: ",session.livetime)

            if strconv.FormatInt(session.key,10)==key{
         //   session.livetime=session.livetime-1

            svc.telemetrySessions[idx]=telemetrySession{cid:session.cid, point:session.point,key : session.key,livetime:5}

            }
            

        }


        }



svc.request_to_axxon("control/telemetry/session/keepalive"+point+"?session_id="+key)

}

func (svc *Axxon) get_free_preset_index(point string) int{
 


		




type preset struct{
    Id int 
    Name string

}


	var index_array[]int


	var s []preset
	//printSlice(s)

	// append works on nil slices.
	//s = append(s,preset{id: 1, name: "str"})
	////fmt.Println(s)

	



  
    var src string
src=svc.request_to_axxon("control/telemetry/preset/info"+point) 
	////fmt.Println("src: ",src)

	////fmt.Println("len: ",len(src))	
	if(len(src)>3){
    src=strings.Replace(src,"{","",1)
    src=strings.Replace(src,"}","",1)  
	
	words := strings.Split(src,",")

	//var count=len(words)
	////fmt.Println(count)


//var m_presets []preset


	for _, word := range words {

		var str=word
		var str1 string

		str1=strings.Split(str,":")[0]
		str1=strings.Replace(str1,"\n","",1)
		str1=strings.Replace(str1,"\"","",2)	
	    str1=strings.Replace(str1," ","",-1)	
		////fmt.Println("str1 4: ",str1)				

		var str2 string

		value, _ := strconv.Atoi(str1)
	
		////fmt.Println("err: ",err)
		str2=strings.Replace((strings.Split(str,":")[1]),"\"","",2)

//dt=strings.Replace(dt,"\"","",2)
index_array = append(index_array,value)
s = append(s,preset{Id: value, Name: str2})

		//s.append(s,preset{id: idx, name: str})
	////fmt.Println(idx)
}



////fmt.Println("index_array: ",index_array)


var res bool
res=false 
var i,x int
for i=0;i<50;i++{
	res=true
	x=0
for x=0;x<len(index_array);x++{
////fmt.Println("[",i," ",index_array[x],"]")
if(i==index_array[x]){
res=false
}	


}
if(res==true){
	return i

}



}
}

return 0

}




