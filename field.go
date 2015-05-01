package struc

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sync"
)

type Field struct {
	Name     string
	CanSet   bool
	Ptr      bool
	Index    int
	Type     Type
	Slice    bool
	Len      int
	Order    binary.ByteOrder
	Sizeof   []int
	Sizefrom []int
	kind     reflect.Kind
}

var bufferPool sync.Pool

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

func (f *Field) Size(val reflect.Value) int {
	if f.Type == Struct {
		fields, err := parseFields(val)
		if err == nil {
			return fields.Sizeof(val)
		}
		return 0
	} else if f.Type == Pad {
		return f.Len
	} else if f.Slice || f.kind == reflect.String {
		return val.Len() * f.Type.Size()
	} else {
		return f.Type.Size()
	}
}

func (f *Field) packVal(buf []byte, val reflect.Value, length int) error {
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
		return fields.Pack(buf, val)
	case Bool, Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64:
		var n uint64
		switch f.kind {
		case reflect.Bool:
			if val.Bool() {
				n = 1
			} else {
				n = 0
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
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
		case Int64, Uint64:
			order.PutUint64(buf, uint64(n))
		}
	case Float32, Float64:
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
			copy(buf, []byte(val.String()))
		default:
			// TODO: handle kind != bytes here
			copy(buf, val.Bytes())
		}
	}
	return nil
}

func (f *Field) safePack(buf []byte, val reflect.Value, length int) error {
	if f.Type == Pad {
		for i := 0; i < length; i++ {
			buf[i] = 0
		}
		return nil
	}
	if f.Slice {
		pos := 0
		for i := 0; i < length; i++ {
			if err := f.packVal(buf[pos:], val.Index(i), 1); err != nil {
				return err
			}
			pos += f.Type.Size()
		}
		return nil
	} else {
		return f.packVal(buf, val, length)
	}
}

func (f *Field) Pack(buf []byte, val reflect.Value, length int) (err error) {
	defer func() {
		if q := recover(); q != nil {
			err = f.safePack(buf, val, length)
		}
	}()

	if f.Type == Pad {
		for i := 0; i < length; i++ {
			buf[i] = 0
		}
		return nil
	}
	if f.Slice {
		// special case byte slices for performance
		if f.Type == Uint8 {
			copy(buf, val.Bytes()[:length])
			return nil
		}
		pos := 0
		for i := 0; i < length; i++ {
			if err := f.packVal(buf[pos:], val.Index(i), 1); err != nil {
				return err
			}
			pos += f.Type.Size()
		}
		return nil
	} else {
		return f.packVal(buf, val, length)
	}
}

func (f *Field) unpackVal(buf []byte, val reflect.Value, length int) error {
	order := f.Order
	if f.Ptr {
		val = val.Elem()
	}
	switch f.Type {
	case Float32, Float64:
		var n float64
		switch f.Type {
		case Float32:
			n = float64(math.Float32frombits(order.Uint32(buf)))
		case Float64:
			n = math.Float64frombits(order.Uint64(buf))
		}
		switch f.kind {
		case reflect.Float32, reflect.Float64:
			val.SetFloat(n)
		default:
			return fmt.Errorf("struc: refusing to unpack float into field %s of type %s", f.Name, f.kind.String())
		}
	case Bool, Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64:
		var n uint64
		switch f.Type {
		case Bool, Int8, Uint8:
			n = uint64(buf[0])
		case Int16, Uint16:
			n = uint64(order.Uint16(buf))
		case Int32, Uint32:
			n = uint64(order.Uint32(buf))
		case Int64, Uint64:
			n = uint64(order.Uint64(buf))
		}
		switch f.kind {
		case reflect.Bool:
			val.SetBool(n != 0)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(n))
		default:
			val.SetUint(n)
		}
	}
	return nil
}

func (f *Field) safeUnpack(buf []byte, val reflect.Value, length int) error {
	if f.Type == Pad || f.kind == reflect.String {
		if f.Type == Pad {
			return nil
		} else {
			val.SetString(string(buf))
			return nil
		}
	} else if f.Slice {
		target := val
		if val.Cap() < length {
			target = reflect.MakeSlice(val.Type(), length, length)
			val.Set(target)
		}
		pos := 0
		size := f.Type.Size()
		for i := 0; i < length; i++ {
			if err := f.unpackVal(buf[pos:pos+size], target.Index(i), 1); err != nil {
				return err
			}
			pos += size
		}
		return nil
	} else {
		return f.unpackVal(buf, val, length)
	}
}

func (f *Field) Unpack(buf []byte, val reflect.Value, length int) (err error) {
	defer func() {
		if q := recover(); q != nil {
			err = f.safeUnpack(buf, val, length)
		}
	}()
	val.Bytes()
	if f.Type == Pad || f.kind == reflect.String {
		if f.Type == Pad {
			return nil
		} else {
			val.SetString(string(buf))
			return nil
		}
	} else if f.Slice {
		target := val
		if val.Cap() < length {
			target = reflect.MakeSlice(val.Type(), length, length)
			val.Set(target)
		}
		// special case byte slices for performance
		if f.Type == Uint8 {
			newbuf := make([]byte, length)
			copy(newbuf, buf[:length])
			val.SetBytes(newbuf[:length])
			return nil
		}
		pos := 0
		size := f.Type.Size()
		for i := 0; i < length; i++ {
			if err := f.unpackVal(buf[pos:pos+size], target.Index(i), 1); err != nil {
				return err
			}
			pos += size
		}
		return nil
	} else {
		return f.unpackVal(buf, val, length)
	}
}

func init() {
	bufferPool.New = func() interface{} {
		return make([]byte, 65536)
	}
}
