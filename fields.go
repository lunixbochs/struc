package struc

import (
	"io"
	"reflect"
	"strings"
)

type Fields []*Field

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
		case PascalString, String:
			size += val.Field(field.Index).Len()
		default:
			size += field.Size()
		}
	}
	return size
}

func (f Fields) Pack(w io.Writer, data interface{}) error {
	val := reflect.ValueOf(data).Elem()
	for _, field := range f {
		i := field.Index
		if field.Sizeof != "" {
			length := val.FieldByName(field.Sizeof).Len()
			val.Field(i).SetInt(int64(length))
		}
		var v reflect.Value
		if i >= 0 {
			v = val.Field(i)
		}
		err := field.Pack(w, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f Fields) Unpack(r io.Reader, data interface{}) error {
	val := reflect.ValueOf(data).Elem()
	for _, field := range f {
		i := field.Index
		if field.Sizefrom > -1 {
			field.Len = int(val.Field(field.Sizefrom).Int())
		}
		var v reflect.Value
		if i >= 0 {
			v = val.Field(i)
		}
		err := field.Unpack(r, v)
		if err != nil {
			return err
		}
	}
	return nil
}
