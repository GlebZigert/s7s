package rif
import "../../api"

type EventClass map[int64]int64

var evClasses = EventClass { // state classes
    // eventCode: eventClass
    0: api.EC_NA,
    1: api.EC_OK,
    2: api.EC_NA, // lock-na
    10: api.EC_LOST,
    11: api.EC_ERROR,
    12: api.EC_ERROR,
    13: api.EC_ERROR,
    14: api.EC_ERROR,
    17: api.EC_LOST, // lock-lost
    18: api.EC_ERROR,
    20: api.EC_ALARM,
    21: api.EC_ALARM,
    22: api.EC_ALARM,
    23: api.EC_ALARM,
    25: api.EC_ALARM,
    100: api.EC_OK,      // Выключено
    101: api.EC_ALARM,   // Включено
    110: api.EC_OK,      // Закрыто
    111: api.EC_ALARM,   // Открыто
    112: api.EC_OK,      // Закрыто ключом
    113: api.EC_ALARM,   // Открыто ключом
    136: api.EC_NA,      // Контроль выкл
    1136: api.EC_NA,     // Удал.Ком. Контроль выкл
    143: api.EC_ALARM,   // Исходящий вызов
    1143: api.EC_ALARM,   // Удал. ком. Исходящий вызов
    144: api.EC_OK,     // Вызов завершен по кан. связи
    145: api.EC_ALARM,   // Входящий вызов
    //1145: api.EC_ALARM,   // Удал. ком. Входящий вызов
    146: api.EC_OK,       // Вызов завершён операторам
    1146: api.EC_OK}       // Удал. ком. Вызов завершён операторам
    //130: //Послана ком. Вкл
    //131: //Послана ком. Выкл
    //137: api.EC_NA, // Контроль вкл
    //1137: api.EC_NA, // Удал.Ком. Контроль вкл
    //150: api.EC_NA, // Послана команда «Открыть» УЗ
    //151: api.EC_NA, // Послана команда «Закрыть»


var evClassesOverride = map[int] EventClass{
    //deviceType: {eventCode: eventClass}
    1: {100: api.EC_INFO, 101: api.EC_INFO},
    11: {100: api.EC_INFO, 101: api.EC_INFO},
    99: {100: api.EC_INFO, 101: api.EC_INFO}}

// event class name (type)
func getClassCode(evCode int64, devType int) (code int64) {
    code = -1
    _, ok := evClassesOverride[devType]
    if ok {
        c, o := evClassesOverride[devType][evCode]
        if o {
            code = c
        }
    }
    if code < 0 {
        code, _ = evClasses[evCode]
    }

    return
}
