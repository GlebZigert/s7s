package parus

import (
    "io"
	"net"
    "fmt"
    "time"
    "errors"
    "net/http"
	"io/ioutil"
)

var (
    httpError = errors.New("HTTP status error")
    connError = errors.New("Connection problem")
)

func httpClient() *http.Client{
    return &http.Client{
        Timeout: 1000 * time.Millisecond,
        Transport: &http.Transport{
                DialContext: (&net.Dialer{
                        Timeout:   1000 * time.Millisecond,
                        KeepAlive: 0 * time.Second}).DialContext,
                DisableKeepAlives: true}}
}

func getRequest(url, login, password string) (body []byte, err error) {
    client := httpClient()
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        // TODO: maybe report a general error, not related to the conn or api?
        return
    } 
    if "" != login && "" != password {
        req.SetBasicAuth(login, password)
    }
    resp, err := client.Do(req)
    if err != nil {
        err = fmt.Errorf("%w: %v", connError, err)
        return
    }
    defer resp.Body.Close()

    body, err = ioutil.ReadAll(resp.Body)
	
    if io.ErrUnexpectedEOF == err {
        err = nil // WTF: is this some kind of bug?
    }
    
    if nil != err {
    	err = fmt.Errorf("%w: %v", connError, err)
    } else if http.StatusOK != resp.StatusCode {
        err = fmt.Errorf("%w: %v", httpError, resp.StatusCode)
    }
    return
}
