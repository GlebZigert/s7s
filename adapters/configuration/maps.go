package configuration

import (
    "database/sql"
//    "s7server/api"
    "s7server/dblayer"
)

func mapFields(myMap *Map) dblayer.Fields {
    fields := dblayer.Fields {
        "id": &myMap.Id,
        "type": &myMap.Type,
        "name": &myMap.Name,
        "cx": &myMap.CX,
        "cy": &myMap.CY,
        "zoom": &myMap.Zoom}
    return fields
}

func shapeFields(shape *Shape) dblayer.Fields {
    fields := dblayer.Fields {
        "id": &shape.Id,
        "map_id": &shape.MapId,
        "service_id": &shape.ServiceId,
        "device_id": &shape.DeviceId,
        "type": &shape.Type,
        "x": &shape.X,
        "y": &shape.Y,
        "z": &shape.Z,
        "w": &shape.W,
        "h": &shape.H,
        "r": &shape.R,
        "data": &shape.Data}
    return fields
}

func (cfg *Configuration) loadMaps() (list MapList, err error){
    myMap := new(Map)
    fields := mapFields(myMap)
    idMaps := make(map[int64]int)
    var ids []int64

    err = db.Table("maps").Rows(nil, fields).Each(func() {
        list = append(list, *myMap)
        ids = append(ids, myMap.Id)
        idMaps[myMap.Id] = len(list) - 1
    })
    if nil != err {return}
    
    if len(idMaps) == 0 {
        return // see err "index out of range [0] with length 0" in next loop bellow
    }
    
    //
    // get shapes
    //
    shape := new(Shape)
    fields = shapeFields(shape)
    err = db.Table("shapes").Seek("map_id", ids).Rows(nil, fields).Each(func() {
        // TODO: index out of range [0] with length 0
        list[idMaps[shape.MapId]].Shapes = append(list[idMaps[shape.MapId]].Shapes, *shape)
    })
    return
}

func (cfg *Configuration) dbDeleteMap(id int64) (err error) {
    tx, err := db.Tx(qTimeout)
    if nil != err {return}
    defer func () {completeTx(tx, err)}()
    
    err = db.Table("shapes").Delete(tx, "map_id", id)
    if nil == err {
        err = db.Table("maps").Delete(tx, id)
    }
    return
}

func (cfg *Configuration) dbUpdatePlanPicture(id int64, picture []byte) (err error) {
    _, err = db.Table("maps").Seek(id).Update(nil, dblayer.Fields {"picture": &picture})
    return
}

func (cfg *Configuration) dbLoadPlanPicture(id int64) (picture []byte, err error) {
    fields := dblayer.Fields {"picture": &picture}
    err = db.Table("maps").Seek(id).First(nil, fields)
    if sql.ErrNoRows == err {
        err = nil // it's not an error
    }
    return
}

func (cfg *Configuration) dbUpdateMap(myMap *Map) (err error) {
    tx, err := db.Tx(qTimeout)
    if nil != err {return}
    defer func () {completeTx(tx, err)}()

    fields := mapFields(myMap)
    err = db.Table("maps").Save(tx, fields)
    if 0 == len(myMap.Shapes) {
        return
    }

    // update shapes
    var ids []int64
    for i, _ := range myMap.Shapes {
        myMap.Shapes[i].MapId = myMap.Id
        fields := shapeFields(&myMap.Shapes[i])
        err = db.Table("shapes").Save(tx, fields)
        if nil != err {return}
        ids = append(ids, myMap.Shapes[i].Id)
    }

    err = db.Table("shapes").Delete(tx, "map_id = ? AND id NOT", myMap.Id, ids)
    return
}

