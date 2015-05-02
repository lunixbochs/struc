package struc

import (
	"bytes"
	"testing"
)

type badFloat struct {
	BadFloat int `struc:"float64"`
}

func TestBadFloatField(t *testing.T) {
	buf := bytes.NewReader([]byte("00000000"))
	err := Unpack(buf, &badFloat{})
	if err == nil {
		t.Fatal("failed to error on bad float unpack")
	}
}

type emptyLengthField struct {
	Strlen int `struc:"sizeof=Str"`
	Str    []byte
}

func TestEmptyLengthField(t *testing.T) {
	var buf bytes.Buffer
	s := &emptyLengthField{0, []byte("test")}
	o := &emptyLengthField{}
	Pack(&buf, s)
	Unpack(&buf, o)
	if !bytes.Equal(s.Str, o.Str) {
		t.Fatal("empty length field encode failed")
	}
}

type fixedSlicePad struct {
	Field []byte `struc:"[4]byte"`
}

func TestFixedSlicePad(t *testing.T) {
	var buf bytes.Buffer
	ref := []byte{0, 0, 0, 0}
	s := &fixedSlicePad{}
	Pack(&buf, s)
	if !bytes.Equal(buf.Bytes(), ref) {
		t.Fatal("implicit fixed slice pack failed")
	}
	Unpack(&buf, s)
	if !bytes.Equal(s.Field, ref) {
		t.Fatal("implicit fixed slice unpack failed")
	}
}
