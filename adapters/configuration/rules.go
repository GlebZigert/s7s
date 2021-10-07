package configuration

import (
    "time"
//    "../../api"
    "../../dblayer"
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

    dataset := cfg.Table("accessrules")
    if hwOnly {
        dataset = dataset.Seek("priority = 0")
    }
    rows, values := dataset.Get(fields)
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
    
    rows, values = cfg.Table("timeranges").Seek("rule_id", ids).Get(fields)
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
    rows, values := cfg.Table("accessrules").Seek("priority = 0").Get(fld)
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
        rule.Id = cfg.Table("accessrules").Insert(fld)
        if cfg.dbCountHWRules() > MAX_HARDWARE_RULES {
            cfg.Table("accessrules").Delete(rule.Id)
            rule.Id = 0
        }
    } else { // update
        cfg.Table("timeranges").Delete("rule_id", rule.Id)
        cfg.Table("accessrules").Seek(rule.Id).Update(fld)
    }
    if rule.Id > 0 {
        table := cfg.Table("timeranges")
        for _, tr := range rule.TimeRanges {
            table.Insert(dblayer.Fields {
                "rule_id": &rule.Id,
                "direction": &tr.Direction,
                "from": &tr.From,
                "to": &tr.To})
        }
    }
}

func (cfg *Configuration) dbDeleteRule(id int64) {
    //cfg.Table("external_links").Delete(`link="user-rule" AND target_id`, id)
    cfg.Table("external_links").Delete(`link="user-zone" AND flags`, id)
    cfg.Table("timeranges").Delete("rule_id", id)
    cfg.Table("accessrules").Delete(id)
}
