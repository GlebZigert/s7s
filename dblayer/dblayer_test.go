package dblayer_test

import (
	"testing"
    //"s7server/adapters/configuration"
    "fmt"
    "s7server/dblayer"
    //"encoding/hex"
//    "time"
)

var appendCond = dblayer.AppendCondTesting

func TestMain(m *testing.M) {
    fmt.Println("Starting...")
    var a []interface{}
    ids := []int64{5, 7, 9}
    
    q, params := appendCond("SELECT * FROM table", a)
    fmt.Println(q, params)

    a = []interface{}{ids}
    q, params = appendCond("SELECT * FROM table", a)
    fmt.Println(q, params)

    a = []interface{}{3}
    q, params = appendCond("SELECT * FROM table", a)
    fmt.Println(q, params)

    a = []interface{}{"rule_id", 5}
    q, params = appendCond("SELECT * FROM table", a)
    fmt.Println(q, params)

    
    a = []interface{}{"rule_id", ids}
    q, params = appendCond("SELECT * FROM table", a)
    fmt.Println(q, params)

    a = []interface{}{"parent_id = ? AND rule = ? AND cat_id = ? AND id", 1, "hello", 3, ids}
    q, params = appendCond("SELECT * FROM table", a)
    fmt.Println(q, params)
    
}
