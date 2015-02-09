package struc

import (
	"bytes"
	"reflect"
	"testing"
)

type Example struct {
	A       int `pack:"5x i 5h s" order:"big"`
	B, C, D int
	Size    int `sizeof:"Str" order:"little"`
	E       int `order:"big"`
	Str     []byte
}

var reference = &Example{1, 2, 3, 4, 5, 0, []byte("asdfasdf")}

var referenceBytes = []byte{
	0, 0, 0, 0, 0, // pad(5)
	0, 0, 0, 1, // int32(1) - big
	0, 2, // int16(2) - big
	0, 3, // int16(3) - big
	0, 4, // int16(4) - big
	8, 0, // int16(8) - little (sizeof str)
	0, 0, // int16(0) - big
	97, 115, 100, 102, 97, 115, 100, 102, // str (length 8)
}

func TestCodec(t *testing.T) {
	var buf bytes.Buffer
	err := Pack(&buf, reference)
	if err != nil {
		t.Fatal(err)
	}
	out := &Example{}
	err = Unpack(&buf, out)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reference, out) {
		t.Fatal("encode/decode failed")
	}
}

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	err := Pack(&buf, reference)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), referenceBytes) {
		t.Fatal("encode failed")
	}
}

func TestDecode(t *testing.T) {
	buf := bytes.NewReader(referenceBytes)
	out := &Example{}
	err := Unpack(buf, out)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reference, out) {
		t.Fatal("decode failed")
	}
}
