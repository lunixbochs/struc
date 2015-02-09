package struc

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
)

type Field struct {
	Index    int
	Type     int
	Len      int
	Order    int
	Sizeof   string
	Sizefrom int
}

func (f *Field) String() string {
	var out string
	if f.Type == Pad {
		return fmt.Sprintf("{type: Pad, len: %d}", f.Len)
	} else {
		order := orderNames[f.Order]
		typeName := typeNames[f.Type]
		out = fmt.Sprintf("%d, type: %s, order: %s", f.Index, typeName, order)
	}
	if f.Sizefrom > -1 {
		out += fmt.Sprintf(", sizefrom: %d", f.Sizefrom)
	} else if f.Len > 0 {
		out += fmt.Sprintf(", len: %d", f.Len)
	}
	if f.Sizeof != "" {
		out += fmt.Sprintf(", sizeof: %s", f.Sizeof)
	}
	return "{" + out + "}"
}

func (f *Field) Pack(w io.Writer, val reflect.Value) error {
	var buf []byte
	order := getByteEncoder(f.Order)
	switch f.Type {
	case Bool, Char, Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64:
		buf = make([]byte, f.Size())
		n := val.Int()
		switch f.Type {
		case Bool:
			if n != 0 {
				buf[0] = 1
			} else {
				buf[0] = 0
			}
		case Char, Int8, Uint8:
			buf[0] = byte(n)
		case Int16, Uint16:
			order.PutUint16(buf, uint16(n))
		case Int32, Uint32:
			order.PutUint32(buf, uint32(n))
		case Int64, Uint64:
			order.PutUint64(buf, uint64(n))
		}
	case Float32, Float64:
		buf = make([]byte, f.Size())
		n := val.Float()
		switch f.Type {
		case Float32:
			order.PutUint32(buf, math.Float32bits(float32(n)))
		case Float64:
			order.PutUint64(buf, math.Float64bits(n))
		}
	case Pad:
		buf = bytes.Repeat([]byte{0}, f.Len)
	case String, PascalString:
		switch val.Kind() {
		case reflect.String:
			buf = []byte(val.String())
		default:
			// TODO: catch the panic here and turn it into an error?
			buf = val.Bytes()
		}
		if f.Type == PascalString {
			if len(buf) > 255 {
				return fmt.Errorf("struc: buffer size %d too long for pascal string")
			}
			buf = append([]byte{byte(len(buf))}, buf...)
		}
	}
	_, err := w.Write(buf)
	return err
}

func (f *Field) Unpack(r io.Reader, val reflect.Value) error {
	order := getByteEncoder(f.Order)
	switch f.Type {
	case Pad, String, PascalString:
		if f.Type == PascalString {
			length := []byte{0}
			if _, err := io.ReadFull(r, length); err != nil {
				return err
			}
			f.Len = int(length[0])
		}
		buf := make([]byte, f.Len)
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return err
		}
		if val.Kind() == reflect.String {
			val.SetString(string(buf))
		} else if val.IsValid() {
			// TODO: catch the panic and convert to error here?
			val.SetBytes(buf)
		}
	case Bool, Char, Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64, Float32, Float64:
		buf := make([]byte, f.Size())
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return err
		}
		switch f.Type {
		case Char, Int8, Uint8:
			val.SetInt(int64(buf[0]))
		case Int16, Uint16:
			val.SetInt(int64(order.Uint16(buf)))
		case Int32, Uint32:
			val.SetInt(int64(order.Uint32(buf)))
		case Int64, Uint64:
			val.SetInt(int64(order.Uint64(buf)))
		}
	}
	return nil
}
