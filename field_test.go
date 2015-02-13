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
