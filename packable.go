package struc

import (
	"encoding/binary"
	"io"
	"reflect"
)

type byteWriter struct {
	buf []byte
	pos int
}

func (b byteWriter) Write(p []byte) (int, error) {
	capacity := len(b.buf) - b.pos
	if capacity < len(p) {
		p = p[:capacity]
	}
	if len(p) > 0 {
		copy(b.buf[b.pos:], p)
		b.pos += len(p)
	}
	return len(p), nil
}

type Packable interface {
	SetByteOrder(order binary.ByteOrder)
	String() string
	Sizeof(val reflect.Value) int
	Pack(buf []byte, val reflect.Value) (int, error)
	Unpack(r io.Reader, val reflect.Value) error
}

type binaryFallback struct {
	val   reflect.Value
	order binary.ByteOrder
}

func (b *binaryFallback) SetByteOrder(order binary.ByteOrder) {
	b.order = order
}

func (b *binaryFallback) String() string {
	return b.val.String()
}

func (b *binaryFallback) Sizeof(val reflect.Value) int {
	return binary.Size(val.Interface())
}

func (b *binaryFallback) Pack(buf []byte, val reflect.Value) (int, error) {
	tmp := byteWriter{buf: buf}
	err := binary.Write(tmp, b.order, val.Interface())
	return tmp.pos, err
}

func (b *binaryFallback) Unpack(r io.Reader, val reflect.Value) error {
	return binary.Read(r, b.order, val.Interface())
}
