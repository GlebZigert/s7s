package configuration

import (
    "net/http"
    "strconv"
    "strings"
    "errors"
    "bytes"
    "time"
    "fmt"
)

const (
    PayloadLimit = 16 * 1024 * 1024
)
var argumentError = errors.New("HTTP argument error")

func (cfg *Configuration) HTTPHandler(w http.ResponseWriter, r *http.Request) (err error) {
    parts := strings.Split(r.URL.Path, "/")
    if 3 != len(parts) || "" == parts[2] {
        cfg.Err("Wrong HTTP request: invalid path")
        http.NotFound(w, r)
        return
    }
    resource := parts[2]
    r.ParseForm()
    handler := httpHandlers[resource]
    if nil != handler {
        err = handler(cfg, w, r)
        if errors.Is(err, argumentError) {
            // TODO: log argument err?
            httpStatus(w, http.StatusBadRequest)
        } else {
            // TODO: log db error
        }
    } else {
        http.NotFound(w, r)
    }
    return
}

var httpHandlers = map[string] func(*Configuration, http.ResponseWriter, *http.Request) error {
    ///////////////////////////////////////////////////////////////////////////////////
"plan": func (cfg *Configuration, w http.ResponseWriter, r *http.Request) (err error) {
    id, err := getIntVal(r.Form["id"])
    if nil != err {
        return fmt.Errorf("%w: id", argumentError)
    }

    if "POST" == r.Method {
        r.Body = http.MaxBytesReader(w, r.Body, PayloadLimit)
        buf := new(bytes.Buffer)
        buf.ReadFrom(r.Body)
        err = cfg.dbUpdatePlanPicture(int64(id), buf.Bytes())
        if nil != err {
            return err // due to global err shadowed
        }
        cfg.Broadcast("PlanUpload", id)
    } else if "GET" == r.Method {
        picture, err := cfg.dbLoadPlanPicture(int64(id))
        if nil != err {
            return err // due to global err shadowed
        }
        if 0 == len(picture) {
            http.NotFound(w, r)
        } else {
            w.Write(picture)
        }
    }
    return
},
///////////////////////////////////////////////////////////////////////////////////
"user": func(cfg *Configuration, w http.ResponseWriter, r *http.Request) (err error) {
    id, err := getIntVal(r.Form["id"])
    if nil != err {
        return fmt.Errorf("%w: id", argumentError)
    }
    if "POST" == r.Method {
        r.Body = http.MaxBytesReader(w, r.Body, PayloadLimit)
        buf := new(bytes.Buffer)
        buf.ReadFrom(r.Body)
        err = cfg.dbUpdateUserPicture(int64(id), buf.Bytes())
        if nil != err {
            return
        }
        cfg.Broadcast("UserUpload", id)
    } else if "GET" == r.Method {
        picture, err := cfg.dbLoadUserPicture(int64(id))
        if nil != err {
            return err // global err shadowed
        }
        if 0 == len(picture) {
            http.NotFound(w, r)
        } else {
            w.Write(picture)
        }
    }
    return
},
///////////////////////////////////////////////////////////////////////////////////
"journal": func(cfg *Configuration, w http.ResponseWriter, r *http.Request) (err error) {
    var n int // for form value
    ths := []string{"#", "Время", "Источник", "Устройство", "Событие", "Пользователь", "Причины", "Принятые меры"}
    if "GET" != r.Method {
        httpStatus(w, http.StatusMethodNotAllowed)
        return
    }

    filter := EventFilter{}
    filter.Start, err = time.Parse(time.RFC3339, getStringVal(r.Form["start"]))
    if nil != err {return fmt.Errorf("%w: start", argumentError)}

    filter.End, err = time.Parse(time.RFC3339, getStringVal(r.Form["end"]))
    if nil != err {return fmt.Errorf("%w: end", argumentError)}

    n, err = getIntVal(r.Form["serviceId"])
    if nil != err {return fmt.Errorf("%w: serviceId", argumentError)}
    filter.ServiceId = int64(n)

    n, err = getIntVal(r.Form["userId"])
    if nil != err {return fmt.Errorf("%w: userId", argumentError)}
    filter.UserId = int64(n)

    n, err = getIntVal(r.Form["limit"])
    if nil != err {return fmt.Errorf("%w: limit", argumentError)}
    filter.Limit = int64(n)

    filter.Class, err = getIntVal(r.Form["class"])
    if nil != err {return fmt.Errorf("%w: class", argumentError)}

    html := `<html><table border="1" cellpadding="3" cellspacing="0"><tr>`
    for _, th := range ths {
        html += "<th>" + th + "</th>"
    }
    html += "</tr>\n"

    events, err := cfg.loadEvents(&filter)
    if nil != err {return}
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
    return
},}

func httpStatus(w http.ResponseWriter, code int) {
    http.Error(w, http.StatusText(code), code)
}

func getIntVal(list []string) (n int, err error) {
    if nil != list && len(list) > 0 {
        n, err = strconv.Atoi(list[0])
    }
    return
}

func getStringVal(list []string) (s string) {
    if nil != list && len(list) > 0 {
        s = list[0]
    }
    return
}