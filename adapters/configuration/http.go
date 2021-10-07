package configuration

import (
    "net/http"
    "strconv"
    "strings"
    "bytes"
    "time"
    "fmt"
)

const (
    PayloadLimit = 16 * 1024 * 1024
)

func (cfg *Configuration) HTTPHandler(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(r.URL.Path, "/")
    if 3 != len(parts) || "" == parts[2] {
        cfg.Err("Wrong HTTP request: invalid path")
        return
    }
    resource := parts[2]
    r.ParseForm()
    switch resource {
        case "plan": cfg.planHTTPHandler(w, r)
        case "user": cfg.userHTTPHandler(w, r)
        case "journal": cfg.journalHTTPHandler(w, r)
        default: http.NotFound(w, r)
    }
}

func (cfg *Configuration) planHTTPHandler(w http.ResponseWriter, r *http.Request) {
    if "POST" == r.Method {
        //f, err := os.OpenFile("./downloaded", os.O_WRONLY|os.O_CREATE, 0644)
        //catch(err)
        //defer f.Close()
        //reader := &io.LimitedReader{R: r.Body, N: PayloadLimit}
        //io.Copy(f, reader)
        r.Body = http.MaxBytesReader(w, r.Body, PayloadLimit)
        
        id := getIntVal(r.Form["id"])
        buf := new(bytes.Buffer)
        buf.ReadFrom(r.Body)
        cfg.dbUpdatePlanPicture(int64(id), buf.Bytes())
    } else if "GET" == r.Method {
        id, _ := strconv.Atoi(r.Form["id"][0])
        picture := cfg.dbLoadPlanPicture(int64(id))
        if 0 == len(picture) {
            http.NotFound(w, r)
        } else {
            w.Write(picture)
        }
    }    
}

func (cfg *Configuration) userHTTPHandler(w http.ResponseWriter, r *http.Request) {
    id := getIntVal(r.Form["id"])
    if "POST" == r.Method {
        r.Body = http.MaxBytesReader(w, r.Body, PayloadLimit)
        buf := new(bytes.Buffer)
        buf.ReadFrom(r.Body)
        cfg.dbUpdateUserPicture(int64(id), buf.Bytes())
    } else if "GET" == r.Method {
        picture := cfg.dbLoadUserPicture(int64(id))
        if 0 == len(picture) {
            http.NotFound(w, r)
        } else {
            w.Write(picture)
        }
    }    
}

func (cfg *Configuration) journalHTTPHandler(w http.ResponseWriter, r *http.Request) {
    ths := []string{"#", "Время", "Источник", "Устройство", "Событие", "Пользователь", "Причины", "Принятые меры"}
    if "GET" != r.Method {
        http.NotFound(w, r)
        return
    }
    start, _ := time.Parse(time.RFC3339, getStringVal(r.Form["start"]))
    end, _ := time.Parse(time.RFC3339, getStringVal(r.Form["end"]))
    filter := EventFilter{
        Start: start,
        End: end,
        ServiceId: int64(getIntVal(r.Form["serviceId"])),
        UserId: int64(getIntVal(r.Form["userId"])),
        Limit: int64(getIntVal(r.Form["limit"])),
        Class: getIntVal(r.Form["class"])}

    html := `<html><table border="1" cellpadding="3" cellspacing="0"><tr>`
    for _, th := range ths {
        html += "<th>" + th + "</th>"
    }
    html += "</tr>\n"
    
    events := cfg.loadEvents(&filter)
    for i := range events {
        html += fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
                           events[i].Id,
                           time.Unix(events[i].Time, 0),
                           events[i].ServiceName,
                           events[i].DeviceName,
                           events[i].Text,
                           events[i].UserName,
                           events[i].Reason,
                           events[i].Reaction)
    }
    
    html += "</table>\n<script>window.print()</script></html>"
    w.Write([]byte(html))
}

func getIntVal(list []string) (n int) {
    if nil != list && len(list) > 0 {
        n, _ = strconv.Atoi(list[0])
    }
    return
}

func getStringVal(list []string) (s string) {
    if nil != list && len(list) > 0 {
        s = list[0]
    }
    return
}