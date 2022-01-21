package configuration

import (
    //"context"
    "../../dblayer"
)

func (cfg *Configuration) LoadLinks(sourceId int64, link string) (list []ExtLink) {
    //list := make([]ExtLink, 0)
    var id int64
    var scope int64
    var flags int64

    fields := dblayer.Fields {
        "scope_id": &scope,
        "target_id": &id,
        "flags": &flags}

    rows, values, _ := db.Table("external_links").
        Seek("link = ? AND source_id = ?", link, sourceId).
        Get(nil, fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, ExtLink{scope, id, flags})
    }

    return
}


func (cfg *Configuration) SaveLinks(sourceId int64, linkType string, list []ExtLink) (err error){
    tx, err := db.Tx(qTimeout)
    if nil != err {
        return
    }
    //defer func() {if nil != err {tx.Rollback()}}()
    
    table := db.Table("external_links")
    err = table.Delete(tx, "link = ? AND source_id = ?", linkType, sourceId)
    if nil != err {
        tx.Rollback()
        return
    }

    for _, link := range list {
        _, err = table.Insert(tx, dblayer.Fields {
            "source_id": sourceId,
            "link": linkType,
            "scope_id": link[0],
            "target_id": link[1],
            "flags": link[2]})
        if nil != err {
            break
        }
    }
    if nil != err {
        tx.Rollback()
    } else {
        tx.Commit()
    }
    return
}
