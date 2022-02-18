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

func (cfg *Configuration) loadAllRules() []*Rule{
    return cfg.loadRules(false)
}

func (cfg *Configuration) loadHWRules() []*Rule{
    return cfg.loadRules(true)
}

func (cfg *Configuration) loadRules(hwOnly bool) []*Rule{
    var list []*Rule
    rule := new(Rule)

    fields := dblayer.Fields {
        "id": &rule.Id,
        "name": &rule.Name,
        "description": &rule.Description,
        "start_date": &rule.StartDate,
        "end_date": &rule.EndDate,
        "priority": &rule.Priority}

    dataset := db.Table("accessrules")
    if hwOnly {
        dataset = dataset.Seek("priority = 0")
    }
    rows, values, _ := dataset.Get(nil, fields)
    defer rows.Close() // TODO: defer triggered for this rows?

    idPos := make(map[int64]int)
    var ids []int64
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        tmp := *rule
        list = append(list, &tmp) // TODO: try without tmp
        
        ids = append(ids, rule.Id)
        idPos[rule.Id] = len(list) - 1
    }
    
    //
    // get timeranges
    //
    tr := new(TimeRange)
    fields = dblayer.Fields {
        "rule_id": &tr.RuleId,
        "direction": &tr.Direction,
        "from": &tr.From,
        "to": &tr.To}
    
    rows, values, _ = db.Table("timeranges").Seek("rule_id", ids).Get(nil, fields)
    defer rows.Close() // TODO: defer triggered for this rows?
    
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        //tmp := *tr
        list[idPos[tr.RuleId]].TimeRanges = append(list[idPos[tr.RuleId]].TimeRanges, *tr)
        
    }
    return list
}

func (cfg *Configuration) dbCountHWRules() (count int64) {
    fld := dblayer.Fields {"COUNT(*)": &count}
    rows, values, _ := db.Table("accessrules").Seek("priority = 0").Get(nil, fld)
    defer rows.Close() // TODO: defer triggered for this rows?
    
    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
    }
    return
}

func (cfg *Configuration) dbUpdateRules(rule *Rule) {
    endDate := rule.EndDate.Add(24 * time.Hour - 1 * time.Second)
    fld := dblayer.Fields {
        "name": &rule.Name,
        "description": &rule.Description,
        "start_date": &rule.StartDate,
        "end_date": &endDate,
        "priority": &rule.Priority}
    
    if 0 >= rule.Id { // new
        rule.Id, _ = db.Table("accessrules").Insert(nil, fld)
        if cfg.dbCountHWRules() > MAX_HARDWARE_RULES {
            db.Table("accessrules").Delete(nil, rule.Id)
            rule.Id = 0
        }
    } else { // update
        db.Table("timeranges").Delete(nil, "rule_id", rule.Id)
        db.Table("accessrules").Seek(rule.Id).Update(nil, fld)
    }
    if rule.Id > 0 {
        table := db.Table("timeranges")
        for _, tr := range rule.TimeRanges {
            table.Insert(nil, dblayer.Fields {
                "rule_id": &rule.Id,
                "direction": &tr.Direction,
                "from": &tr.From,
                "to": &tr.To})
        }
    }
}

func (cfg *Configuration) dbDeleteRule(id int64) {
    //db.Table("external_links").Delete(nil, `link="user-rule" AND target_id`, id)
    db.Table("external_links").Delete(nil, `link="user-zone" AND flags`, id)
    db.Table("timeranges").Delete(nil, "rule_id", id)
    db.Table("accessrules").Delete(nil, id)
}
