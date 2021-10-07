package z5rweb

import (
    //"../configuration"
)
/*
func (svc *Z5RWeb) checkHWAccess(devId int64) {
    // get actual oldCards
    return
    cards := svc.cfg.GetCards(devId)
    
    // get stored on-board oldCards from DB
    oldCards := svc.loadCards()
    
    // fing oldCards to delete
    var delCards []string
    for i := range oldCards {
        card := oldCards[i].Card
        if _, ok := cards[card]; !ok {
            delCards = append(delCards, card)
        }
    }

    // make new timezones
    var rules []*configuration.Rule
    tzCache := make(map[int64]struct{})
    for _, rulesList := range cards {
        for _, rule := range rulesList {
            if _, ok := tzCache[rule.Id]; !ok {
                tzCache[rule.Id] = struct{}{}
                rules = append(rules, rule)
            }
        }
    }
    svc.Log("RULES:", *rules[0], *rules[1])
    
    
    // get stored on-board timezones from DB
    //oldTimezones := svc.loadTimezones()

}
*/