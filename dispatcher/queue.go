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

const (
    maxQueueSize = 10000
    sendRetryInterval = 3 // re-reply interval, seconds
)

var outbox chan api.ReplyMessage // everybody have its own copy

func (dispatcher *Dispatcher) queueServer(ctx context.Context) {
    var queue []*api.ReplyMessage // unlimited length queue
    outbox = make(chan api.ReplyMessage, maxQueueSize / 10) // buffered replies
    timer := time.NewTimer(sendRetryInterval * time.Second)
    rand.Seed(time.Now().UnixNano()) // for failure emulation
    
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
        for len(outbox) > 0 { // scan all messages
            msg := <-outbox
            queue = append(queue, &msg)
        }
        if len(queue) > maxQueueSize {
            log.Println("Queue overflow:", len(queue), "of", maxQueueSize, "- trim old messages")
            queue = queue[(maxQueueSize - len(queue)):]
        }
        n, err := dispatcher.scanQueue(queue)
        
        if nil == err && n > 0 {
            log.Println("Queue: all", n, "items processed")
        }
        if nil != err {
            log.Println("Queue: only", n, "of", len(queue), "processed")
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
    // scan for events
    for n = range queue {
        events, _ := queue[n].Data.(api.EventsList)
        if len(events) > 0 && 0 == events[0].Id {
            // not yet processed/stored events
            err = dispatcher.processEvents(queue[n].Service, events)
            if nil == err {
                dispatcher.scanAlgorithms(events)
            }
        }
        if nil != err {
            log.Println("Events queue scan aborted:", err)
            break // stop further events processing to avoid order shuffle
        }
    }
    
    // then try to send processed events and all other
    for n = 0; n < len(queue); n++ {
        err = dispatcher.processReply(queue[n])
        if nil != err {
            // something went wrong, possible DB failure
            // TODO: report db/reply failure to ARM-client?
            break
        }
    }
    
    return // num of processed messages
}

func (dispatcher *Dispatcher) processEvents(serviceId int64, events api.EventsList) error {
    // events may not contain serviceId or service name, so fill them
    dispatcher.RLock()
    title := dispatcher.services[serviceId].GetSettings().Title
    dispatcher.RUnlock()
    for j := range events {
        events[j].ServiceId = serviceId
        events[j].ServiceName = title
    }

    // forward to core
    return dispatcher.cfg.ProcessEvents(events)
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
        return // abort sending, client disconnected or no clients at all (cid == 0)
    }

    var filter map[int64]int64
    defer func (d interface{}) {reply.Data = d}(reply.Data) // save & restore original data
    
    if events, _ := reply.Data.(api.EventsList); len(events) > 0 && 0 != events[0].Id {
        // send processed events & filter list by devices permissions
        log.Println("::: APPLY EV FILTER :::", len(events), " events for svc #", reply.Service)
        idList := events.GetList()
        filter, err = dispatcher.cfg.Authorize(cid, idList)
        if nil == err {
            reply.Data = events.Filter(cid, filter, api.ARMFilter[client.role])
        }
    }
    if original, ok := reply.Data.(configuration.Filterable); ok {
        // filter by devices permissions
        log.Println("::: APPLY DEV FILTER :::", reply.Service, reply.Action)
        // INFO: filtering performed inside services to handle special conditions such as groups (virtual elements)
        idList := original.GetList()
        filter, err = dispatcher.cfg.Authorize(cid, idList)
        if nil == err {
            reply.Data = original.Filter(filter)
        }
    }
    
    if nil == err && nil != reply.Data {
        res, _ := json.Marshal(reply)
        // TODO: handle send error? implement retry limit for send? if send failed, client has gone?
        _ = websocket.Message.Send(client.ws, string(res))
    }
        
    return
}

// check for automatic algorihms, special events, and so
func (dispatcher *Dispatcher) scanAlgorithms(events api.EventsList) {
    var aEvents api.EventsList
    for j := range events {
        algos := events[j].Algorithms
        for i := range algos {
            //log.Println("!!! ALGO:", algos[i])
            aEvents = append(aEvents, api.Event{Class: api.EC_ALGO_STARTED, Text: "Запуск алгоритма " + algos[i].Name})
        }
        
        if len(aEvents) > 0 {
            reply := api.ReplyMessage{Service: 0, Action: "Events", Data: aEvents}
            //log.Println("ALGO EVENTS:", reply)
            dispatcher.broadcast(0, &reply)
        }

        for i := range algos {
            if algos[i].TargetZoneId > 0 {
                dispatcher.doZoneCommand(0, algos[i].TargetZoneId, algos[i].Command)
            } else {
                cmd := api.Command{
                    algos[i].TargetDeviceId,
                    algos[i].Command,
                    algos[i].Argument}
                res, _ := json.Marshal(&cmd)
                /*if err != nil {
                    panic(err)
                }*/

                q := Query{algos[i].TargetServiceId, "ExecCommand", 0, res}
                //log.Println("!!! QUERY:", q)
                //log.Println("!!! CMD:", cmd)
                dispatcher.do(0, &q)
            }
        }
    }
}
