package configuration

import (
//    "database/sql"
//    "../../api"
    "../../dblayer"
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
        cfg.Table(table).Insert(*fields)
    } else if 0 == *pId { // new WITH id
        delete(*fields, "id")
        *pId = cfg.Table(table).Insert(*fields)
    } else { // update
        delete(*fields, "id")
        cfg.Table(table).Seek(*pId).Update(*fields)
    }

}*/

func (cfg *Configuration) loadMaps() (list []Map){
    myMap := new(Map)
    fields := mapFields(myMap)

    rows, values := cfg.Table("maps").Get(fields)
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
    rows, values = cfg.Table("shapes").Seek("map_id", ids).Get(fields)
    defer rows.Close()
    
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        // TODO: index out of range [0] with length 0
        list[idMaps[shape.MapId]].Shapes = append(list[idMaps[shape.MapId]].Shapes, *shape)
    }
    return
}

func (cfg *Configuration) dbDeleteMap(id int64) {
    cfg.Table("maps").Delete(id)
    cfg.Table("shapes").Delete("map_id", id)
}

func (cfg *Configuration) dbUpdatePlanPicture(id int64, picture []byte) {
    cfg.Table("maps").Seek(id).Update(dblayer.Fields {"picture": &picture})
}

func (cfg *Configuration) dbLoadPlanPicture(id int64) []byte {
    var picture []byte
    fields := dblayer.Fields {"picture": &picture}
    rows, values := cfg.Table("maps").Seek(id).Get(fields)
    defer rows.Close() // TODO: defer triggered for this rows?
    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
    }
    return picture
}

func (cfg *Configuration) dbUpdateMap(myMap *Map) {
    fields := mapFields(myMap)
    cfg.Table("maps").Save(fields)
    if len(myMap.Shapes) > 0 {
        cfg.dbUpdateShapes(myMap.Id, myMap.Shapes)
    }
}

func (cfg *Configuration) dbUpdateShapes(mapId int64, shapes []Shape) {
    var ids []int64
    for i, _ := range shapes {
        shapes[i].MapId = mapId
        fields := shapeFields(&shapes[i])
        cfg.Table("shapes").Save(fields)
        ids = append(ids, shapes[i].Id)
    }
    
    cfg.Table("shapes").Delete("map_id = ? AND id NOT", mapId, ids)
}