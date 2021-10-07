package cam

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
    "sync"
	"github.com/deepch/vdk/av"
)

var Config = loadConfig()

type ConfigST struct {
    sync.RWMutex
    Server  ServerST            `json:"server"`
	Streams map[string]StreamST `json:"streams"`
}

type ServerST struct {
	HTTPPort string `json:"http_port"`
}

type StreamST struct {
    URL    string `json:"url"`
	Status bool   `json:"status"`
	Codecs []av.CodecData
	Clients     map[string]viwer
}
type viwer struct {
	c chan av.Packet
}

func loadConfig() *ConfigST {
	var tmp ConfigST
	data, err := ioutil.ReadFile("adapters/cam/config.json")
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		log.Fatalln(err)
	}
	for i, v := range tmp.Streams {
		v.Clients = make(map[string]viwer)
		tmp.Streams[i] = v
	}
	return &tmp
}

func (element *ConfigST) cast(uuid string, pck av.Packet) {
    element.RLock()
    defer element.RUnlock()
	for _, v := range element.Streams[uuid].Clients {
		if len(v.c) < cap(v.c) {
			v.c <- pck
		}
	}
}

func (element *ConfigST) ext(suuid string) bool {
	_, ok := element.Streams[suuid]
	return ok
}

func (element *ConfigST) addCodecs(suuid string, codecs []av.CodecData) {
	t := element.Streams[suuid]
	t.Codecs = codecs
	element.Streams[suuid] = t
}

func (element *ConfigST) getCodecs(suuid string) []av.CodecData {
	return element.Streams[suuid].Codecs
}

func (element *ConfigST) addClient(suuid string) (string, chan av.Packet) {
	cuuid := pseudoUUID()
	ch := make(chan av.Packet, 100)
    element.Lock()
	element.Streams[suuid].Clients[cuuid] = viwer{c: ch}
    element.Unlock()
	return cuuid, ch
}

func (element *ConfigST) list() (string, []string) {
	var res []string
	var fist string
	for k := range element.Streams {
		if fist == "" {
			fist = k
		}
		res = append(res, k)
	}
	return fist, res
}

func (element *ConfigST) delClient(suuid, cuuid string) {
    element.Lock()
	delete(element.Streams[suuid].Clients, cuuid)
    element.Unlock()
}

func pseudoUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}
