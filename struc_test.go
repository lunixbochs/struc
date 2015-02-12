package struc

import (
	"bytes"
	"reflect"
	"testing"
)

type Nested struct {
	Test2 int
}

type Example struct {
	Pad     []byte `struc:"[5]pad"`
	A       int    `struc:"int32"`
	B, C, D int    `struc:"uint16"`
	Size    int    `struc:"sizeof=Str,little"`
	Str     string
	Test    []byte `struc:"[4]byte"`
	Nested  Nested
	NestedP *Nested
}

var reference = &Example{
	nil,
	1,
	2, 3, 4,
	8,
	"asdfasdf",
	[]byte("1234"),
	Nested{1},
	&Nested{2},
}

var referenceBytes = []byte{
	0, 0, 0, 0, 0, // pad(5)
	0, 0, 0, 1, // int32(1) - big
	0, 2, // int16(2) - big
	0, 3, // int16(3) - big
	0, 4, // int16(4) - big
	8, 0, 0, 0, // int32(8) - sizeof=Str, little
	97, 115, 100, 102, 97, 115, 100, 102, // str (length 8)
	49, 50, 51, 52, // [4]byte
	0, 0, 0, 1, // Nested{1} (int)
	0, 0, 0, 2, // *Nested{2} (int)
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
