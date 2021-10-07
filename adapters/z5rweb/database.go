package z5rweb

import (
//    "log"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
//    "../../api"
//    "../../dblayer"
)

const (
    DB_LOGIN = "sqltdb1"
    DB_PASSWORD = "sqltdb2"
)

func (svc *Z5RWeb) openDB(fn string) {
    //TODO: https://github.com/mattn/go-sqlite3#user-authentication
    //var db interface{}
    var err error
    svc.DB, err = sql.Open("sqlite3", fn)
    catch(err)
    svc.MakeTables(tables, false)
}

func (svc *Z5RWeb) loadCards() (list []Card) {
    card := new(Card)
    fields := cardFields(card)

    rows, values := svc.Table("cards").Get(fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *card)
    }
    return
}

func (svc *Z5RWeb) loadTimezones() (list []Timezone) {
    tz := new(Timezone)
    fields := timezoneFields(tz)

    rows, values := svc.Table("timezones").Get(fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *tz)
    }
    return
}

/*func (svc *Z5RWeb) loadDevice(dType string, sn int64) *Device {
    device := new(Device)
    fields := deviceFields(device)

    rows, values := svc.Table("devices").Seek("type = ? AND serial_number = ?", dType, sn).Get(*fields)
    defer rows.Close()
    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        return device
    } else {
        return nil
    }
}*/
/*
func (svc *Z5RWeb) dbDeleteDevice(id int64) {
    // TODO: don't delete, archive?
    svc.Table("devices").Delete(id)
}

func (svc *Z5RWeb) dbUpdateDevice(device *Device) {
    fields := dblayer.Fields {
        "name": device.Name,
        "external_zone": device.ExternalZone,
        "internal_zone": device.InternalZone}
    svc.Table("devices").Seek(device.Id).Update(fields) // don't create, just update
}
*/