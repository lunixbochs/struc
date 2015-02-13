package struc

import (
	"testing"
)

func TestBadValue(t *testing.T) {
	if err := Pack(nil, nil); err == nil {
		t.Fatal("failed throw error for bad struct value")
	}
	if err := Unpack(nil, nil); err == nil {
		t.Fatal("failed throw error for bad struct value")
	}
	if _, err := Sizeof(nil); err == nil {
		t.Fatal("failed to throw error for bad struct value")
	}
}

func TestBadType(t *testing.T) {
	defer func() { recover() }()
	Type(-1).Size()
	t.Fatal("failed to panic for invalid Type.Size()")
}
