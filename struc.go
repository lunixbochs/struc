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
	size := fields.Sizeof(val)
	var orgbuf []byte
	var buf []byte
	if size <= 65536 {
		orgbuf = bufferPool.Get().([]byte)
		buf = orgbuf[:size]
	} else {
		buf = make([]byte, size)
	}
	if err := fields.Pack(buf, val); err != nil {
		return err
	}
	_, err = w.Write(buf)
	if size <= 65536 {
		bufferPool.Put(orgbuf)
	}
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
