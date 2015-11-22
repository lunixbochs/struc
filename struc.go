package struc

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

func prep(data interface{}) (reflect.Value, Packable, error) {
	value := reflect.ValueOf(data)
	for value.Kind() == reflect.Ptr {
		next := value.Elem().Kind()
		if next == reflect.Struct || next == reflect.Ptr {
			value = value.Elem()
		} else {
			break
		}
	}
	switch value.Kind() {
	case reflect.Struct:
		fields, err := parseFields(value)
		return value, fields, err
	default:
		if !value.IsValid() {
			return reflect.Value{}, nil, fmt.Errorf("Invalid reflect.Value for %+v", data)
		}
		return value, &binaryFallback{value, binary.BigEndian}, nil
	}
}

func Pack(w io.Writer, data interface{}) error {
	return PackWithOrder(w, data, nil)
}

// TODO: this is destructive with caching
func PackWithOrder(w io.Writer, data interface{}, order binary.ByteOrder) error {
	val, fields, err := prep(data)
	if err != nil {
		return err
	}
	if order != nil {
		fields.SetByteOrder(order)
	}
	if val.Type().Kind() == reflect.String {
		val = val.Convert(reflect.TypeOf([]byte{}))
	}
	size := fields.Sizeof(val)
	buf := make([]byte, size)
	if _, err := fields.Pack(buf, val); err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

func Unpack(r io.Reader, data interface{}) error {
	return UnpackWithOrder(r, data, nil)
}

func UnpackWithOrder(r io.Reader, data interface{}, order binary.ByteOrder) error {
	val, fields, err := prep(data)
	if err != nil {
		return err
	}
	if order != nil {
		fields.SetByteOrder(order)
	}
	return fields.Unpack(r, val)
}

func Sizeof(data interface{}) (int, error) {
	val, fields, err := prep(data)
	if err != nil {
		return 0, err
	}
	return fields.Sizeof(val), nil
}
