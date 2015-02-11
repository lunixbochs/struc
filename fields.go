package struc

import (
	"encoding/binary"
	"io"
	"reflect"
	"strings"
)

type Fields []*Field

func (f Fields) SetByteOrder(order binary.ByteOrder) {
	for _, field := range f {
		field.Order = order
	}
}

func (f Fields) String() string {
	fields := make([]string, len(f))
	for i, field := range f {
		fields[i] = field.String()
	}
	return "{" + strings.Join(fields, ", ") + "}"
}

func (f Fields) Sizeof(data interface{}) int {
	val := reflect.ValueOf(data).Elem()
	size := 0
	for _, field := range f {
		switch field.Type {
		case String:
			size += val.Field(field.Index).Len()
		default:
			size += field.Size()
		}
	}
	return size
}

func (f Fields) Pack(w io.Writer, data interface{}) error {
	val := reflect.ValueOf(data).Elem()
	for i, field := range f {
		v := val.Field(i)
		length := field.Len
		if field.Slice && v.CanSet() {
			length = v.Len()
		} else if field.Sizefrom > -1 {
			length = int(val.Field(field.Sizefrom).Int())
		}
		if field.Sizeof != "" {
			length := val.FieldByName(field.Sizeof).Len()
			v.SetInt(int64(length))
		}
		err := field.Pack(w, v, length)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f Fields) Unpack(r io.Reader, data interface{}) error {
	val := reflect.ValueOf(data).Elem()
	for i, field := range f {
		v := val.Field(i)
		length := field.Len
		if field.Sizefrom > -1 {
			length = int(val.Field(field.Sizefrom).Int())
		}
		err := field.Unpack(r, v, length)
		if err != nil {
			return err
		}
	}
	return nil
}
