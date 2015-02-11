package struc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

type BenchExample struct {
	Test    [5]byte
	A       int32
	B, C, D int16
	Test2   [4]byte
	Length  int32
}

type BenchStrucExample struct {
	Test    [5]byte `[5]byte`
	A       int     `int32`
	B, C, D int     `int16`
	Test2   [4]byte `[4]byte`
	Length  int     `sizeof:"Data" int32`
	Data    []byte
}

var benchRef = &BenchExample{
	[5]byte{1, 2, 3, 4, 5},
	1, 2, 3, 4,
	[4]byte{1, 2, 3, 4},
	8,
}

var eightBytes = []byte("8bytestr")

var benchStrucRef = &BenchStrucExample{
	[5]byte{1, 2, 3, 4, 5},
	1, 2, 3, 4,
	[4]byte{1, 2, 3, 4},
	0, eightBytes,
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := Pack(&buf, benchStrucRef)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlibEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := binary.Write(&buf, binary.BigEndian, benchRef)
		if err != nil {
			b.Fatal(err)
		}
		_, err = buf.Write(eightBytes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	var out BenchExample
	var buf bytes.Buffer
	Pack(&buf, benchStrucRef)
	bufBytes := buf.Bytes()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewReader(bufBytes)
		err := Unpack(buf, &out)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlibDecode(b *testing.B) {
	var out BenchExample
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, *benchRef)
	_, err := buf.Write(eightBytes)
	if err != nil {
		b.Fatal(err)
	}
	bufBytes := buf.Bytes()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewReader(bufBytes)
		err := binary.Read(buf, binary.BigEndian, &out)
		if err != nil {
			b.Fatal(err)
		}
		tmp := make([]byte, out.Length)
		_, err = buf.Read(tmp)
		if err != nil {
			b.Fatal(err)
		}
	}
}
