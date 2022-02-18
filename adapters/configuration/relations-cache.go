package configuration

import  "s7server/dblayer"

func (cache *RelationsCache) expandParents(userId, parentId int64) (users []int64, err error) {
    cache.Lock()
    defer cache.Unlock()

    if nil == cache.parents || nil == cache.children {
        err = cache.cache()
    }
    
    return append(cache.parents[parentId], userId, parentId), err
}

// reset cache if userId is a cached group
// force reset if userId == 0
func (cache *RelationsCache) checkReset(userId int64) {
    cache.Lock()
    defer cache.Unlock()
    if nil == cache.parents {
        return
    }
    if _, ok := cache.parents[userId]; ok || 0 == userId { // reset cache if userId is group
        cache.parents = nil
        cache.children = nil
    }
}
func (cache *RelationsCache) expandChildren(userId int64) (list []int64, err error) {
    list = []int64{userId}
    children, err := cache.childrenList(userId)
    if nil != err {
        return
    }
    cache.RLock()
    for _, id := range children {
        list = append(list, cache.children[id]...)
    }
    cache.RUnlock()
    list = append(list, children...)
    return
}

func (cache *RelationsCache) childrenList(parentId int64) (list []int64, err error) {
    var id int64
    fields := dblayer.Fields {"id": &id}

    rows, values, err := db.Table("users").
        Seek("archived = false AND parent_id = ?", parentId).
        Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()
    for rows.Next() && nil == err {
        err = rows.Scan(values...)
        if nil == err {
            list = append(list, id)
        }
    }
    if nil == err {
        err = rows.Err()
    }
    return
}

func (cache *RelationsCache) cache() (err error) {
    parents := make(map[int64] []int64)
    children := make(map[int64] []int64)
    var userId, parentId int64
    
    fields := dblayer.Fields{
        "id": &userId,
        "parent_id": &parentId}

    // TODO: children_id is always greater than parent_id, but until transfer between groups happens (or use timestamp for group change?)
    //
    cond := "parent_id > 0 AND type = 1 AND archived = false" // user root can't have linked devices etc.
    rows, values, err := db.Table("users").Order("id").Seek(cond).Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()

    for rows.Next() && nil == err {
        if err = rows.Scan(values...); nil == err {
            parents[userId] = append(parents[userId], parentId)
            parents[userId] = append(parents[userId], parents[parentId]...)
        }
    }
    if nil == err {
        err = rows.Err()
    }
    if nil != err {
        return
    }
    
    for userId = range parents {
        for _, parentId := range parents[userId] {
            children[parentId] = append(children[parentId], userId)
        }
    }
    
    cache.parents = parents
    cache.children = children

    return
}