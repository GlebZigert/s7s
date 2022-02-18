package configuration

import (
//    "database/sql"
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

// create or update
/*
func (cfg *Configuration) saveRow(fields *dblayer.Fields, table string) {
    var pId *int64
    tmp, ok := (*fields)["id"]
    if ok {
        pId = tmp.(*int64)
    }
    if !ok { // new WITHOUT id
        db.Table(table).Insert(nil, *fields)
    } else if 0 == *pId { // new WITH id
        delete(*fields, "id")
        *pId = db.Table(table).Insert(nil, *fields)
    } else { // update
        delete(*fields, "id")
        db.Table(table).Seek(*pId).Update(nil, *fields)
    }

}*/

func (cfg *Configuration) loadMaps() (list MapList){
    myMap := new(Map)
    fields := mapFields(myMap)

    rows, values, _ := db.Table("maps").Get(nil, fields)
    defer rows.Close() // TODO: defer triggered for this rows?

    idMaps := make(map[int64]int)
    var ids []int64
    
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *myMap)
        ids = append(ids, myMap.Id)
        idMaps[myMap.Id] = len(list) - 1
    }
    
    if len(idMaps) == 0 {
        return // see err "index out of range [0] with length 0" in next loop bellow
    }
    
    //
    // get shapes
    //
    shape := new(Shape)
    fields = shapeFields(shape)
    rows, values, _ = db.Table("shapes").Seek("map_id", ids).Get(nil, fields)
    defer rows.Close()
    
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        // TODO: index out of range [0] with length 0
        list[idMaps[shape.MapId]].Shapes = append(list[idMaps[shape.MapId]].Shapes, *shape)
    }
    return
}

func (cfg *Configuration) dbDeleteMap(id int64) (err error) {
    err = db.Table("shapes").Delete(nil, "map_id", id)
    if nil == err {
        err = db.Table("maps").Delete(nil, id)
    }
    return
}

func (cfg *Configuration) dbUpdatePlanPicture(id int64, picture []byte) {
    db.Table("maps").Seek(id).Update(nil, dblayer.Fields {"picture": &picture})
}

func (cfg *Configuration) dbLoadPlanPicture(id int64) []byte {
    var picture []byte
    fields := dblayer.Fields {"picture": &picture}
    rows, values, _ := db.Table("maps").Seek(id).Get(nil, fields)
    defer rows.Close() // TODO: defer triggered for this rows?
    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
    }
    return picture
}

func (cfg *Configuration) dbUpdateMap(myMap *Map) {
    fields := mapFields(myMap)
    db.Table("maps").Save(nil, fields)
    if len(myMap.Shapes) > 0 {
        cfg.dbUpdateShapes(myMap.Id, myMap.Shapes)
    }
}

func (cfg *Configuration) dbUpdateShapes(mapId int64, shapes []Shape) (err error) {
    var ids []int64
    for i, _ := range shapes {
        shapes[i].MapId = mapId
        fields := shapeFields(&shapes[i])
        db.Table("shapes").Save(nil, fields)
        ids = append(ids, shapes[i].Id)
    }

    err = db.Table("shapes").Delete(nil, "map_id = ? AND id NOT", mapId, ids)
    return
}