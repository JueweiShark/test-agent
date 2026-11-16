package schema

import (
    "database/sql/driver"
    "encoding/json"
    "errors"
    "fmt"
)

type JsonObject map[string]interface{}

func (jsonObject *JsonObject) Scan(value interface{}) error {

    switch v := value.(type) {
    case []byte:
        err := json.Unmarshal(v, jsonObject)
        if err != nil {
            return err
        }
    }
    return nil
}

func (jsonObject JsonObject) Value() (driver.Value, error) {
    return json.Marshal(jsonObject)
}

type Int64Array []int64

func (ia *Int64Array) Scan(value interface{}) error {
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("type assertion to []byte failed")
    }

    return json.Unmarshal(bytes, ia)
}

func (ia Int64Array) Value() (driver.Value, error) {
    bytes, err := json.Marshal(ia)
    if err != nil {
        return nil, err
    }
    return bytes, nil
}

type StringArray []int64

func (ia *StringArray) Scan(value interface{}) error {
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("type assertion to []byte failed")
    }

    return json.Unmarshal(bytes, ia)
}

func (ia StringArray) Value() (driver.Value, error) {
    bytes, err := json.Marshal(ia)
    if err != nil {
        return nil, err
    }
    return bytes, nil
}

type BitBool bool

func (b *BitBool) Scan(value interface{}) error {
    switch v := value.(type) {
    case []byte:
        *b = string(v) == "\x01"
    case int64:
        *b = v == 1
    case float64:
        *b = v == 1
    default:
        *b = false
    }
    return nil
}

func (b BitBool) Value() (driver.Value, error) {
    if b {
        return "\x01", nil
    }
    return "\x00", nil
}

type TinyIntBool bool

func (b *TinyIntBool) Scan(value interface{}) error {
    var i int64
    switch v := value.(type) {
    case int64:
        i = v
    case float64:
        i = int64(v)

    default:
        return fmt.Errorf("unsupported Scan type %T", value)
    }
    *b = i != 0
    return nil
}

func (b TinyIntBool) Value() (driver.Value, error) {
    if b {
        return 1, nil
    }
    return 0, nil
}
