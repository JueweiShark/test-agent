package encoding_json

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

type customPbjson struct {
	marshaler   jsonpb.Marshaler
	unmarshaler jsonpb.Unmarshaler
}

func init() {
	encoding.RegisterCodec(NewCustomPbjson())
}
func NewCustomPbjson() customPbjson {
	return customPbjson{
		marshaler: jsonpb.Marshaler{
			EmitDefaults: false,
		},
		unmarshaler: jsonpb.Unmarshaler{},
	}
}

func (marsh customPbjson) Name() string {
	return "customPbjson"
}

func (marsh customPbjson) Unmarshal(data []byte, v interface{}) error {
	switch m := v.(type) {
	case proto.Message:
		return marsh.unmarshaler.Unmarshal(bytes.NewReader(data), m)
	case json.Unmarshaler:
		return m.UnmarshalJSON(data)
	default:
		rv := reflect.ValueOf(v)
		for rv := rv; rv.Kind() == reflect.Ptr; {
			if rv.IsNil() {
				rv.Set(reflect.New(rv.Type().Elem()))
			}
			rv = rv.Elem()
		}
		if m, ok := reflect.Indirect(rv).Interface().(proto.Message); ok {
			return marsh.unmarshaler.Unmarshal(bytes.NewReader(data), m)
		}
		return json.Unmarshal(data, m)
	}
}

func (marsh customPbjson) Marshal(v interface{}) ([]byte, error) {
	switch m := v.(type) {
	case proto.Message:
		str, err := marsh.marshaler.MarshalToString(m)
		if err != nil {
			return nil, err
		}
		return []byte(str), nil
	case json.Marshaler:
		return m.MarshalJSON()
	default:
		return json.Marshal(m)
	}
}
