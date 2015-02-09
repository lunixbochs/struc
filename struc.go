package struc

import (
	"io"
)

func Pack(w io.Writer, data interface{}) error {
	fields, err := ParseFields(data)
	if err != nil {
		return err
	}
	return fields.Pack(w, data)
}

func PackWithOrder(w io.Writer, data interface{}, order int) error {
	if fields, err := ParseFields(data); err == nil {
		if err := fields.SetByteOrder(order); err != nil {
			return err
		}
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

func UnpackWithOrder(r io.Reader, data interface{}, order int) error {
	fields, err := ParseFields(data)
	if err != nil {
		return err
	}
	if err := fields.SetByteOrder(order); err != nil {
		return err
	}
	return fields.Unpack(r, data)
}

func Sizeof(data interface{}) (int, error) {
	fields, err := ParseFields(data)
	if err != nil {
		return 0, err
	}
	return fields.Sizeof(data), nil
}
