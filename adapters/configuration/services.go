package configuration

import (
    "time"
    "s7server/api"
    "s7server/dblayer"
)

/*const (
    DB_LOGIN = "sqltdb1"
    DB_PASSWORD = "sqltdb2"
)*/


/////////////////////////////////////////////////////////////////////
////////////////////////// S E R V I C E S //////////////////////////
/////////////////////////////////////////////////////////////////////

/*func (cfg *Configuration) getServiceName(serviceId int64) (name string) {
    fields := dblayer.Fields{"title": &name}

    rows, values, _ := db.Table("services").
        Seek(serviceId).
        Get(nil, fields, 1)
    
    if rows.Next() {
        _ = rows.Scan(values...)
    }
    rows.Close()
    return
}*/

// TODO: handle encryption error
func (cfg *Configuration) newService(s *api.Settings) (err error) {
    // TODO: cipher password field!
    fld := dblayer.Fields {
            "type": s.Type,
            "title": s.Title,
            "host": s.Host,
            "login": s.Login,
            "keep_alive": s.KeepAlive,
            "db_host": s.DBHost,
            "db_name": s.DBName,
            "db_login": s.DBLogin,
    }
    cipherPasswords(s, fld)
    s.Id, err = db.Table("services").Insert(nil, fld)
    return nil
}

func (cfg *Configuration) updService(s api.Settings) (err error) {
    // type field is absent due it can't be changed
    fld := dblayer.Fields {
            "title": s.Title,
            "host": s.Host,
            "login": s.Login,
            //"password": &password,
            "keep_alive": s.KeepAlive,
            "db_host": s.DBHost,
            "db_name": s.DBName,
            "db_login": s.DBLogin,
            /*"db_password": &s.dbPassword*/}
    
    cipherPasswords(&s, fld)
    _, err = db.Table("services").Seek(s.Id).Update(nil, fld)
    return
}

func (cfg *Configuration) dbDeleteService(id int64) (err error) {
    //db.Table("services").Delete(nil, id)
    timestamp := time.Now().Unix()
    _, err = db.Table("services").Seek(id).Update(nil, dblayer.Fields{"archived": timestamp})
    return
}

func (cfg *Configuration) loadServices() (list []*api.Settings, err error) {
    s := new(api.Settings)
    fields := dblayer.Fields {
        "id":           &s.Id,
        "type":         &s.Type,
        "title":        &s.Title,
        "host":         &s.Host,
        "login":        &s.Login,
        "password":     &s.Password,
        "keep_alive":   &s.KeepAlive,
        "db_host":      &s.DBHost,
        "db_name":      &s.DBName,
        "db_login":     &s.DBLogin,
        "db_password":  &s.DBPassword}

    err = db.Table("services").Seek("archived IS NULL").Order("id").
        Rows(nil, fields).Each(func () {
            tmp := *s        
            list = append(list, &tmp)
        })
    if nil == err {
        for i := range list {
            if len(s.Password) > 0 {
                list[i].Password, err = decrypt(list[i].Password)
            }
            if nil != err {break}
            
            if len(s.DBPassword) > 0 {
                list[i].DBPassword, err = decrypt(list[i].DBPassword)
            }
            if nil != err {break}
        }
    }
    return
}


func cipherPasswords(s *api.Settings, fld dblayer.Fields) (err error) {
    // TODO: remove next 2 lines after upplying new db schema
    fld["password"] = ""
    fld["db_password"] = ""

    if "" != s.Password {
        fld["password"], err = encrypt(s.Password)
    }
    
    if nil == err && "" != s.DBPassword {
        fld["db_password"], err = encrypt(s.DBPassword)
    }

    return
}
