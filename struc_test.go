package struc

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

type Nested struct {
	Test2 int
}

type Example struct {
	Pad    []byte `struc:"[5]pad"`        // 00 00 00 00 00
	I8f    int    `struc:"int8"`          // 01
	I16f   int    `struc:"int16"`         // 00 02
	I32f   int    `struc:"int32"`         // 00 00 00 03
	I64f   int    `struc:"int64"`         // 00 00 00 00 00 00 00 04
	U8f    int    `struc:"uint8,little"`  // 05
	U16f   int    `struc:"uint16,little"` // 06 00
	U32f   int    `struc:"uint32,little"` // 07 00 00 00
	U64f   int    `struc:"uint64,little"` // 08 00 00 00 00 00 00 00
	Boolf  int    `struc:"bool"`          // 01
	Byte4f []byte `struc:"[4]byte"`       // "abcd"

	I8    int8    // 09
	I16   int16   // 00 0a
	I32   int32   // 00 00 00 0b
	I64   int64   // 00 00 00 00 00 00 00 0c
	U8    uint8   `struc:"little"` // 0d
	U16   uint16  `struc:"little"` // 0e 00
	U32   uint32  `struc:"little"` // 0f 00 00 00
	U64   uint64  `struc:"little"` // 10 00 00 00 00 00 00 00
	Bool  bool    // 00
	Byte4 [4]byte // "efgh"

	Size int    `struc:"sizeof=Str,little"` // 0a 00 00 00
	Str  string // "ijklmnopqr"
	Strb string `struc:"[4]byte"` // stuv

	Nested  Nested  // 00 00 00 01
	NestedP *Nested // 00 00 00 02
	TestP64 *int    `struc:"int64"` // 00 00 00 05
}

var five = 5

var reference = &Example{
	nil,
	1, 2, 3, 4, 5, 6, 7, 8, 0, []byte{'a', 'b', 'c', 'd'},
	9, 10, 11, 12, 13, 14, 15, 16, true, [4]byte{'e', 'f', 'g', 'h'},
	10, "ijklmnopqr", "stuv",
	Nested{1}, &Nested{2}, &five,
}

var referenceBytes = []byte{
	0, 0, 0, 0, 0, // pad(5)
	1, 0, 2, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4, // fake int8-int64(1-4)
	5, 6, 0, 7, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, // fake little-endian uint8-uint64(5-8)
	0,                  // fake bool(0)
	'a', 'b', 'c', 'd', // fake [4]byte

	9, 0, 10, 0, 0, 0, 11, 0, 0, 0, 0, 0, 0, 0, 12, // real int8-int64(9-12)
	13, 14, 0, 15, 0, 0, 0, 16, 0, 0, 0, 0, 0, 0, 0, // real little-endian uint8-uint64(13-16)
	1,                  // real bool(1)
	'e', 'f', 'g', 'h', // real [4]byte

	10, 0, 0, 0, // little-endian int32(10) sizeof=Str
	'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', // Str
	's', 't', 'u', 'v', // fake string([4]byte)

	0, 0, 0, 1, // Nested{1}
	0, 0, 0, 2, // &Nested{2}
	0, 0, 0, 0, 0, 0, 0, 5, // &five
}

func TestCodec(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, reference); err != nil {
		t.Fatal(err)
	}
	out := &Example{}
	if err := Unpack(&buf, out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reference, out) {
		t.Fatal("encode/decode failed")
	}
}

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, reference); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), referenceBytes) {
		t.Fatal("encode failed")
	}
}

func TestDecode(t *testing.T) {
	buf := bytes.NewReader(referenceBytes)
	out := &Example{}
	if err := Unpack(buf, out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reference, out) {
		t.Fatal("decode failed")
	}
}

func TestSizeof(t *testing.T) {
	size, err := Sizeof(reference)
	if err != nil {
		t.Fatal(err)
	}
	if size != len(referenceBytes) {
		t.Fatal("sizeof failed")
	}
}

type ExampleEndian struct {
	T int `struc:"int16"`
}

func TestEndianSwap(t *testing.T) {
	var buf bytes.Buffer
	big := &ExampleEndian{1}
	if err := PackWithOrder(&buf, big, binary.BigEndian); err != nil {
		t.Fatal(err)
	}
	little := &ExampleEndian{}
	if err := UnpackWithOrder(&buf, little, binary.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if little.T != 256 {
		t.Fatal("big -> little conversion failed")
	}
}
