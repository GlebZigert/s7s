package configuration

import (
    "io"
    "os"
    "fmt"
    "sort"
    "time"
    "errors"
    "regexp"
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
    var cleanup bool // cleanup on next tick after backup
    timer := time.NewTimer(1 * time.Minute)
    for nil == ctx.Err() {
        select {
            case <-ctx.Done():
                return

            case <-timer.C:
                // TODO: get backup period from settings
                // TODO: atomic db timeouts increase?
                if nil == cfg.backupDatabase(dbBackupInterval) {
                    cleanup = true
                } else if cleanup && nil == cfg.cleanupEvents(maxDBEvents) {
                    cfg.Log("Events list is shrinked to", maxDBEvents, "items")
                    cleanup = false
                }
        }
        timer.Reset(1 * time.Minute)
    }
}

func (cfg *Configuration) applyBackup() (err error) {
    dbFile, dbBak, dbList, err := cfg.listDatabases()
    if nil != err {return}
    
    var i int
    for i = range dbList {
        if 0 <= strings.Index(dbList[i], cfg.nextDatabase) {
            break
        }
    }
    if i >= len(dbList) {return}
    
    //cfg.Log("Backup current db:", dbFile, "=>", dbBak)
    //err = os.Rename(dbFile, dbBak)
    err = cfg.backupDatabase(0)
    if nil != err {
        cfg.Err("Backup failed")
        return
    }
    cfg.Log("Restore backup:", dbList[i], "=>", dbFile)
    err = copyFile(dbList[i], dbFile)
    if nil == err {return}
    
    cfg.Err("Restore failed, try to rollback")
    err = os.Rename(dbBak, dbFile)
    if nil != err {
        cfg.Err("Rollback failed")
    }
    return
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
    err = db.MakeTables(tables, true)
    if nil != err {
        db.Close()
        return
    }
    db.MakeTables(tableUpdates, false) // ignore errors
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

func (cfg *Configuration) backupDatabase(minInterval time.Duration) (err error) {
    cfg.backupLock.Lock()
    defer cfg.backupLock.Unlock()
    start := time.Now()
    
    defer func () {
        dur := time.Now().Sub(start)
        if nil == err {
            cfg.Log("Database backup completed in", dur)
            cfg.Broadcast("Events", api.EventsList{api.Event{Class: api.EC_DB_BACKED_UP}})
        } else if !errors.Is(err, tooFrequentBackups) {
            cfg.Err("Database backup failed in", dur, ":", err)
            cfg.Broadcast("Events", api.EventsList{api.Event{Class: api.EC_DB_BACKUP_FAILED}})
        }
    }()
    
    lastTime, err := cfg.lastBackupTime()
    if nil != err {return}

    if time.Now().Sub(lastTime) < minInterval {
        return tooFrequentBackups
    }
    
    dbFile, dbBak, dbList, err := cfg.listDatabases()
    if nil != err {return}
   
    // 2. backup database
    cmd := exec.Command("sqlite3", dbFile, `.backup "` + dbBak + `"`)
    // TODO: not supported on linux, use conditional build if needed
    //cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    err = cmd.Run()
    //out, err := cmd.CombinedOutput()
    if nil != err {return}

    // 3. clean old databases
    for i, fn := range dbList {
        if i >= dbMaxBackups - 1 {
            err = os.Remove(fn)
            if nil != err {return}
        }
    }
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
