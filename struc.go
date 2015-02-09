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

func Unpack(r io.Reader, data interface{}) error {
	fields, err := ParseFields(data)
	if err != nil {
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
