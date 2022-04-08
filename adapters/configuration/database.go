package configuration

import (
    "io"
    "os"
    "fmt"
    "sort"
    "time"
    "regexp"
	"syscall"
	"os/exec"
    "context"
    "strings"
    "path/filepath"
    "database/sql"    
    sqlite3 "github.com/mattn/go-sqlite3"
)


import (
    "s7server/api"
    "s7server/dblayer"
)

const (
    dbMaxBackups = 10
    timestampLayout = "20060102150405"
)

func (cfg *Configuration) dbBackupSheduler(ctx context.Context) {
    defer cfg.Log("DB backup sheduler stopped")
    timer := time.NewTimer(1 * time.Minute)
    for nil == ctx.Err() {
        select {
            case <-ctx.Done():
                return

            case <-timer.C:
                lastTime, err := cfg.lastBackupTime()
                // TODO: get backup period from settings
                // TODO: atomic db timeouts increase?
                if nil == err && time.Now().Sub(lastTime) >= 12 * time.Hour {
                    start := time.Now()
                    err = cfg.backupDatabase()
                    if nil == err {
                        cfg.Log("Sheduled database backup completed in", time.Now().Sub(start))
                        cfg.Broadcast("Events", api.EventsList{api.Event{Class: api.EC_DB_BACKED_UP}})
                    }
                }
                if nil != err {
                    cfg.Err("Sheduled database backup failed")
                    cfg.Broadcast("Events", api.EventsList{api.Event{Class: api.EC_DB_BACKUP_FAILED}})
                }
        }
        timer.Reset(1 * time.Minute)
    }
}


func (cfg *Configuration) tryDatabase(fn string) (err error) {
    //TODO: https://github.com/mattn/go-sqlite3#user-authentication
    //var db interface{}
    database, err := sql.Open("sqlite3", fn + connParams)
    if nil != err {return}

    ctx, _ := context.WithTimeout(context.TODO(), 1 * time.Second)
    err = database.PingContext(ctx)
    
    if nil != err {
        database.Close()
        return
    }

    db.Bind(database, qTimeout)
    err = db.MakeTables(tables)
    if nil != err {
        db.Close()
        return
    }
    db.MakeTables(tableUpdates) // ignore errors
    return
}

func (cfg *Configuration) openDatabase(maxAttempts int) (err error) {
    dbFile, dbBak, dbList, err := cfg.listDatabases()
    if nil != err {return}

    err = cfg.tryDatabase(dbFile)
    if se, ok := err.(sqlite3.Error); !ok || sqlite3.ErrCorrupt != se.Code {
        return // unrecoverable error or no error (nil)
    }

    cfg.Log("Primary DB file is damaged. Found backups:", dbList)
    if 0 == len(dbList) {return}
    // backup "configuration-0.db"
    err = os.Rename(dbFile, dbBak)
    if nil != err {return}
    cfg.Log("Primary db saved:", dbFile, "=>", dbBak)
    
    for i, fn := range dbList {
        if i >= maxAttempts {break} // stop, enough
        cfg.Log("Trying backup", fn)
        
        // prepare current backup
        err = copyFile(fn, dbFile)
        if nil != err {break}

        err = cfg.tryDatabase(dbFile)
        if sqliteErr, ok := err.(sqlite3.Error); !ok || sqlite3.ErrCorrupt != sqliteErr.Code {
            break // unrecoverable error or no error
        }
    }
    if nil == err {
        // notify about backup usage
        cfg.Broadcast("Events", api.EventsList{api.Event{Class: api.EC_USE_DB_BACKUP}})
    } else {
        // recover original db
        if nil == os.Rename(dbBak, dbFile) {
            cfg.Log("Rollback primary db backup:", dbBak, "=>", dbFile)
        }
    }
    return
}

func (cfg *Configuration) backupDatabase() (err error) {
    dbFile, dbBak, dbList, err := cfg.listDatabases()
    if nil != err {return}
   
    for i, fn := range dbList {
        if i >= dbMaxBackups {
            err = os.Remove(fn)
            if nil != err {return}
        }
    }
    
    // 2. backup database
    cmd := exec.Command("sqlite3", dbFile, `.backup "` + dbBak + `"`)
    cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    err = cmd.Run()
    //out, err := cmd.CombinedOutput()
    return
}

// use time.IsZero() when no backups yet?
func (cfg *Configuration) lastBackupTime() (dt time.Time, err error) {
    _, _, dbList, err := cfg.listDatabases()
    if nil != err || 0 == len(dbList) {return}
    
    re := regexp.MustCompile(`-(\d{14,14})\.db$`)
    matches := re.FindStringSubmatch(dbList[0])
    //cfg.Log("MATCHES:", matches)
    if len(matches) < 2 {return}
    dt, err = time.ParseInLocation(timestampLayout, matches[1], time.Now().Location())

    return
}

// returns [dbFile, bakFile (not created yet), filelist...]
func (cfg *Configuration) listDatabases() (dbFile, dbBak string, dbList []string, err error) {
    dbFile = cfg.GetStorage() + ".db"
    dbBak = strings.Replace(dbFile, "-0.", "-" + time.Now().Format(timestampLayout) + ".", 1)
    pattern := strings.Replace(dbFile, "-0.", "-2*.", 1)
    dbList, err = filepath.Glob(pattern)
    if nil != err {
        err = fmt.Errorf("Can't list DB backups: %w", err)
        return
    }
    sort.Sort(sort.Reverse(sort.StringSlice(dbList)))
    return
}

func (cfg *Configuration) LoadLinks(sourceId int64, link string) (list []ExtLink, err error) {
    defer func () {cfg.complaints <- err}()
    //list := make([]ExtLink, 0)
    var id int64
    var scope int64
    var flags int64

    fields := dblayer.Fields {
        "scope_id": &scope,
        "target_id": &id,
        "flags": &flags}

    rows, values, err := db.Table("external_links").
        Seek("link = ? AND source_id = ?", link, sourceId).
        Get(nil, fields)
    if nil != err {
        return
    }
    defer rows.Close()

    for rows.Next() {
        err = rows.Scan(values...)
        if nil != err {
            break
        }
        list = append(list, ExtLink{scope, id, flags})
    }
    // TODO: clean list if err?
    return
}


func (cfg *Configuration) SaveLinks(sourceId int64, linkType string, list []ExtLink) (err error){
    defer func () {cfg.complaints <- err}()
    tx, err := db.Tx(qTimeout)
    if nil != err {
        return
    }
    defer func () {completeTx(tx, err)}()
    //defer func() {if nil != err {tx.Rollback()}}()
    
    table := db.Table("external_links")
    err = table.Delete(tx, "link = ? AND source_id = ?", linkType, sourceId)
    if nil != err {
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
    return
}

//////////////////////////////////////////////////////////////////////

func copyFile(src, dst string) (err error) {
    // TODO: check existance & !dir?
    fin, err := os.Open(src)
    if err != nil {return}
    defer fin.Close()

    fout, err := os.Create(dst)
    if err != nil {return}
    defer fout.Close()

    _, err = io.Copy(fout, fin)

    return
}
