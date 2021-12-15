package configuration

import (
    "context"
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

    rows, values := cfg.Table("external_links").
        Seek("link = ? AND source_id = ?", link, sourceId).
        Get(fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, ExtLink{scope, id, flags})
    }

    return
}


func (cfg *Configuration) SaveLinks(sourceId int64, linkType string, list []ExtLink) (err error){
    ctx := context.Background()
	tx, err := cfg.BeginTx(ctx)
    
    table := cfg.Table("external_links")
    err = table.Tx(tx).Delete("link = ? AND source_id = ?", linkType, sourceId)
    if nil == err {
        return
    }

    for _, link := range list {
        table.Insert(dblayer.Fields {
            "source_id": sourceId,
            "link": linkType,
            "scope_id": link[0],
            "target_id": link[1],
            "flags": link[2]})
    }
    return
}
