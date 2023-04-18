package dblayer

import (
    "log"
    "time"
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
    db          *sql.DB
    timeout     time.Duration
}

type QUD struct { // Query-Update-Delete
    DBLayer
    table   string
    //db      *sql.DB
    //tx      *sql.Tx
    cond    string      // WHERE a = ? AND b = ?
    group   string
    order   string
    params  []interface{} // params for condition expression
}

type Rows struct {
    rows    *sql.Rows
    values  []interface{}
    err     error
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

func (dbl *DBLayer) MakeTables(tables []string, strict bool) (err error){
    for i := 0; i < len(tables) && (nil == err || !strict); i++ {
        //log.Println(tables[i])
        ctx, _ := context.WithTimeout(context.TODO(), dbl.timeout)
        _, err = dbl.db.ExecContext(ctx, tables[i])
    }
    return
}

func (dbl *DBLayer) Bind(db *sql.DB, timeout int) (err error) {
    dbl.db = db
    dbl.timeout = time.Duration(timeout) * time.Millisecond
    return
}

func (dbl *DBLayer) Close() (err error) {
    return dbl.db.Close()
}


func (dbl *DBLayer) Table(t string) *QUD {
    // TODO: maybe db: &dbl.DB ?
    //return &QUD{table: t, db: dbl.db}
    return &QUD{table: t, DBLayer: *dbl}
}

func (dbl *DBLayer) Tx(ms int) (*sql.Tx, error) {
    ctx := context.Background()
    ctx, _ = context.WithTimeout(ctx, time.Duration(ms) * time.Millisecond)
    return dbl.db.BeginTx(ctx, nil)
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
        _, err = qud.Insert(tx, fields)
    } else if 0 == *pId { // new WITH id
        delete(fields, "id")
        *pId, err = qud.Insert(tx, fields)
    } else { // update
        delete(fields, "id")
        _, err = qud.Seek(*pId).Update(tx, fields)
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
    qud.params = values
    res, err := qud.execQuery(tx, q)
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

func (qud *QUD) First(tx *sql.Tx, mymap Fields) (err error) {
    rows, values, err := qud.get(tx, "", mymap, 1)
    if nil != err {return}
    defer rows.Close()
    if rows.Next() {
        err = rows.Scan(values...)
    } else {
        err = sql.ErrNoRows
    }
    if nil == err {
        err = rows.Err()
    }
    return
}

func (qud *QUD) Get(tx *sql.Tx, mymap Fields, limits ...int64) (*sql.Rows, []interface{}, error) {
    return qud.get(tx, "", mymap, limits...)
}

func (qud *QUD) GetDistinct(tx *sql.Tx, mymap Fields, limits ...int64) (*sql.Rows, []interface{}, error) {
    return qud.get(tx, "DISTINCT", mymap, limits...)
}

func (qud *QUD) get(tx *sql.Tx, hint string, mymap Fields, limits ...int64) (res *sql.Rows, values []interface{}, err error) {
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
    
    if nil == tx {
        ctx, _ := context.WithTimeout(context.TODO(), qud.timeout)
        res, err = qud.db.QueryContext(ctx, q, qud.params...)
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
    
    res, err := qud.execQuery(tx, q)
    
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
    
    _, err = qud.execQuery(tx, q)

    qud.reset()
    return
}

func (qud *QUD) execQuery(tx *sql.Tx, q string) (sql.Result, error) {
    if nil == tx {
        ctx, _ := context.WithTimeout(context.TODO(), qud.timeout)
        return qud.db.ExecContext(ctx, q, qud.params...)
    } else {
        return tx.Exec(q, qud.params...)
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

////////////////////////////////////////

func (qud *QUD) DistinctRows(tx *sql.Tx, mymap Fields, limits ...int64) (*Rows) {
    rows, values, err := qud.get(tx, "DISTINCT", mymap, limits...)
    return &Rows{rows, values, err}
}

func (qud *QUD) Rows(tx *sql.Tx, mymap Fields, limits ...int64) (*Rows) {
    rows, values, err := qud.get(tx, "", mymap, limits...)
    return &Rows{rows, values, err}
}

func (r *Rows) Each(ready func()) (err error) {
    if nil != r.err {
        return r.err
    }
    defer r.rows.Close()
    for r.rows.Next() {
        err = r.rows.Scan(r.values...)
        if nil != err {
            return
        }
        if nil != ready {
            ready()
        }
    }
    if nil == err {
        err = r.rows.Err()
    }
    return
}

////////////////////////////////////////////////////////////////////
/*
func Scan(rows *sql.Rows, values []interface{}, store func()) (err error) {
    for rows.Next() {
        err = rows.Scan(values...)
        if nil != err {
            return
        }
        store()
    }
    if nil == err {
        err = rows.Err()
    }
    return
}*/

func JoinSlice(a []int64) string {
    b := make([]string, len(a))
    for i, v := range a {
        b[i] = strconv.FormatInt(v, 10)
    }
    return strings.Join(b, ", ")
}
