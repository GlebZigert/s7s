package securos

import (
    "sync"
    "../../api"
)

type Reply func (int, string, string, interface{})

type Securos struct {
    sync.RWMutex
    api.API
    
	devices map[int] Device
}

type Device struct {
	Id 		int 	  `json:"id"`
	Name	string    `json:"name"`
}

type State struct {
	Id 			int 	`json:"id"`
	DateTime 	uint 	`json:"datetime"`
	Name 		string 	`json:"name"`
}
