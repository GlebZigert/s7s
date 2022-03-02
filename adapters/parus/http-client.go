package parus

import (
    "io"
	"io/ioutil"
    "net/http"
	"net"
	"errors"
    "time"
//    "fmt"
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

func getRequest(url string) (res []byte, eRet error) {
	defer func () {
		if r := recover(); r != nil {
			eRet = r.(error)
		}
	}()
	//camClient.Timeout = 500 * time.Millisecond
    client := httpClient()
    resp, err := client.Get(url)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    
    //fmt.Print("\n======== GET", url, " : ", resp.StatusCode, " ========\n")

    //fmt.Print("\n---------------------------------------\n")
    //fmt.Println(string(body))
	
    if http.StatusOK != resp.StatusCode {
    	panic(errors.New("GET " + url + " => " + resp.Status))
    }

    if err != nil && err != io.ErrUnexpectedEOF {
    	panic(err)
    }
    return body, nil
}

