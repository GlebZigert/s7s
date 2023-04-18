package configuration

import (
    "s7server/dblayer"
)

func (cfg *Configuration) dbUpdateSettings(name, value string) (err error) {
    tx, err := db.Tx(qTimeout)
    if nil != err {return}
    defer func () {completeTx(tx, err)}()

    table := db.Table("settings")
    fields := dblayer.Fields{"value": value}
    nRows, err := table.Seek("name = ?", name).Update(tx, fields)
    if nil != err {return}
    if nRows > 0 {return} // something was updated
    
    fields["name"] = name
    _, err = table.Insert(tx, fields)
    return
}

func (cfg *Configuration) dbListSettings() (list []SettingsPair, err error){
    s := new(SettingsPair)
    fields := dblayer.Fields{"name": &s.Name, "value": &s.Value}

    err = db.Table("settings").Rows(nil, fields).Each(func() {
        list = append(list, *s)
    })
    return
}