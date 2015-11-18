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
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	size := 0
	for i, field := range f {
		v := val.Field(i)
		if v.CanSet() {
			size += field.Size(v)
		}
	}
	return size
}

func (f Fields) Pack(buf []byte, val reflect.Value) error {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	pos := 0
	for i, field := range f {
		if !field.CanSet {
			continue
		}
		v := val.Field(i)
		length := field.Len
		if field.Sizefrom != nil {
			length = int(val.FieldByIndex(field.Sizefrom).Int())
		}
		if length <= 0 && field.Slice {
			length = v.Len()
		}
		if field.Sizeof != nil {
			length := val.FieldByIndex(field.Sizeof).Len()
			v = reflect.ValueOf(length)
		}
		err := field.Pack(buf[pos:], v, length)
		if err != nil {
			return err
		}
		pos += field.Size(v)
	}
	return nil
}

func (f Fields) Unpack(r io.Reader, val reflect.Value) error {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	var tmp [8]byte
	var buf []byte
	for i, field := range f {
		if !field.CanSet {
			continue
		}
		v := val.Field(i)
		length := field.Len
		if field.Sizefrom != nil {
			length = int(val.FieldByIndex(field.Sizefrom).Int())
		}
		if v.Kind() == reflect.Ptr && !v.Elem().IsValid() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if field.Type == Struct {
			if field.Slice {
				vals := reflect.MakeSlice(v.Type(), length, length)
				for i := 0; i < length; i++ {
					v := vals.Index(i)
					fields, err := parseFields(v)
					if err != nil {
						return err
					}
					if err := fields.Unpack(r, v); err != nil {
						return err
					}
				}
				v.Set(vals)
			} else {
				// TODO: DRY (we repeat the inner loop above)
				fields, err := parseFields(v)
				if err != nil {
					return err
				}
				if err := fields.Unpack(r, v); err != nil {
					return err
				}
			}
			continue
		} else {
			size := length * field.Type.Size()
			if size < 8 {
				buf = tmp[:size]
			} else {
				buf = make([]byte, size)
			}
			if _, err := io.ReadFull(r, buf); err != nil {
				return err
			}
			err := field.Unpack(buf[:size], v, length)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
