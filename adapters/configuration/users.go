package configuration

import (
//    "strconv"
    "strings"
    "crypto/md5"
    "encoding/hex"
    "../../dblayer"
    "../../api"
)

/////////////////////////////////////////////////////////////////////
///////////////////////////// U S E R S /////////////////////////////
/////////////////////////////////////////////////////////////////////

// TODO: validate user type
//        {id: 1, text: "Группа"},
//        {id: 2, text: "Сотрудник"},
//        {id: 3, text: "Посетитель"},
//        {id: 4, text: "Автомобиль"}

func (cfg *Configuration) shiftStarted(userId int64) (bool, int64) {
    var timestamp, lastEvent, id int64
    shiftEvents := []int64{api.EC_USER_SHIFT_STARTED, api.EC_USER_SHIFT_COMPLETED}

    fields := dblayer.Fields{
        "id": &id,
        "class": &lastEvent,
        "MAX(time)": &timestamp}

    rows, values, _ := db.Table("events").
        Seek("service_id = 0 AND user_id = ? AND class", userId, shiftEvents).
        Group("user_id").
        Get(nil, fields)

    defer rows.Close()
    
    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
    }
    if lastEvent != api.EC_USER_SHIFT_STARTED {
        id = 0
    }
    return lastEvent == api.EC_USER_SHIFT_STARTED, id
}


func (cfg *Configuration) dbUpdateUserPicture(id int64, picture []byte) {
    db.Table("users").Seek(id).Update(nil, dblayer.Fields {"photo": &picture})
}

func (cfg *Configuration) dbLoadUserPicture(id int64) []byte {
    var picture []byte
    fields := dblayer.Fields {"photo": &picture}
    rows, values, _ := db.Table("users").Seek(id).Get(nil, fields)
    defer rows.Close() // TODO: defer triggered for this rows?
    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
    }
    return picture
}


func (cfg *Configuration) loadUserCards(userId int64) (list []string) {
    var pin, card string
    list = make([]string, 0)
    fields := dblayer.Fields{"pin": &pin, "card": &card}

    rows, values, _ := db.Table("cards").Seek("user_id = ?", userId).Get(nil, fields)
    defer rows.Close()

    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        if "" != pin {
            card = pin + " " + card
        }
        list = append(list, card)
    }
    return
}

func (cfg *Configuration) saveUserCards(user *User) {
    var userId int64
    var card string
    var badCards []string
    var onlyCards []string
    
    cards := make(map[string] string)
    
    // 1. filter unsafe cards (non-numeric?)
    for i := range user.Cards {
        //if _, err := strconv.ParseInt(user.Cards[i], 16, 64); err == nil {
        parts := strings.Split(strings.TrimSpace(user.Cards[i]), " ")
        card = parts[len(parts)-1]
        if len(parts) > 1 {
            cards[card] =  parts[0]
        } else {
            cards[card] =  ""
        }
        onlyCards = append(onlyCards, card)
        //} else {
        //  badCards = append(badCards, user.Cards[i])
        //}
    }

    // 2. load cards from db
    table := db.Table("cards")
    cond := "user_id = ? OR card IN('" + strings.Join(onlyCards, "','") + "')"
    fields := dblayer.Fields {"user_id": &userId, "card": &card}
    
    rows, values, _ := table.Seek(cond, user.Id).Get(nil, fields)
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        //cfg.Log("CHECK:", user.Id, card)
        if _, ok := cards[card]; ok && user.Id != userId {
            // someone else's card
            badCards = append(badCards, card) 
            delete(cards, card)
        }
    }
    rows.Close()
    
    // TODO: delete unused, update only updated cards?
    table.Delete(nil, "user_id = ?", user.Id)
    
    // insert cards
    userId = user.Id
    for card, pin := range cards {
        fields["card"] = card
        fields["pin"] = pin
        table.Insert(nil, fields)
        // TODO: notify subscribers
    }
    //cfg.Log("BAD:", badCards)
    //cfg.Log("GOOD:", cards)
    if len(badCards) > 0 {
        user.Warnings = append(user.Warnings, "Следующие карты не были сохранены: " + strings.Join(badCards, "; "))
    }

    // 5. TODO: notify subscribers
}

func (cfg *Configuration) validateUser(user *User) bool {
    // TODO: check unique login
    if user.Type < 0 || user.Type > 5 {
        user.Errors = append(user.Errors, "Неизвестный тип пользователя")
    }
    // new user && arm access && short password
    if 0 == user.Id && user.Role > 0 && len(user.Password) < 4 {
        user.Errors = append(user.Errors, "Короткий пароль")
    }
    // old user && arm access && password update && short password
    if 0 != user.Id && user.Role > 0 && len(user.Password) > 0 && len(user.Password) < 4 {
        user.Errors = append(user.Errors, "Короткий пароль")
    }
    if user.Type != 2 {
        user.Role = 0
    }
    
    return len(user.Errors) == 0
}

func (cfg *Configuration) dbUpdateUser(user *User, filter map[string] interface{}) {
    if !cfg.validateUser(user) {
        cfg.Err("Bad user params:", user)
        return
    }

    newGroup := 0 == user.Id && 1 == user.Type 
    
    // updateable fields
    fields := dblayer.Fields {
        "name":         user.Name,
        "surename":     user.Surename,
        "middle_name":  user.MiddleName,
        "rank":         user.Rank,
        "organization": user.Organization,
        "position":     user.Position,
        "login": user.Login}

    if nil != filter {
        for field := range fields {
            if _, ok := filter[field]; !ok {
                delete(fields, field)
            }
        }
    }
    
    if len(user.Password) > 0 {
        fields["token"] = md5hex(authSalt + user.Password)
    }
    
    if 0 == user.Id {
        if nil != fields["name"] && "" != fields["name"] {
            fields["parent_id"] = user.ParentId
            fields["type"] = user.Type
            fields["role"] = user.Role
            fields["archived"] = user.Archived
            user.Id, _ = db.Table("users").Insert(nil, fields)
        }
    } else if len(fields) > 0 {
        db.Table("users").Seek(user.Id).Update(nil, fields)
    }
    if 0 != user.Id {
        if nil != filter["zones"] {
            cfg.SaveLinks(user.Id, "user-zone", user.Zones)
        }
        if nil != filter["devices"] {
            cfg.SaveLinks(user.Id, "user-device", user.Devices)
        }
        if nil != filter["cards"] {
            cfg.saveUserCards(user)
        }

        if newGroup {
            cfg.cacheRelations() // TODO: just update map, don't use DB
        }
    }
}

// for internal usage - recursively delete whole branch
func (cfg *Configuration) deleteBranch(ids []int64) {
    var groups []int64
    var userId int64
    cond := "type = 1 AND archived = false AND parent_id"
    // fing subgroups
    fields := dblayer.Fields {"id": &userId}
    rows, _, _ := db.Table("users").Seek(cond, ids).Get(nil, fields)
    for rows.Next() {
        err := rows.Scan(&userId)
        catch(err)
        groups = append(groups, userId)
    }; rows.Close() // don't use defer due to recursion
    
    // "delete" sub-subnodes if needed
    if len(groups) > 0 {
        cfg.deleteBranch(groups)
    }

    // if no errors, "delete" direct subnodes of current parents list
    fields = dblayer.Fields{"archived": true}
    db.Table("users").Seek(cond, ids).Update(nil, fields)
    db.Table("cards").Delete(nil, "user_id", ids)
    db.Table("external_links").Delete(nil, `link IN ("user-zone", "user-device") AND source_id`, ids)
}

func (cfg *Configuration) dbDeleteUser(id int64) {
    // delete from the end of branch (prevent loss of nodes in case of error)
    cfg.deleteBranch([]int64{id})
    // if was no errors, delete "root" of all barnch
    db.Table("users").Seek(id).Update(nil, "archived = true")
    db.Table("cards").Delete(nil, "user_id = ?", id)
    db.Table("external_links").Delete(nil, `link IN ("user-zone", "user-device") AND source_id = ?`, id)
    // TODO: clean broken links for user links, if users "deleted" instead "archived"
    // SELECT ul.user_id FROM user_links ul LEFT JOIN users u ON ul.user_id = u.id AND u.archived = false WHERE u.id IS NULL;
   
    if _, ok := cfg.cache.parents[id]; ok { // rebuild cache if userId is group
        cfg.cacheRelations()
    }
}

func (cfg *Configuration) loadUsers() (list []User) {
    userMap := make(map[int64]int) // cache for user_id <-> card mapping
    user := new(User)
    fields := dblayer.Fields {
        "id":           &user.Id,
        "parent_id":    &user.ParentId,
        "type":         &user.Type,
        "role":         &user.Role,
        "name":         &user.Name,
        "surename":     &user.Surename,
        "middle_name":  &user.MiddleName,
        "rank":         &user.Rank,
        "organization": &user.Organization,
        "position":     &user.Position,
        "login":        &user.Login}

    rows, values, _ := db.Table("users").Seek("archived = false").Get(nil, fields)
    defer rows.Close()
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, *user)
        userMap[user.Id] = len(list) - 1
    }
    

    var userId int64
    var card string

    fields = dblayer.Fields {"user_id": &userId, "card": &card}
    rows, values, _ = db.Table("cards").Get(nil, fields)
    defer rows.Close()
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        if pos, ok := userMap[userId]; ok {
            list[pos].Cards = append(list[pos].Cards, card)
        }
    }
    return
}
func (cfg *Configuration) childrenList(parentId int64) (list []int64) {
    var id int64
    fields := dblayer.Fields {"id": &id}

    rows, values, _ := db.Table("users").
        Seek("archived = false AND parent_id = ?", parentId).
        Get(nil, fields)
    defer rows.Close()
    for rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        list = append(list, id)
    }
    return
}

func (cfg *Configuration) GetUser(id int64) *User {
    user := new(User)
    //cfg.tables["users"].query("fields").where("cond")
    //db.Table("users").Find("cond").Get(nil, "list")
    fields := dblayer.Fields {
        "id":           &user.Id,
        "archived":     &user.Archived,
        "parent_id":    &user.ParentId,
        "type":         &user.Type,
        "role":         &user.Role,
        "name":         &user.Name,
        "surename":     &user.Surename,
        "middle_name":  &user.MiddleName,
        "rank":         &user.Rank,
        "organization": &user.Organization,
        "position":     &user.Position,
        "login":        &user.Login}

    rows, values, _ := db.Table("users").Seek("archived = false AND id = ?", id).Get(nil, fields)
    defer rows.Close()

    if rows.Next() {
        err := rows.Scan(values...)
        catch(err)
        return user
    }

    return nil
}


/////////////////////////////////////////////////////////////////////
///////////////////////////// E X T R A /////////////////////////////
/////////////////////////////////////////////////////////////////////

func md5hex(text string) string {
   hash := md5.Sum([]byte(text))
   return hex.EncodeToString(hash[:])
}

