package dblayer

import (
    "log"
    "fmt"
    "time"
    "errors"
    "strings"
    "strconv"
    "context"
    "database/sql"
    //_ "github.com/mattn/go-sqlite3"
)

// usage examples:
// .Table().Insert(nil, )
// .Table().Get(nil, fields, limit)
// .Table().Update(nil, fields)
// .Table().Delete(nil, cond)
// .Table().Seek(cond).Get(nil, fields, limit)
// .Table().Seek(cond).Update(nil, fields)
// .Table().Seek(cond).Delete(nil, )


var LogTables []string

type Fields map[string] interface{}

type DBLayer struct {
    db  *sql.DB
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
        _, err = dbl.db.Exec(tables[i])
    }
    return
}

func (dbl *DBLayer) Bind(db *sql.DB) (err error) {
    dbl.db = db
    return
}

func (dbl *DBLayer) Close() (err error) {
    return dbl.db.Close()
}


func (dbl *DBLayer) Table(t string) *QUD {
    // TODO: maybe db: &dbl.DB ?
    return &QUD{table: t, db: dbl.db}
}

func (dbl *DBLayer) Tx(ms int) (*sql.Tx, error) {
    ctx := context.Background()
    ctx, _ = context.WithTimeout(ctx, time.Duration(ms) * time.Millisecond)
    return dbl.db.BeginTx(ctx, nil)
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
func (qud *QUD) Save(tx *sql.Tx, fields Fields) (err error) {
    var pId *int64
    tmp, ok := fields["id"]
    if ok {
        pId = tmp.(*int64)
    }
    if !ok { // new WITHOUT id field
        _, err = qud.Insert(nil, fields)
    } else if 0 == *pId { // new WITH id
        delete(fields, "id")
        *pId, err = qud.Insert(nil, fields)
    } else { // update
        delete(fields, "id")
        _, err = qud.Seek(*pId).Update(nil, fields)
    }
    //qud.reset()
    return
}

func (qud *QUD) Insert(tx *sql.Tx, fields Fields) (id int64, err error) {
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
    
    //var res sql.Result
    res, err := qud.execQuery(tx, q, qud.params)
    if nil == err {
        id, err = res.LastInsertId()
    }

    return
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

func (qud *QUD) Get(tx *sql.Tx, mymap Fields, limits ...int64) (*sql.Rows, []interface{}, error) {
    return qud.RealGet(tx, "", mymap, limits...)
}

func (qud *QUD) GetDistinct(tx *sql.Tx, mymap Fields, limits ...int64) (*sql.Rows, []interface{}, error) {
    return qud.RealGet(tx, "DISTINCT", mymap, limits...)
}

func (qud *QUD) RealGet(tx *sql.Tx, hint string, mymap Fields, limits ...int64) (res *sql.Rows, values []interface{}, err error) {
    keys := ""
    values = make([]interface{}, len(mymap))

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
    
    //res, err := qud.execQuery(tx, q, qud.params)    

    if nil == tx {
        res, err = qud.db.Query(q, qud.params...)
    } else {
        res, err = tx.Query(q, qud.params...)
    }
    
    qud.reset()
    
    return res, values, err
}

func (qud *QUD) Update(tx *sql.Tx, fld interface{}) (numRows int64, err error) {
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
    
    res, err := qud.execQuery(tx, q, qud.params)
    
    if nil == err {
        numRows, err = res.RowsAffected()
    }
    qud.reset()

    return
}

func (qud *QUD) Delete(tx *sql.Tx, args ...interface{}) (err error) {
    if len(args) > 0 {
        qud.Seek(args...)
    }
    q := "DELETE FROM " + qud.table + qud.cond
    logQuery(qud.table, q, qud.params)
    
    _, err = qud.execQuery(tx, q, qud.params)

    qud.reset()
    return
}

func (qud *QUD) execQuery(tx *sql.Tx, q string, params []interface{}) (sql.Result, error) {
    if nil == tx {
        return qud.db.Exec(q, params...)
    } else {
        return tx.Exec(q, params...)
    }
}

// all calls shoud be chained .Table().Seek().Get(nil, )
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
