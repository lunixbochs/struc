package struc

import (
	"encoding/binary"
	"unsafe"
)

const (
	Pad = iota
	Bool
	Char
	Int8
	Uint8
	Int16
	Uint16
	Int32
	Uint32
	Int64
	Uint64
	Float32
	Float64
	String
	PascalString
)

var typeLookup = map[byte]int{
	'x': Pad,
	'?': Bool,
	'c': Char,
	'b': Int8,
	'B': Uint8,
	'h': Int16,
	'H': Uint16,
	'i': Int32,
	'I': Uint32,
	'q': Int64,
	'Q': Uint64,
	'f': Float32,
	'd': Float64,
	's': String,
	'p': PascalString,
}

var typeRevLookup = map[int]byte{
	Pad:          'x',
	Bool:         '?',
	Char:         'c',
	Int8:         'b',
	Uint8:        'B',
	Int16:        'h',
	Uint16:       'H',
	Int32:        'i',
	Uint32:       'I',
	Int64:        'q',
	Uint64:       'Q',
	Float32:      'f',
	Float64:      'd',
	String:       's',
	PascalString: 'p',
}

var typeNames = map[int]string{
	Pad:          "Pad",
	Bool:         "Bool",
	Char:         "Char",
	Int8:         "Int8",
	Uint8:        "Uint8",
	Int16:        "Int16",
	Uint16:       "Uint16",
	Int32:        "Int32",
	Uint32:       "Uint32",
	Int64:        "Int64",
	Uint64:       "Uint64",
	Float32:      "Float32",
	Float64:      "Float64",
	String:       "String",
	PascalString: "PascalString",
}

const (
	Native = iota
	Big
	Little
)

var orderLookup = map[string]int{
	"native": Native,
	"big":    Big,
	"little": Little,
}

var orderNames = map[int]string{
	Native: "native",
	Big:    "big",
	Little: "little",
}

func (f *Field) Size() int {
	size := 0
	switch f.Type {
	case Pad, Char, Int8, Uint8, Bool:
		size = 1
	case Int16, Uint16:
		size = 2
	case Int32, Uint32, Float32:
		size = 4
	case Int64, Uint64, Float64:
		size = 8
	case String, PascalString:
		size = 1 * f.Len
	}
	return size
}

func getByteEncoder(order int) binary.ByteOrder {
	if order == Native {
		var i int16 = 0x0102
		if *(*byte)(unsafe.Pointer(&i)) == 2 {
			order = Little
		} else {
			order = Big
		}
	}
	switch order {
	case Big:
		return binary.BigEndian
	case Little:
		return binary.LittleEndian
	default:
		panic("Invalid byte order")
	}
}
