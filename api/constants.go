package api

var ARMFilter = func (armFilter map[int64] []int64) map[int64] map[int64] struct{} {
    filter := make(map[int64] map[int64] struct{})
    // [role][class]
    for class, _ := range armFilter {
        for _, role := range armFilter[class] {
            if _, ok := filter[role]; !ok {
                filter[role] = make(map[int64] struct{})
            }
            filter[role][class] = struct{}{}
        }
    }
    return filter
}(map[int64] []int64 { // [armType] => event class to catch if no suitable device/user in event
    //1: nil, // all events
    EC_GLOBAL_ALARM: {ARM_UNIT, ARM_CHECKPOINT, ARM_GUARD, ARM_OPERATOR, ARM_SECRET/*, ARM_BUREAU*/},
    EC_ENTER_ZONE: {ARM_SECRET},
    EC_ACCESS_VIOLATION: {ARM_UNIT, ARM_CHECKPOINT, ARM_GUARD},
    EC_ACCESS_VIOLATION_ENDED: {ARM_UNIT, ARM_CHECKPOINT, ARM_GUARD},
})

// access control RequestPassage codes
const (
    ACS_ACCESS_GRANTED = 0
    ACS_UNKNOWN_CARD = 1
    ACS_ANTIPASSBACK = 2
    ACS_ACCESS_DENIED = 3
    ACS_PIN_REQUIRED = 4
    ACS_WRONG_PIN = 5
    ACS_MAX_VISITORS = 6
)

// User Types
const (
    UT_GROUP = 1
    UT_PERSONAL = 2
    UT_GUEST = 3
    UT_CAR = 4
)

// User ARM
const (
    ARM_ADMIN       = 1 // админ
    ARM_UNIT        = 2 // начальник ВЧ
    ARM_CHECKPOINT  = 3 // начальник КПП
    ARM_GUARD       = 4 // начальник караула
    ARM_OPERATOR    = 5 // оператор
    ARM_SECRET      = 6 // гостайна
    ARM_BUREAU      = 7 // бюро пропусков
)

// Access Modes
const (
    AM_WATCH = 1
    AM_CONTROL = 2
    AM_RELATED_AP = 4 // related access point (same zone)
)

var ClassText = map[int64] string {
    EC_CONNECTION_OK: "Связь установлена",
    EC_CONNECTION_LOST: "Связь отсутствует",
    EC_GLOBAL_ALARM: "Общая тревога",
    EC_INFO_ALARM_RESET: "Сброс тревог",
    EC_USER_LOGGED_IN: "Пользователь подключился",
    EC_ALREADY_LOGGED_IN: "Пользователь уже подключен",
    EC_LOGIN_FAILED: "Ошибка аутентификации",
    EC_USER_LOGGED_OUT: "Пользователь отключился",
    EC_ARM_TYPE_MISMATCH: "Смена типа АРМ недопустима",
    EC_LOGIN_TIMEOUT: "Реквизиты доступа не получены вовремя",
    EC_USERS_LIMIT_EXCEED: "Превышено максимальное число пользователей",
    EC_USER_SHIFT_STARTED: "Начало новой смены",
    EC_USER_SHIFT_COMPLETED: "Смена завершена",
    EC_ACCESS_VIOLATION: "Нарушение режима доступа в зону",
    EC_ACCESS_VIOLATION_ENDED: "Прекращено нарушение режима доступа в зону",
    EC_ONLINE: "Связь установлена",
    EC_LOST: "Связь отсутствует",
    EC_ARMED: "Поставлено на охрану",
    EC_DISARMED: "Снято с охраны",
    EC_POINT_BLOCKED: "Проход запрещён",
    EC_FREE_PASS: "Свободный проход",
    EC_NORMAL_ACCESS: "Штатный доступ",
    EC_ALGO_STARTED: "Алгоритм запущен",
    EC_UPS_PLUGGED: "Питание от сети",
    EC_UPS_UNPLUGGED: "Питание от батарей",
    
    // services
    EC_SERVICE_READY: "Служба работает",
    EC_SERVICE_SHUTDOWN: "Служба остановлена",
    EC_SERVICE_FAILURE: "Сбой в работе внутренней службы",
    EC_SERVICE_ONLINE: "Соединение установлено",
    EC_DATABASE_READY: "БД готова",
    EC_SERVICE_OFFLINE: "Соединение отсутствует",
    EC_DATABASE_UNAVAILABLE: "БД недоступна",
    EC_SERVICE_ERROR: "Внешняя служба работает некорректно",
    EC_DATABASE_ERROR: "Проблемы с БД",
}


// statuses accepted by SetServiceStatus
var serviceStatuses = map[int64] string {
    EC_SERVICE_READY: "self",
    EC_SERVICE_SHUTDOWN: "self",
    EC_SERVICE_FAILURE: "self",
    EC_SERVICE_ONLINE: "tcp",
    EC_SERVICE_OFFLINE: "tcp",
    EC_SERVICE_ERROR: "tcp",
    EC_DATABASE_READY: "db",
    EC_DATABASE_UNAVAILABLE: "db",
    EC_DATABASE_ERROR: "db",
}

// event classes
// event classes may turn into universal codes in the future
const (
    EC_NA = 0 //iota
    // INFO
    EC_INFO                 = 100
    EC_ENTER_ZONE           = 101
    EC_EXIT_ZONE            = 102        // virtual code
    EC_INFO_ALARM_RESET     = 103
    EC_USER_LOGGED_IN       = 104
    EC_USER_LOGGED_OUT      = 105
    EC_ARM_TYPE_MISMATCH    = 106
    EC_LOGIN_TIMEOUT        = 107
    EC_USER_SHIFT_STARTED   = 108
    EC_USER_SHIFT_COMPLETED = 109
    EC_SERVICE_READY        = 110
    EC_SERVICE_SHUTDOWN     = 111
    EC_ARMED                = 112
    EC_DISARMED             = 113
    EC_POINT_BLOCKED        = 114
    EC_FREE_PASS            = 115
    EC_NORMAL_ACCESS        = 116
    EC_ALGO_STARTED         = 117
    
    // OK
    EC_OK                     = 200
    EC_ACCESS_VIOLATION_ENDED = 201
    EC_CONNECTION_OK          = 202
    EC_SERVICE_ONLINE         = 203
    EC_DATABASE_READY         = 204
    EC_ONLINE                 = 205
    EC_UPS_PLUGGED            = 206
    
    // ERROR
    EC_ERROR                = 300
    EC_USERS_LIMIT_EXCEED   = 301
    EC_SERVICE_FAILURE      = 302 // internal error
    EC_SERVICE_ERROR        = 303 // remote service error
    EC_DATABASE_ERROR       = 304
    
    
    // LOST (no link)
    EC_LOST                 = 400
    EC_CONNECTION_LOST      = 401
    EC_SERVICE_OFFLINE      = 402
    EC_DATABASE_UNAVAILABLE = 403
    
    
    // ALARM
    EC_ALARM                = 500
    EC_GLOBAL_ALARM         = 501
    EC_ACCESS_VIOLATION     = 502
    EC_ALREADY_LOGGED_IN    = 503
    EC_LOGIN_FAILED         = 504
    EC_UPS_UNPLUGGED        = 505
    //EC_PICKING_PIN_DETECTED = 503
)

var EventClasses = []int64 {
    EC_NA,
    EC_INFO,
    EC_OK,
    EC_ERROR,
    EC_LOST,
    EC_ALARM}
