package sigur

import (
    "log"
//    "strconv"
//    "strings"
    "database/sql"
//    "fmt"
    _ "../../mysql"
    "../../dblayer"
)
/*
var lastLog
func (sigur *Sigur) logOnce(s string) {
    if lastLog != s {
        lastLog = s
        log.Println(name, s)
    }
}*/

func (sigur *Sigur) initDB() {
    log.Println("Check evTypes[10]:", evTypes[10])
    //var err error
    // connection string
    cstr := sigur.Settings.DBLogin + ":" + sigur.Settings.DBPassword + "@tcp(" + sigur.Settings.DBHost + ")/" + sigur.Settings.DBName
    //sigur.db, err = sql.Open("mysql", "root:start7@tcp(192.168.20.237:3306)/TC-DB-MAIN")
    //log.Println(sigur.GetName(), "opening db:", cstr)
    sigur.DB, _ = sql.Open("mysql", cstr)
    /*
    if err != nil {
        //log.Println(sigur.GetName(), err)
        return err
    }

    err = sigur.db.Ping()
    if err != nil {
        //log.Println(sigur.GetName(), err)
        return err
    } else {
        //log.Println(sigur.GetName(), "connected to database")
    }

    return nil
    */
}

func (sigur *Sigur) dbListPersonal() []*Personal{
    var list []*Personal
    /*rows, err := sigur.db.Query("SELECT id, parent_id, type, name, codekey FROM PERSONAL")
        list = append(list, p)
    } */   
    return list
}


func (sigur *Sigur) dbListDevices() map[int] Device {
    devices := make(map[int]Device)

    dev := new(Device)
    fields := dblayer.Fields {
        "id": &dev.Id,
        "parent_id": &dev.ParentId,
        "type": &dev.Type,
        "name": &dev.Name}
    
    rows, values := sigur.Table("DEVICES").Get(fields)
    defer rows.Close() // TODO: defer triggered for this rows?

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        devices[dev.Id] = *dev
    }

    return devices
}

func (sigur *Sigur) loadRules() []AccessRule {
    list := []AccessRule{}

    rule := new(AccessRule)
    fields := dblayer.Fields {
        "ID": &rule.Id,
        "RULETYPE": &rule.RuleType,
        "POWERIDX": &rule.PowerIdx,
        "NAME": &rule.Name}
    
    rows, values := sigur.Table("ACCESSRULES").Get(fields)
    defer rows.Close() // TODO: defer triggered for this rows?

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *rule)
    }

    return list
}


func catch(err error) {
    if nil != err {
        panic(err)
    }
}
