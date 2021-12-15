package dblayer

import (
    "log"
    "fmt"
    "errors"
    "strings"
    "strconv"
    "context"
    "database/sql"
    //_ "github.com/mattn/go-sqlite3"
)

// usage examples:
// .Table().Insert()
// .Table().Get(fields, limit)
// .Table().Update(fields)
// .Table().Delete(cond)
// .Table().Seek(cond).Get(fields, limit)
// .Table().Seek(cond).Update(fields)
// .Table().Seek(cond).Delete()


var LogTables []string

type Fields map[string] interface{}

type DBLayer struct {
    DB  *sql.DB
}

type QUD struct { // Query-Update-Delete
    table   string
    db      *sql.DB
    tx      *sql.Tx
    cond    string      // WHERE a = ? AND b = ?
    group   string
    order   string
    params  []interface{} // params for condition expression
}

func logQuery(table, q string, p interface{}) {
    if nil == LogTables {
        log.Println(q, p)
    } else {
        for i := range LogTables {
            if strings.Contains(table, LogTables[i]) /*LogTables[i] == table*/ {
                log.Println(q, p)
                break
            }
        }
    }
}

func (dbl *DBLayer) MakeTables(tables []string) (err error){
    for i := 0; i < len(tables) && nil == err; i++ {
        //log.Println(tables[i])
        _, err = dbl.DB.Exec(tables[i])
    }
    return
}

func (dbl *DBLayer) Table(t string) *QUD {
    // TODO: maybe db: &dbl.DB ?
    return &QUD{table: t, db: dbl.DB}
}

func (dbl *DBLayer) BeginTx(ctx context.Context) (*sql.Tx, error) {
    // TODO: maybe db: &dbl.DB ?
    return dbl.DB.BeginTx(ctx, nil)
}

func (qud *QUD) Tx(tx *sql.Tx) *QUD {
    qud.tx = tx
    return qud
}

func (qud *QUD) Order(o string) *QUD {
    qud.order = o
    return qud
}

func (qud *QUD) Group(g string) *QUD {
    qud.group = g
    return qud
}

// create or update
func (qud *QUD) Save(fields Fields) {
    var pId *int64
    tmp, ok := fields["id"]
    if ok {
        pId = tmp.(*int64)
    }
    if !ok { // new WITHOUT id
        qud.Insert(fields)
    } else if 0 == *pId { // new WITH id
        delete(fields, "id")
        *pId = qud.Insert(fields)
    } else { // update
        delete(fields, "id")
        qud.Seek(*pId).Update(fields)
    }
    //qud.reset()
}

func (qud *QUD) Insert(fields Fields) int64 {
    //keys, values := fieldsMap(fields)
    keys := ""
    values := make([]interface{}, len(fields))

    i := 0
    for k, v := range fields {
        if "" != keys {
            keys += "', '"
        }
        keys += k

        values[i] = v
        i++
    }
    
    q := "INSERT INTO " + qud.table + " ('" + keys + "') VALUES (?" + strings.Repeat(", ?", len(values)-1) + ")"
    logQuery(qud.table, q, values)
    
    res, err := qud.db.Exec(q, values...)
    catch(err, q, values)
    
    id, err := res.LastInsertId()
    catch(err, "", nil)

    return id
}

func (qud *QUD) Seek(args ...interface{}) *QUD {
    var where, list string
    var count int
    if len(args) == 0 {
        return qud
    }

    for pos, arg := range args {
        switch v := arg.(type) {
            case string:
                if 0 == pos {
                    count = strings.Count(v, "?")
                    where = v
                } else {
                    qud.params = append(qud.params, arg)
                }
            case int, int64:
                if 0 == pos || pos > count {
                    list = " = ?"
                }
                qud.params = append(qud.params, arg)

            case []int64: // TODO: use strings.Repeat()
                //list = "IN(" + JoinSlice(v) + ")"
                if len(v) > 0 {
                    list = "IN(?" + strings.Repeat(", ?", len(v)-1) + ")"
                    for i := range v {
                        qud.params = append(qud.params, v[i])
                    }
                }
            case []string: // TODO: use strings.Repeat()
                //list = "IN(" + JoinSlice(v) + ")"
                if len(v) > 0 {
                    list = "IN(?" + strings.Repeat(", ?", len(v)-1) + ")"
                    for i := range v {
                        qud.params = append(qud.params, v[i])
                    }
                }
            
            default:
                qud.params = append(qud.params, arg)
            }
	}
    if "" != list && "" == where {
        where = "id"
    }
    if "" != list || "" != where {
        qud.cond = " WHERE " + where + " " + list
    }

    return qud
}

func (qud *QUD) Get(mymap Fields, limits ...int64) (*sql.Rows, []interface{}) {
    return qud.RealGet("", mymap, limits...)
}

func (qud *QUD) GetDistinct(mymap Fields, limits ...int64) (*sql.Rows, []interface{}) {
    return qud.RealGet("DISTINCT", mymap, limits...)
}

func (qud *QUD) RealGet(hint string, mymap Fields, limits ...int64) (*sql.Rows, []interface{}) {
    keys := ""
    values := make([]interface{}, len(mymap))

    i := 0
    for k, v := range mymap {
        if "" != keys {
            keys += ", "
        }
        if strings.ContainsRune(k, '(') || strings.ContainsRune(k, ' '){
            keys += k
        } else if strings.ContainsRune(k, '.') {
            keys += strings.Replace(k, ".", ".`", 1) + "`"
        } else {
            keys += "`" + k + "`"
        }

        values[i] = v
        i++
    }
    q := "SELECT " + hint + " " + keys + " FROM " + qud.table + " " + qud.cond
    if "" != qud.order {
        q += " ORDER BY " + qud.order
    }
    if "" != qud.group {
        q += " GROUP BY " + qud.group
    }
    if len(limits) > 0 {
        q += " LIMIT ?"// + strconv.FormatInt(limits[0], 10)
        qud.params = append(qud.params, limits[0])
    }
    if len(limits) > 1 {
        q += ", ?"
        qud.params = append(qud.params, limits[1])
    }
    
    logQuery(qud.table, q, qud.params)
    
    res, err := qud.db.Query(q, qud.params...)
    catch(err, q, qud.params)
    
    qud.reset()
    return res, values
}

func (qud *QUD) Update(fld interface{}) int64 {
    var keys string
    var params []interface{}
    
    switch fields := fld.(type) {
        case string:
            keys = fields
        case Fields:
            for k, v := range fields {
                if "" != keys {
                    keys += ", "
                }
                keys += "'" + k + "' = ?"

                params = append(params, v)
            }
    }

    qud.params = append(params, qud.params...)
    q := "UPDATE " + qud.table + " SET " + keys + qud.cond

    logQuery(qud.table, q, qud.params)
    
    res, err := qud.db.Exec(q, qud.params...)
    catch(err, q, qud.params)
    
    numRows, err := res.RowsAffected()
    catch(err, "", nil)
    
    qud.reset()
    return numRows    
}

func (qud *QUD) Delete(args ...interface{}) (err error) {
    if len(args) > 0 {
        qud.Seek(args...)
    }
    q := "DELETE FROM " + qud.table + qud.cond
    logQuery(qud.table, q, qud.params)
    _, err = qud.db.Exec(q, qud.params...)
    //catch(err, q, qud.params)
    qud.reset()
    return
}

// all calls shoud be chained .Table().Seek().Get()
// but someone can reuse .Table() multiple times
// so we need to clean conditional part of object (cond & params)
// for just in case
func (qud *QUD) reset(args ...interface{}) {
    qud.cond = ""
    qud.group = ""
    qud.order = ""
    qud.params = []interface{}{}
}

////////////////////////////////////////////////////////////////////

func JoinSlice(a []int64) string {
    b := make([]string, len(a))
    for i, v := range a {
        b[i] = strconv.FormatInt(v, 10)
    }
    return strings.Join(b, ", ")
}

func catch(err error, q string, p interface{}) {
    if nil != err {
        s := fmt.Sprintf("%s : %s : %s", err, q, p)
        panic(errors.New(s))
    }
}
