package configuration

import  "s7server/dblayer"

func (cache *RelationsCache) expandParents(userId, parentId int64) (users []int64, err error) {
    cache.Lock()
    defer cache.Unlock()

    err = cache.prepareCache()
    if nil != err {return}
    
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
    if nil != err {return}

    cache.Lock()
    defer cache.Unlock()

    err = cache.prepareCache()
    if nil != err {return}

    for _, id := range children {
        list = append(list, cache.children[id]...)
    }

    list = append(list, children...)
    return
}

func (cache *RelationsCache) childrenList(parentId int64) (list []int64, err error) {
    var id int64
    fields := dblayer.Fields {"id": &id}

    err = db.Table("users").
        Seek("archived = 0 AND parent_id = ?", parentId).
        Rows(nil, fields).
        Each(func() {
            list = append(list, id)
        })
    return
}

func (cache *RelationsCache) prepareCache() (err error) {
    if nil != cache.parents && nil != cache.children {
        return
    }

    var userId, parentId int64
    parents := make(map[int64] []int64)
    children := make(map[int64] []int64)
    
    fields := dblayer.Fields{
        "id": &userId,
        "parent_id": &parentId}

    // TODO: children_id is always greater than parent_id, but until transfer between groups happens (or use timestamp for group change?)
    //
    cond := "parent_id > 0 AND type = 1 AND archived = 0" // user root can't have linked devices etc.
    err = db.Table("users").Order("id").Seek(cond).Rows(nil, fields).Each(func() {
        parents[userId] = append(parents[userId], parentId)
        parents[userId] = append(parents[userId], parents[parentId]...)
    })

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