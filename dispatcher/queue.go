package dispatcher

import (
    "log"
    "time"
    "errors"
    "context"
    "math/rand"
    "encoding/json"
    "golang.org/x/net/websocket"
)

import (
    "../api"
    "../adapters/configuration"
)

const sendRetryInterval = 3 // re-reply interval, seconds

var outbox chan api.ReplyMessage // everybody have its own copy

func (dispatcher *Dispatcher) queueServer(ctx context.Context) {
    var queue []*api.ReplyMessage
    outbox = make(chan api.ReplyMessage, 10) // replies
    timer := time.NewTimer(sendRetryInterval * time.Second)
    rand.Seed(time.Now().UnixNano())
    
    for nil == ctx.Err() {
        select {
            case <-ctx.Done():
                break

            case msg := <-outbox:
                timer.Stop()
                queue = append(queue, &msg)
            
            case <-timer.C:
                //n, err = dispatcher.scanQueue(queue)
        }
        for len(outbox) > 0 {
            msg := <-outbox
            queue = append(queue, &msg)
        }
        n, err := dispatcher.scanQueue(queue)
        
        if nil == err && n > 0 {
            log.Println("Queue processed:", n)
        }
        if nil != err {
            log.Println("Queue failure:", n, "of", len(queue), "processed")
        }
        if n > 0 { // TODO: cut slice really
            queue = queue[n:]
        }
        // TODO: handle queue overflow
        
        timer.Reset(sendRetryInterval * time.Second)
    }
    log.Println("Reply queue scan stopped,", len(queue), "messages dropped")
}

// returns count of processed elements
func (dispatcher *Dispatcher) scanQueue(queue []*api.ReplyMessage) (n int, err error) {
    if 0 == len(queue) {
        return
    }
    
    for ; n < len(queue); n++ {
        err = dispatcher.processReply(queue[n])
        if nil != err {
            break
        }
    }
    return // num of processed messages
}

func (dispatcher *Dispatcher) processReply(reply *api.ReplyMessage) (err error) {
    cid := reply.UserId
    log.Println("Reply to", cid, ">", reply.Service, reply.Action, "task", reply.Task)
    //reply := ReplyMessage{Service: service, Action: action, Task: 0, Data: data}
    //log.Println(header)

    if 0 == 1 /*rand.Intn(3)*/ { // emulate failure
        return errors.New("WHOOPS!")
    }
    
    dispatcher.RLock()
    client, ok := dispatcher.clients[cid]
    dispatcher.RUnlock()
    if !ok {
        return // discard message, client disconnected
    }

    var filter map[int64]int64
    defer func (d interface{}) {reply.Data = d}(reply.Data) // save & restore original data
    
    if events, ok := reply.Data.(api.EventsList); ok && len(events) > 0 {
        if 0 == events[0].Id { // not yet processed
            err = dispatcher.processEvents(reply.Service, events)
        }
        if nil != err {
            return
        }
        // filter by devices permissions
        log.Println("::: APPLY EV FILTER :::", len(events), " events for svc #", reply.Service)
        idList := events.GetList()
        filter, err = dispatcher.cfg.Authorize(cid, idList)
        if nil == err {
            reply.Data = events.Filter(cid, filter, api.ARMFilter[client.role])
        }
    } else if original, ok := reply.Data.(configuration.Filterable); ok {
        // filter by devices permissions
        log.Println("::: APPLY DEV FILTER :::", reply.Service, reply.Action)
        // INFO: perform filtering inside services to handle special conditions such as groups (virtual elements)
        idList := original.GetList()
        filter, err = dispatcher.cfg.Authorize(cid, idList)
        if nil == err {
            reply.Data = original.Filter(filter)
        }
    } else {
        log.Println("::: FILTER FAILED :::", reply.Service, reply.Action)
    }
    
    if nil == err && nil != reply.Data {
        res, _ := json.Marshal(reply)
        // TODO: handle send error
        // TODO: implement retry limit for send
        _ = websocket.Message.Send(client.ws, string(res))
    }
        
    return // TODO: report db/reply failure to client
}
