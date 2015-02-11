package struc

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

func value(data interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		if v.Kind() == reflect.Struct {
			return v, nil
		}
	}
	return reflect.Value{}, fmt.Errorf("struc: got %s, expected pointer to struct", v.Kind().String())
}

func prep(data interface{}) (reflect.Value, Fields, error) {
	val, err := value(data)
	if err != nil {
		return reflect.Value{}, nil, err
	}
	fields, err := parseFields(val)
	return val, fields, err
}

func Pack(w io.Writer, data interface{}) error {
	val, fields, err := prep(data)
	if err != nil {
		return err
	}
	return fields.Pack(w, val)
}

// TODO: this is destructive with caching
func PackWithOrder(w io.Writer, data interface{}, order binary.ByteOrder) error {
	val, fields, err := prep(data)
	if err != nil {
		return err
	}
	fields.SetByteOrder(order)
	return fields.Pack(w, val)
}

func Unpack(r io.Reader, data interface{}) error {
	val, fields, err := prep(data)
	if err != nil {
		return err
	}
	return fields.Unpack(r, val)
}

func UnpackWithOrder(r io.Reader, data interface{}, order binary.ByteOrder) error {
	val, fields, err := prep(data)
	if err != nil {
		return err
	}
	fields.SetByteOrder(order)
	return fields.Unpack(r, val)
}

func Sizeof(data interface{}) (int, error) {
	val, fields, err := prep(data)
	if err != nil {
		return 0, err
	}
	return fields.Sizeof(val), nil
}
