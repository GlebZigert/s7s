package z5rweb

import (
//    "log"
//    "database/sql"
//    _ "github.com/mattn/go-sqlite3"
//    "../../api"
    "../../dblayer"
)


var tables []string = []string {`
    CREATE TABLE IF NOT EXISTS timezones (
        zone        INTEGER PRIMARY KEY,
        begin       TEXT NOT NULL,
        end         TEXT NOT NULL,
        days        TEXT NOT NULL
    )`,`
    CREATE TABLE IF NOT EXISTS cards (
        card        TEXT PRIMARY KEY,
        flags       INTEGER NOT NULL,
        timezone    INTEGER NOT NULL
    )`}

func timezoneFields(tz *Timezone) dblayer.Fields {
    return dblayer.Fields {
        "zone": &tz.Zone,
        "begin": &tz.Begin,
        "end": &tz.End,
        "days": &tz.Days}
}

func cardFields(card *Card) dblayer.Fields {
    return dblayer.Fields {
//        "id":           &card.Id,
        "card":         &card.Card,
        "flags":        &card.Flags,
        "timezone":     &card.Timezone}
}
