package dispatcher

import (
    "log"
    "time"
    "context"
    "encoding/json"
    "golang.org/x/net/websocket"
)

import (
    "../api"
    "../adapters/configuration"
)

const sendRetryInterval = 3 // re-reply interval, seconds

var inbox chan *api.ReplyMessage

func (dispatcher *Dispatcher) queueServer(ctx context.Context) {
    var queue []*api.ReplyMessage
    inbox = make(chan *api.ReplyMessage, 10) // replies
    timer := time.NewTimer(sendRetryInterval * time.Second)
    for nil == ctx.Err() {
        var n int
        select {
            case <-ctx.Done():
                break

            case msg := <-inbox:
                timer.Stop()
                queue = append(queue, msg)
                n = dispatcher.scanQueue(queue)
            
            case <-timer.C:
                n = dispatcher.scanQueue(queue)

        }
        if n > 0 { // TODO: cut slice really
            queue = []*api.ReplyMessage{}
        }
        timer.Reset(sendRetryInterval * time.Second)
    }
    log.Println("Reply queue scan stopped")
}

func (dispatcher *Dispatcher) scanQueue(queue []*api.ReplyMessage) int {
    log.Println("SCANQ")
    if 0 == len(queue) {
        return 0
    }
    
    for i := range queue {
        dispatcher.processReply(queue[i])
    }
    return len(queue)
}

func (dispatcher *Dispatcher) processReply(reply *api.ReplyMessage) {
    cid := reply.UserId
    log.Println("Reply to", cid, ">", reply.Service, reply.Action, "task", reply.Task)
    //reply := ReplyMessage{Service: service, Action: action, Task: 0, Data: data}
    //log.Println(header)

    dispatcher.RLock()
    client, ok := dispatcher.clients[cid]
    dispatcher.RUnlock()
    if !ok {
        return
    }

    data := reply.Data // store original data
    if events, ok := reply.Data.(api.EventsList); ok {
        // filter by devices permissions
        log.Println("::: APPLY EV FILTER :::", len(events), " events for svc #", reply.Service)
        idList := events.GetList()
        devFilter := dispatcher.cfg.Authorize(cid, idList)
        reply.Data = events.Filter(cid, devFilter, api.ARMFilter[client.role])
    } else if original, ok := reply.Data.(configuration.Filterable); ok {
        // filter by devices permissions
        log.Println("::: APPLY DEV FILTER :::", reply.Service, reply.Action)
        // INFO: perform filtering inside services to handle special conditions such as groups (virtual elements)
        idList := original.GetList()
        filter := dispatcher.cfg.Authorize(cid, idList)
        reply.Data = original.Filter(filter)
    } else {
        log.Println("::: FILTER FAILED :::", reply.Service, reply.Action)
    }
    
    if nil != reply.Data {
        res, err := json.Marshal(reply)
        if err != nil {
            panic(err)
        }
        websocket.Message.Send(client.ws, string(res))
    }
    reply.Data = data // restore data
}
