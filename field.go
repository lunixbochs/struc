package struc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
)

type Field struct {
	Name     string
	CanSet   bool
	Struct   bool
	Ptr      bool
	Index    int
	Type     int
	Slice    bool
	Len      int
	Order    binary.ByteOrder
	Sizeof   []int
	Sizefrom []int
	// our offset in the struct, from reflect.StructField.Offset
	offset uintptr
	kind   reflect.Kind
}

func (f *Field) String() string {
	var out string
	if f.Type == Pad {
		return fmt.Sprintf("{type: Pad, len: %d}", f.Len)
	} else {
		typeName := typeNames[f.Type]
		out = fmt.Sprintf("type: %s, order: %v", typeName, f.Order)
	}
	if f.Sizefrom != nil {
		out += fmt.Sprintf(", sizefrom: %v", f.Sizefrom)
	} else if f.Len > 0 {
		out += fmt.Sprintf(", len: %d", f.Len)
	}
	if f.Sizeof != nil {
		out += fmt.Sprintf(", sizeof: %v", f.Sizeof)
	}
	return "{" + out + "}"
}

func (f *Field) packVal(w io.Writer, val reflect.Value, length int) error {
	var buf []byte
	order := f.Order
	if f.Ptr {
		val = val.Elem()
	}
	switch f.Type {
	case Struct:
		fields, err := parseFields(val)
		if err != nil {
			return err
		}
		return fields.Pack(w, val)
	case Bool, Int8, Int16, Int32, Uint8, Uint16, Uint32:
		var n uint64
		var tmp [4]byte
		buf = tmp[:f.Size()]
		switch f.kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			n = uint64(val.Int())
		default:
			n = val.Uint()
		}
		switch f.Type {
		case Bool:
			if n != 0 {
				buf[0] = 1
			} else {
				buf[0] = 0
			}
		case Int8, Uint8:
			buf[0] = byte(n)
		case Int16, Uint16:
			order.PutUint16(buf, uint16(n))
		case Int32, Uint32:
			order.PutUint32(buf, uint32(n))
		}
	case Int64, Uint64:
		var n uint64
		var tmp [8]byte
		buf = tmp[:f.Size()]
		if f.kind == reflect.Int64 {
			n = uint64(val.Int())
		} else {
			n = val.Uint()
		}
		order.PutUint64(buf, uint64(n))
	case Float32, Float64:
		var tmp [8]byte
		buf = tmp[:f.Size()]
		n := val.Float()
		switch f.Type {
		case Float32:
			order.PutUint32(buf, math.Float32bits(float32(n)))
		case Float64:
			order.PutUint64(buf, math.Float64bits(n))
		}
	case String:
		switch f.kind {
		case reflect.String:
			buf = []byte(val.String())
		default:
			// TODO: handle kind != bytes here
			buf = val.Bytes()
		}
	}
	_, err := w.Write(buf)
	return err
}

func (f *Field) Pack(w io.Writer, val reflect.Value, length int) error {
	if f.Type == Pad {
		_, err := w.Write(make([]byte, length))
		return err
	}
	if f.Slice {
		for i := 0; i < length; i++ {
			if err := f.packVal(w, val.Index(i), 1); err != nil {
				return err
			}
		}
		return nil
	} else {
		return f.packVal(w, val, length)
	}
}

func (f *Field) unpackVal(r io.Reader, val reflect.Value, length int) error {
	order := f.Order
	if f.Ptr {
		val = val.Elem()
	}
	switch f.Type {
	case Struct:
		fields, err := parseFields(val)
		if err != nil {
			return err
		}
		return fields.Unpack(r, val)
	case Bool, Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64, Float32, Float64:
		var tmp [8]byte
		buf := tmp[:f.Size()]
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return err
		}
		var n uint64
		switch f.Type {
		case Int8, Uint8:
			n = uint64(buf[0])
		case Int16, Uint16:
			n = uint64(order.Uint16(buf))
		case Int32, Uint32:
			n = uint64(order.Uint32(buf))
		case Int64, Uint64:
			n = uint64(order.Uint64(buf))
		}
		switch f.kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(n))
		default:
			val.SetUint(n)
		}
	}
	return nil
}

func (f *Field) Unpack(r io.Reader, val reflect.Value, length int) error {
	if f.Type == Pad || f.kind == reflect.String {
		buf := make([]byte, length)
		if f.Type == Pad {
			_, err := r.Read(buf)
			return err
		} else {
			_, err := r.Read(buf)
			val.SetString(string(buf))
			return err
		}
	} else if f.Slice {
		target := val
		if val.Cap() < length {
			target = reflect.MakeSlice(val.Type(), length, length)
			val.Set(target)
		}
		for i := 0; i < length; i++ {
			if err := f.unpackVal(r, target.Index(i), 1); err != nil {
				return err
			}
		}
		return nil
	} else {
		return f.unpackVal(r, val, length)
	}
}
