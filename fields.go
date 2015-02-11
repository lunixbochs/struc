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

func (f Fields) Sizeof(val reflect.Value) int {
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

func (f Fields) Pack(w io.Writer, val reflect.Value) error {
	for i, field := range f {
		if !field.CanSet {
			continue
		}
		v := val.Field(i)
		length := field.Len
		if field.Slice && field.CanSet && field.Type != Pad {
			length = v.Len()
		} else if field.Sizefrom != nil {
			length = int(val.FieldByIndex(field.Sizefrom).Int())
		}
		if field.Sizeof != nil {
			length := val.FieldByIndex(field.Sizeof).Len()
			v.SetInt(int64(length))
		}
		err := field.Pack(w, v, length)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f Fields) Unpack(r io.Reader, val reflect.Value) error {
	for i, field := range f {
		if !field.CanSet {
			continue
		}
		v := val.Field(i)
		length := field.Len
		if field.Sizefrom != nil {
			length = int(val.FieldByIndex(field.Sizefrom).Int())
		}
		err := field.Unpack(r, v, length)
		if err != nil {
			return err
		}
	}
	return nil
}
