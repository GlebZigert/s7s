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

func getRequest(url string) (body []byte, err error) {
    client := httpClient()
    resp, err := client.Get(url)
    if err != nil {
        err = fmt.Errorf("%w: %v", connError, err)
        return
    }
    defer resp.Body.Close()

    body, err = ioutil.ReadAll(resp.Body)
	
    if http.StatusOK != resp.StatusCode {
        err = fmt.Errorf("%w: %v", httpError, resp.StatusCode)
    } else if err == io.ErrUnexpectedEOF {
    	err = fmt.Errorf("%w: %v", connError, err)
    }
    return
}

