package dispatcher

import "context"

import "../api"

var eQueue chan []*api.Event // events
var rQueue chan []*api.Event // replies

func (dispatcher *Dispatcher) queueServer(ctx context.Context) {
    
    
}