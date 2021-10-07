package configuration

import (
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


/*
func (cfg *Configuration) clearLinksSource(sourceId int64, linkType string) {
    table := cfg.Table("external_links")
    table.Delete("link = ? AND source_id = ?", linkType, sourceId)
}

func (cfg *Configuration) clearLinksTarget(targetId int64, linkType string) {
    table := cfg.Table("external_links")
    table.Delete("link = ? AND target_id = ?", linkType, sourceId)
}
*/

func (cfg *Configuration) SaveLinks(sourceId int64, linkType string, list []ExtLink) {
    table := cfg.Table("external_links")
    table.Delete("link = ? AND source_id = ?", linkType, sourceId)

    for _, link := range list {
        table.Insert(dblayer.Fields {
            "source_id": sourceId,
            "link": linkType,
            "scope_id": link[0],
            "target_id": link[1],
            "flags": link[2]})
    }
}
