package securos

import (
	"io/ioutil"
    //"bytes"
    "net/http"
	"net"
	"errors"
    "time"
    "strings"
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

func getAuthRequest(url, login, password string) *[]byte {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
	   	panic(err)
	}
    req.SetBasicAuth(login, password)

    client := httpClient()
    resp, err := client.Do(req)
    catch(err)
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    catch(err)
    if http.StatusOK != resp.StatusCode {
    	panic(errors.New("GET " + url + " => " + resp.Status))
    }
    
    return &body
}

// read local file (for debug)
func postRequest2(url string, payload string) (*[]byte, error) {
	parts := strings.Split(url, "/")
	filename := parts[len(parts) - 1]
	xml, err := ioutil.ReadFile("xml/" + filename + ".xml")
	return &xml, err
}

/*func postRequest(url string, payload string) (b1 *[]byte, eRet error) {
	defer func () {
		if r := recover(); r != nil {
			eRet = r.(error)
		}
	}()

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(payload)))
	if err != nil {
	   	panic(err)
	}
	req.Header.Set("Content-type", "application/soap+xml")

    //camClient.Timeout = 1000 * time.Millisecond
    camClient := httpClient()
    resp, err := camClient.Do(req)
    if err != nil {
        panic(NewPTZConnectionError(err.Error()))
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)

    //fmt.Print("\n========", url, " : ", resp.StatusCode, "========\n")
    //fmt.Println(payload)
    //fmt.Print("\n---------------------------------------\n")
    //fmt.Println(string(body))

    if http.StatusOK != resp.StatusCode {
    	panic(errors.New("POST " + url + " => " + resp.Status))
    }
    
    if err != nil {
    	//panic(err)
    	panic(NewPTZConnectionError(err.Error()))
    }
    
    return &body, nil
}*/