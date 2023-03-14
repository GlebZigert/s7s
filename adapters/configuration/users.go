package configuration

import (
    "time"
    "strings"
    "crypto/md5"
    "encoding/hex"
    "database/sql"

    "s7server/dblayer"
    "s7server/api"
)

const userRelations = `link IN ("user-zone", "user-device", "user-rule")`

/////////////////////////////////////////////////////////////////////
///////////////////////////// U S E R S /////////////////////////////
/////////////////////////////////////////////////////////////////////

// TODO: validate user type
//        {id: 1, text: "Группа"},
//        {id: 2, text: "Сотрудник"},
//        {id: 3, text: "Посетитель"},
//        {id: 4, text: "Автомобиль"}

func (cfg *Configuration) currentShiftId(userId int64) (id int64, err error) {
    var timestamp, lastEvent int64
    shiftEvents := []int64{api.EC_USER_SHIFT_STARTED, api.EC_USER_SHIFT_COMPLETED}

    fields := dblayer.Fields{
        "id": &id,
        "class": &lastEvent,
        "MAX(time)": &timestamp}

    err = db.Table("events").
        Seek("service_id = 0 AND user_id = ? AND class", userId, shiftEvents).
        Group("user_id").
        First(nil, fields)
    if sql.ErrNoRows == err {
        err = nil // it's not an error
    }

    if lastEvent != api.EC_USER_SHIFT_STARTED {
        id = 0
    }
    return
}


func (cfg *Configuration) dbUpdateUserPicture(id int64, picture []byte) (err error) {
    _, err = db.Table("users").Seek(id).Update(nil, dblayer.Fields {"photo": &picture})
    return
}

func (cfg *Configuration) dbLoadUserPicture(id int64) (picture []byte, err error) {
    fields := dblayer.Fields {"photo": &picture}
    err = db.Table("users").Seek(id).First(nil, fields)
    if sql.ErrNoRows == err {
        err = nil // it's not an error
    }
    return
}


func (cfg *Configuration) loadUserCards(userId int64) (list []string, err error) {
    var pin, card string
    list = make([]string, 0)
    fields := dblayer.Fields{"pin": &pin, "card": &card}

    err = db.Table("cards").Seek("user_id = ?", userId).Rows(nil, fields).Each(func () {
        if "" != pin {
            card = pin + " " + card
        }
        list = append(list, card)
    })
    return
}

func (cfg *Configuration) saveUserCards(user *User) (err error) {
    var userId int64
    var card string
    var badCards []string
    var key int64
    var keys []int64
    cards := make(map[int64] [2]string)
    
    // 1. make keys
    for i := range user.Cards {
        parts := strings.Split(" " + strings.TrimSpace(user.Cards[i]), " ")
        card = parts[len(parts) - 1]
        key = keyFromCard(card)
        keys = append(keys, key)
        cards[key] = [2]string{parts[len(parts) - 2], card}
    }
    
    tx, err := db.Tx(qTimeout)
    if nil != err {return}
    defer func () {completeTx(tx, err)}()

    // 2. load cards from db
    table := db.Table("cards")
    fields := dblayer.Fields{"user_id": &userId, "key": &key}
    err = table.Seek("key", keys).Rows(tx, fields).Each(func() {
        if card, ok := cards[key]; ok && user.Id != userId {
            // someone else's card
            badCards = append(badCards, card[0] + " " + card[1])
            delete(cards, key)
        }
    })

    if nil != err {return}

    // 3. clean old
    // TODO: delete unused, update only updated cards?
    err = table.Delete(tx, "user_id = ?", user.Id)
    if nil != err {return}

    // 4. insert cards
    for key, pc := range cards {
        fields = dblayer.Fields{"user_id": user.Id, "pin": pc[0], "card": pc[1], "key": key}
        _, err = table.Insert(tx, fields)
        if nil != err {return}
    }

    if len(badCards) > 0 {
        user.Warnings = append(user.Warnings, "Следующие карты не были сохранены:\n" + strings.Join(badCards, "\n"))
    }
    // 5. TODO: notify subscribers?
    return
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

func (cfg *Configuration) dbUpdateUser(user *User, filter map[string] interface{}) (err error) {
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
        "settings":     user.Settings,
        "login": user.Login}

    if nil != filter {
        for field := range fields {
            camelField := toCamelCase(field)
            if _, ok := filter[camelField]; !ok {
                delete(fields, field)
            }
        }
    }

    if len(user.Password) > 0 {
        fields["token"] = md5hex(authSalt + user.Password)
    }

    // if 0 == len(fields) {return}

    if 0 != len(fields) { // user fields update required
        var userId int64
        tx, eee := db.Tx(qTimeout)
        err = eee
        if nil != err {return}
        //defer func () {completeTx(tx, err)}()
        
        // check for duplicate login
        if "" != fields["login"] {
            err = db.Table("users").Seek("archived = 0 AND login = ?", fields["login"]).First(tx, dblayer.Fields {"id": &userId})
        }

        if sql.ErrNoRows == err {
            err = nil // it's not an error
        }

        if nil != err {return}

        if 0 != userId && userId != user.Id {
            tx.Rollback()
            user.Errors = append(user.Warnings, "указанный логин занят")
            return // duplicate login
        }

        if 0 == user.Id {
            if nil != fields["name"] && "" != fields["name"] {
                fields["parent_id"] = user.ParentId
                fields["type"] = user.Type
                fields["role"] = user.Role
                //fields["archived"] = user.Archived
                user.Id, err = db.Table("users").Insert(tx, fields)
            }
        } else {
            _, err = db.Table("users").Seek(user.Id).Update(tx, fields)
        }

        if nil != err {
            tx.Rollback()
            return
        }
        tx.Commit()
    }
    
    // update user's relations if needed
    if nil == err && nil != filter["zones"] {
        err = cfg.SaveLinks(user.Id, "user-zone", user.Zones)
    }
    if nil == err && nil != filter["devices"] {
        cfg.SaveLinks(user.Id, "user-device", user.Devices)
    }
    if nil == err && nil != filter["cards"] {
        cfg.saveUserCards(user)
    }

    if newGroup {
        cfg.cache.checkReset(0) // TODO: just update map, don't use DB
    }
    return
}

// for internal usage - recursively delete whole branch
func (cfg *Configuration) deleteBranch(tx *sql.Tx, ids []int64) (err error) {
    var groups []int64
    var all []int64
    var userId int64
    var userType int
    
    // 1. fing subgroups & all children
    fields := dblayer.Fields {"id": &userId, "type": &userType}
    cond := "archived = 0 AND parent_id"
    err = db.Table("users").Seek(cond, ids).Rows(tx, fields).Each(func() {
        if 1 == userType {
            groups = append(groups, userId)
        }
        all = append(all, userId)
    })
    if nil != err {return}

    if 0 == len(all) {
        return
    }

    // 2. "delete" sub-groups if needed
    if len(groups) > 0 {
        err = cfg.deleteBranch(tx, groups)
    }
    if nil != err {return}
    
    // "delete" direct subnodes for groups
    fields = dblayer.Fields{"archived": time.Now().Unix()}
    if nil == err {
        _, err = db.Table("users").Seek("archived = 0 AND id", all).Update(tx, fields)
    }
    
    // delete cards
    if nil == err {
        err = db.Table("cards").Delete(tx, "user_id", all)
    }
    
    // delete links
    if nil == err {
        err = db.Table("external_links").Delete(tx, userRelations + ` AND source_id`, all)
    }
    return
}

func (cfg *Configuration) dbDeleteUser(id int64) (err error) {
	tx, err := db.Tx(qTimeout)
    if nil != err {
        return
    }
    
    // delete from the end of branch (prevent loss of nodes in case of error)
    err = cfg.deleteBranch(tx, []int64{id})
    // if was no errors, delete "root" of all barnches
    if nil == err {
        _, err = db.Table("users").Seek(id).Update(tx, dblayer.Fields{"archived": time.Now().Unix()})
    }
    if nil == err {
        err = db.Table("cards").Delete(tx, "user_id = ?", id)
    }
    if nil == err {
        err = db.Table("external_links").Delete(tx, userRelations + ` AND source_id = ?`, id)
    }
    // TODO: clean broken links for user links, if users "deleted" instead "archived"
    // SELECT ul.user_id FROM user_links ul LEFT JOIN users u ON ul.user_id = u.id AND u.archived = false WHERE u.id IS NULL;
    if nil == err {
        err = tx.Commit()
        cfg.cache.checkReset(id)
    } else {
        tx.Rollback() // don't overwrite existing error
    }
    return
}

func (cfg *Configuration) loadUsers() (list []User, err error) {
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
        "login":        &user.Login,
        "settings":     &user.Settings,
    }

    err = db.Table("users").Seek("archived = 0").Rows(nil, fields).Each(func() {
        list = append(list, *user)
        userMap[user.Id] = len(list) - 1
    })
    if nil != err {return}
    
    // read cards
    var userId int64
    var card string
    fields = dblayer.Fields {"user_id": &userId, "card": &card}
    err = db.Table("cards").Rows(nil, fields).Each(func(){
        if pos, ok := userMap[userId]; ok {
            list[pos].Cards = append(list[pos].Cards, card)
        }
    })
    return
}

// returns nil if user not found, without sql.ErrNoRows
func (cfg *Configuration) GetUser(id int64) (user *User, err error) {
    defer func () {cfg.complaints <- de(err, "GetUser")}()
    user = new(User)
    //cfg.tables["users"].query("fields").where("cond")
    //db.Table("users").Find("cond").Get(nil, "list")
    fields := dblayer.Fields {
        "id":           &user.Id,
        //"archived":     &user.Archived,
        "parent_id":    &user.ParentId,
        "type":         &user.Type,
        "role":         &user.Role,
        "name":         &user.Name,
        "surename":     &user.Surename,
        "middle_name":  &user.MiddleName,
        "rank":         &user.Rank,
        "organization": &user.Organization,
        "position":     &user.Position,
        "login":        &user.Login,
        "settings":     &user.Settings,
    }

    err = db.Table("users").Seek("archived = 0 AND id = ?", id).First(nil, fields)
    if nil != err {
        user = nil // it's not an error
    }
    if sql.ErrNoRows == err {
        err = nil // NoRows is not really an error
    }
    
    return
}


/////////////////////////////////////////////////////////////////////
///////////////////////////// E X T R A /////////////////////////////
/////////////////////////////////////////////////////////////////////

func toSnakeCase(camel string) (snake string) {
	for i, r := range camel {
		c := string(r)
		lower := strings.ToLower(c)
		if 0 == i {
			snake += lower
		} else {
			if c == lower {
				snake += lower
			} else {
				snake += "_" + lower
			}
		}
	}
	return
}

func toCamelCase(snake string) (camel string) {
	list := strings.Split(strings.ToLower(snake), "_")
	for i := range list {
		if 0 == i {
			continue
		}
		list[i] = strings.Title(list[i])
	}
	return strings.Join(list, "")
}


func md5hex(text string) string {
   hash := md5.Sum([]byte(text))
   return hex.EncodeToString(hash[:])
}

