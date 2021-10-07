package configuration

import (
    "time"
    "../../api"
    "../../dblayer"
)

/*const (
    DB_LOGIN = "sqltdb1"
    DB_PASSWORD = "sqltdb2"
)*/


/////////////////////////////////////////////////////////////////////
////////////////////////// S E R V I C E S //////////////////////////
/////////////////////////////////////////////////////////////////////

func (cfg *Configuration) getServiceName(serviceId int64) (name string) {
    fields := dblayer.Fields{"title": &name}

    rows, values := cfg.Table("services").
        Seek(serviceId).
        Get(fields, 1)
    
    if rows.Next() {
        _ = rows.Scan(values...)
    }
    rows.Close()
    return
}


func (cfg *Configuration) newService(s *api.Settings) {
    // TODO: cipher password field!
    var password, dbPassword string
    if "" != s.Password {
        password = encrypt(s.Password)
    }
    if "" != s.DBPassword {
        dbPassword = encrypt(s.DBPassword)
    }
    fld := dblayer.Fields {
            "type": &s.Type,
            "title": &s.Title,
            "host": &s.Host,
            "login": &s.Login,
            "password": &password,
            "keep_alive": &s.KeepAlive,
            "db_host": &s.DBHost,
            "db_name": &s.DBName,
            "db_login": &s.DBLogin,
            "db_password": dbPassword}
    
    s.Id = cfg.Table("services").Insert(fld)
}

func (cfg *Configuration) updService(s api.Settings) {
    // type field is absent due it can't be changed
    var password, dbPassword string
    fld := dblayer.Fields {
            "title": &s.Title,
            "host": &s.Host,
            "login": &s.Login,
            //"password": &password,
            "keep_alive": &s.KeepAlive,
            "db_host": &s.DBHost,
            "db_name": &s.DBName,
            "db_login": &s.DBLogin,
            /*"db_password": &s.dbPassword*/}
    
    if "" != s.Password {
        password = encrypt(s.Password)
        fld["password"] = &password
    }
    if "" != s.DBPassword {
        dbPassword = encrypt(s.DBPassword)
        fld["db_password"] = &dbPassword
    }
    cfg.Table("services").Seek(s.Id).Update(fld)
    //cfg.UpdateRows("services", fld, s.Id)
}

func (cfg *Configuration) dbDeleteService(id int64) {
    //cfg.Table("services").Delete(id)
    timestamp := time.Now().Unix()
    cfg.Table("services").Seek(id).Update(dblayer.Fields{"archived": timestamp})
}

func (cfg *Configuration) loadServices() (list []*api.Settings) {
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

    rows, values := cfg.Table("services").Seek("archived IS NULL").Order("id").Get(fields)
    defer rows.Close()
    
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        if len(s.Password) > 0 {
            s.Password = decrypt(s.Password)
        }
        if len(s.DBPassword) > 0 {
            s.DBPassword = decrypt(s.DBPassword)
        }
        tmp := *s        
        list = append(list, &tmp)
    }
    return
}

