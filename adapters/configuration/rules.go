package configuration

import (
    "time"
//    "s7server/api"
    "s7server/dblayer"
)

const MAX_HARDWARE_RULES = 7

/////////////////////////////////////////////////////////////////////
/////////////////////////// ACCESS RULES ////////////////////////////
/////////////////////////////////////////////////////////////////////

func (cfg *Configuration) loadAllRules() ([]*Rule, error) {
    return cfg.loadRules(false)
}

func (cfg *Configuration) loadHWRules() ([]*Rule, error) {
    return cfg.loadRules(true)
}

func (cfg *Configuration) loadRules(hwOnly bool) (list []*Rule, err error) {
    rule := new(Rule)

    fields := dblayer.Fields {
        "id": &rule.Id,
        "name": &rule.Name,
        "description": &rule.Description,
        "start_date": &rule.StartDate,
        "end_date": &rule.EndDate,
        "priority": &rule.Priority}

    idPos := make(map[int64]int)
    var ids []int64
    dataset := db.Table("accessrules")
    if hwOnly {
        dataset = dataset.Seek("priority = 0")
    }
    err = dataset.Rows(nil, fields).Each(func() {
        tmp := *rule
        list = append(list, &tmp) // TODO: try without tmp
        
        ids = append(ids, rule.Id)
        idPos[rule.Id] = len(list) - 1
    })
    if nil != err {return}
    
    //
    // get timeranges
    //
    tr := new(TimeRange)
    fields = dblayer.Fields {
        "rule_id": &tr.RuleId,
        "direction": &tr.Direction,
        "from": &tr.From,
        "to": &tr.To}
    
    err = db.Table("timeranges").Seek("rule_id", ids).Rows(nil, fields).Each(func() {
        list[idPos[tr.RuleId]].TimeRanges = append(list[idPos[tr.RuleId]].TimeRanges, *tr)
    })
    return
}

/*func (cfg *Configuration) dbCountHWRules() (count int64, err error) {
    fld := dblayer.Fields {"COUNT(*)": &count}
    err = db.Table("accessrules").Seek("priority = 0").First(nil, fld)
    if sql.ErrNoRows == err {
        err = nil // it's not an error
    }
    return
}*/

func (cfg *Configuration) dbUpdateRules(rule *Rule) (err error) {
    endDate := rule.EndDate.Add(24 * time.Hour - 1 * time.Second)
    fld := dblayer.Fields {
        "name": &rule.Name,
        "description": &rule.Description,
        "start_date": &rule.StartDate,
        "end_date": &endDate,
        "priority": &rule.Priority}

    tx, err := db.Tx(qTimeout)
    if nil != err {return}
    defer func () {completeTx(tx, err)}()

    if 0 >= rule.Id { // new
        rule.Id, err = db.Table("accessrules").Insert(tx, fld)
        /*var count int64
        count, err = cfg.dbCountHWRules()
        if nil != err {return}
        if count > MAX_HARDWARE_RULES { // TODO: rewrite using tx.Rollback()
            err = db.Table("accessrules").Delete(nil, rule.Id)
            rule.Id = 0
        }*/
    } else { // update
        err = db.Table("timeranges").Delete(tx, "rule_id", rule.Id)
        if nil == err {
            _, err = db.Table("accessrules").Seek(rule.Id).Update(tx, fld)
        }
    }
    if nil != err {return}
    
    table := db.Table("timeranges")
    for _, tr := range rule.TimeRanges {
        _, err = table.Insert(tx, dblayer.Fields {
            "rule_id": &rule.Id,
            "direction": &tr.Direction,
            "from": &tr.From,
            "to": &tr.To})
        if nil != err {break}
    }
    return
}

func (cfg *Configuration) dbDeleteRule(id int64) (err error) {
    tx, err := db.Tx(qTimeout)
    if nil != err {return}
    defer func () {completeTx(tx, err)}()

    err = db.Table("external_links").Delete(tx, `link="user-zone" AND flags`, id)
    if nil == err {
        err = db.Table("timeranges").Delete(tx, "rule_id", id)
    }
    if nil == err {
        err = db.Table("accessrules").Delete(tx, id)
    }
    return
}
