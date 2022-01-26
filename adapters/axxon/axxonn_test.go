package axxon

import (
	"testing"
    "log"
)

func TestMain(m *testing.M) {
    res := getAuthRequest("http://192.168.0.231:8888/api/v1/cameras", "user", "start7")
    log.Println(string(*res))
}
