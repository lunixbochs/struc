package struc

import (
	"encoding/binary"
	"io"
)

func Pack(w io.Writer, data interface{}) error {
	fields, err := ParseFields(data)
	if err != nil {
		return err
	}
	return fields.Pack(w, data)
}

// TODO: this is destructive with caching
func PackWithOrder(w io.Writer, data interface{}, order binary.ByteOrder) error {
	if fields, err := ParseFields(data); err == nil {
		fields.SetByteOrder(order)
		return fields.Pack(w, data)
	} else {
		return err
	}
}

func Unpack(r io.Reader, data interface{}) error {
	fields, err := ParseFields(data)
	if err != nil {
		return err
	}
	return fields.Unpack(r, data)
}

func UnpackWithOrder(r io.Reader, data interface{}, order binary.ByteOrder) error {
	fields, err := ParseFields(data)
	if err != nil {
		return err
	}
	fields.SetByteOrder(order)
	return fields.Unpack(r, data)
}

func Sizeof(data interface{}) (int, error) {
	fields, err := ParseFields(data)
	if err != nil {
		return 0, err
	}
	return fields.Sizeof(data), nil
}
