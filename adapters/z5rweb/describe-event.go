package z5rweb

import (
    "strings"
    "strconv"
)

import "s7server/api"

const (
    EID_DEVICE_ONLINE = 101
    EID_DEVICE_OFFLINE = 102
    EID_DEVICE_ERROR = 103
    EID_EVENTS_LOADED = 104
)


var evTypes = map[int64] struct {Class int64; Reader int; Text string} {
    0: {api.EC_INFO, 1, "Открыто кнопкой изнутри"},
    1: {api.EC_INFO, 2, "Открыто кнопкой изнутри"},
    2: {api.EC_INFO, 1, "Ключ не найден в банке ключей"}, // TODO: ALARM
    3: {api.EC_INFO, 2, "Ключ не найден в банке ключей"},
    4: {api.EC_OK, 1, "Ключ найден, дверь открыта"},
    5: {api.EC_OK, 2, "Ключ найден, дверь открыта"},
    6: {api.EC_INFO, 1, "Ключ найден, доступ запрещён"},
    7: {api.EC_INFO, 2, "Ключ найден, доступ запрещён"},
    8: {api.EC_INFO, 1, "Открыто оператором по сети"}, // !!!!!!!!!!!!!!!!!!!!!!!
    9: {api.EC_INFO, 2, "Открыто оператором по сети"}, // !!!!!!!!!!!!!!!!!!!!!!!
    10: {api.EC_INFO, 1, "Ключ найден, дверь заблокирована"},
    11: {api.EC_INFO, 2, "Ключ найден, дверь заблокирована"},
    12: {api.EC_ALARM, 1, "Дверь взломана"},
    13: {api.EC_ALARM, 2, "Дверь взломана"},
    14: {api.EC_ALARM, 1, "Дверь оставлена открытой, время вышло"},
    15: {api.EC_ALARM, 2, "Дверь оставлена открытой, время вышло"},
    16: {api.EC_ENTER_ZONE, 1, "Проход состоялся"}, // swapped with #17
    17: {api.EC_ENTER_ZONE, 2, "Проход состоялся"}, // swapped with #16
    20: {api.EC_INFO, 0, "Перезагрузка контроллера"},
    21: {api.EC_INFO, 0, "Питание "}, //?
    32: {api.EC_INFO, 1, "Дверь открыта"},
    33: {api.EC_INFO, 2, "Дверь открыта"},
    34: {api.EC_OK, 1, "Дверь закрыта"},
    35: {api.EC_OK, 2, "Дверь закрыта"},
    37: {api.EC_INFO, 0, "Переключение режимов работы"}, // ignore and replace
    38: {api.EC_INFO, 0, "Пожарные события"},
    39: {api.EC_INFO, 0, "Охранные события"},
    40: {api.EC_INFO, 1, "Проход не совершен за заданное время"},
    41: {api.EC_INFO, 2, "Проход не совершен за заданное время"},
    48: {api.EC_INFO, 1, "Совершен вход в шлюз"},
    49: {api.EC_INFO, 2, "Совершен вход в шлюз"},
    50: {api.EC_INFO, 1, "Заблокирован вход, шлюз занят"},
    51: {api.EC_INFO, 2, "Заблокирован вход, шлюз занят"},
    52: {api.EC_INFO, 1, "Разрешен вход в шлюз"},
    53: {api.EC_INFO, 2, "Разрешен вход в шлюз"},
    54: {api.EC_INFO, 1, "Заблокирован проход (антипассбек)"},
    55: {api.EC_INFO, 2, "Заблокирован проход (антипассбек)"},

    // custom events
    62: {api.EC_INFO, 1, "Код не соответствует карте"},
    63: {api.EC_INFO, 2, "Код не соответствует карте"},
    
    64: {api.EC_ALARM, 1, "Попытка подбора кода"},
    65: {api.EC_ALARM, 2, "Попытка подбора кода"},

    66: {api.EC_INFO, 1, "Превышено количество посетителей"},
    67: {api.EC_INFO, 2, "Превышено количество посетителей"},

    68: {api.EC_INFO, 1, "Точка доступа заблокирована"},
    69: {api.EC_INFO, 2, "Точка доступа заблокирована"},
    
    // service events
    100: {api.EC_INFO, 0, "Неизвестное состояние"},
    EID_DEVICE_ONLINE: {api.EC_ONLINE, 0, "Соединение установлено"},
    EID_DEVICE_OFFLINE: {api.EC_LOST, 0, "Связь потеряна"},
    EID_DEVICE_ERROR: {api.EC_ERROR, 0, "Обнаружен сбой в работе устройства"},
    EID_EVENTS_LOADED: {api.EC_INFO, 0, "Переданы события из памяти"},
}


/*func eventClass(code int64) (class int64){
    info, ok := evTypes[code]
    if ok {
        class = info.Class
    } else {
        class = api.EC_NA
    }
    return
}*/

func getReader(code int64) int64 {
    info, _ := evTypes[code]
    return int64(info.Reader)
}

func formatCard(card string) string {
    key, _ := strconv.ParseInt(card, 16, 64)
    n := int(key & 0xFFFFFF)
    return strconv.Itoa(n >> 16) + "," + strconv.Itoa(n & 0xFFFF)
}

func describeEvent(event *Event) (txt string, cls int64) {
    info, ok := evTypes[int64(event.Event)]
    if ok {
        txt = info.Text
        cls = info.Class
        if 21 == event.Event {
            if 0 == event.Flag {
                txt += "пропало"
            } else {
                txt += "появилось"
            }
        }
        /*if 1 == info.Reader {
            txt = "Вход: " + txt
        } else if 2 == info.Reader {
            txt = "Выход: " + txt
        }*/
    } else {
        txt = "Неизвестное состояние"
        cls = api.EC_INFO
    }
    card := strings.TrimLeft(event.Card, "0")
    if "" != card {
        txt += " (" + formatCard(card) + ")"
    }
    return
}