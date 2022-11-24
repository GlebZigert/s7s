package configuration

import (
    "fmt"
    "s7server/api"
)

// TODO: made all fields NOT NULL
// due to sql: Scan error on column index 3, name "r": converting NULL to float32 is unsupported

var tableUpdates []string = []string {
// create default user
    fmt.Sprintf("INSERT INTO users(id, type, role, name, login, token) " +
                `VALUES(%d, %d, %d, "%s", "%s", "%s")`,
                1, api.UT_PERSONAL, api.ARM_ADMIN,
                "Администратор", "Администратор",
                md5hex(authSalt + "Start7")),
    "INSERT INTO zones VALUES(1, 'Внешняя территория', 0, NULL)",
    
    // MIGRATIONS HERE >>>
    //"ALTER TABLE users RENAME COLUMN archived TO deleted",
    //"ALTER TABLE users ADD COLUMN archived INTEGER NOT NULL DEFAULT 0",
    //"UPDATE users SET archived = strftime('%s') WHERE deleted = true",
    //"ALTER TABLE events ADD COLUMN zone_id INTEGER NOT NULL DEFAULT 0",
    //"ALTER TABLE zones ADD COLUMN max_visitors INTEGER NOT NULL DEFAULT 0",
    "ALTER TABLE cards ADD COLUMN key INTEGER NOT NULL DEFAULT 0",
}

var tables []string = []string {`
    CREATE TABLE IF NOT EXISTS settings (
        name        TEXT PRIMARY KEY,
        value       TEXT
    )`,`
    CREATE TABLE IF NOT EXISTS maps (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        type        TEXT,
        name        TEXT,
        cx          REAL,
        cy          REAL,
        zoom        REAL,
        picture     BLOB
    )`,`
    CREATE TABLE IF NOT EXISTS shapes (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        map_id      INTEGER,
        service_id  INTEGER,
        device_id   INTEGER,
        type        TEXT,
        x           REAL,
        y           REAL,
        z           REAL,
        r           REAL,
        w           REAL,
        h           REAL,
        data        TEXT
        
    )`,
    // @archived - timestamp when deleted
    `CREATE TABLE IF NOT EXISTS services (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        type        TEXT,
        title       TEXT,
        host        TEXT,
        login       TEXT,
        password    TEXT NOT NULL DEFAULT '',
        keep_alive  INTEGER,
        db_host     TEXT,
        db_name     TEXT,
        db_login    TEXT,
        db_password TEXT NOT NULL DEFAULT '',
        archived    INTEGER
    )`,
    // devices cache for unique device_id for all subsystems
    // @handle - original id or composite key
    // @type - reserved (use for virtual devices, groups, etc.)
    // @data - reserved for storing additional JSON-data for device
    `CREATE TABLE IF NOT EXISTS devices (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        parent_id   INTEGER,
        type        INTEGER,
        last_seen   DATETIME,
        service_id  INTEGER,
        handle      TEXT,
        name        TEXT,
        data        TEXT NOT NULL DEFAULT ''
    )`,
    `CREATE INDEX IF NOT EXISTS system_handle ON devices (service_id, handle)`,
    // TODO: @password is unused
    // @token: auth token - md5(login+salt+pass)
    `CREATE TABLE IF NOT EXISTS users (
        id              INTEGER PRIMARY KEY AUTOINCREMENT,
        parent_id       INTEGER NOT NULL DEFAULT 0,
        type            INTEGER NOT NULL,
        role            INTEGER NOT NULL,
        archived        INTEGER NOT NULL DEFAULT 0,
        name            TEXT NOT NULL,
        surename        TEXT NOT NULL DEFAULT '',
        middle_name     TEXT NOT NULL DEFAULT '',
        rank            TEXT NOT NULL DEFAULT '',
        organization    TEXT NOT NULL DEFAULT '',
        position        TEXT NOT NULL DEFAULT '',
        login           TEXT NOT NULL DEFAULT '',
        token           TEXT NOT NULL DEFAULT '',
        salt            TEXT NOT NULL DEFAULT '',
        photo           BLOB
    )`,`
    CREATE TABLE IF NOT EXISTS cards (
        user_id     INTEGER NOT NULL,
        pin         TEXT NOT NULL,
        card        TEXT UNIQUE NOT NULL,
        key         INTEGER UNIQUE NOT NULL
    )`,
    // @priority < 100 - "basic" rules
    // @priority >= 1000 - "advanced" rules (online)
    `CREATE TABLE IF NOT EXISTS accessrules (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        name        VARCHAR(32),
        description VARCHAR(255),
        start_date  DATE,
        end_date    DATE,
        priority    INTEGER
    )`,
    // @direction - 0: forbidden, 1: in, 2: out, 3: in-out
    // @day_number - for regular days
    // @date - for special days
    `CREATE TABLE IF NOT EXISTS timeranges (
        rule_id         INTEGER KEY,
        direction       INTEGER NOT NULL,
        'from'          DATETIME,
        'to'            DATETIME
    )`,
    // links with objects from external services
    // @link = "zone-device", "user-device", "user-rule" (dirty hack :), etc.
    // "devices" for device #target_id in service #scope_id
    // @scope_id - e.g. service_id or something else
    // @flags - bitwise data
    // TODO: @data is unused?
    `CREATE TABLE IF NOT EXISTS external_links (
        link            TEXT,
        source_id       INTEGER KEY,
        target_id       INTEGER,
        scope_id        INTEGER,
        flags           INTEGER,
        data            TEXT
    )`,
    // ZONES for ACS
    `CREATE TABLE IF NOT EXISTS zones (
        id              INTEGER PRIMARY KEY AUTOINCREMENT,
        name            TEXT NOT NULL,
        max_visitors    INTEGER,
        archived        DATETIME
    )`,
    // Automatic argorithms
    `CREATE TABLE IF NOT EXISTS algorithms (
        id                  INTEGER PRIMARY KEY AUTOINCREMENT,
        name                TEXT NOT NULL,
        service_id          INTEGER,
        device_id           INTEGER,
        zone_id             INTEGER,
        user_id             INTEGER,
        from_state          INTEGER,
        event               INTEGER,
        target_service_id   INTEGER,
        target_device_id    INTEGER,
        target_zone_id      INTEGER,
        command             INTEGER,
        argument            INTEGER,
        extra               TEXT
    )`,
    // Event Log
    `CREATE TABLE IF NOT EXISTS events (
        id                  INTEGER PRIMARY KEY AUTOINCREMENT,
        external_id         INTEGER,
        service_id          INTEGER,
        device_id           INTEGER,
        from_state          INTEGER,
        event               INTEGER,
        commands            TEXT,
        class               INTEGER,
        'text'              TEXT NOT NULL,
        user_id             INTEGER,
        zone_id             INTEGER,
        time                INTEGER,
        'reason'            TEXT NOT NULL DEFAULT '',
        reaction            TEXT NOT NULL DEFAULT ''
    )`}
// CREATE INDEX parent_del ON users (parent_id, deleted);
