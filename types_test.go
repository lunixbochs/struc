package struc

import (
	"testing"
)

func TestBadType(t *testing.T) {
	defer func() { recover() }()
	Type(-1).Size()
	t.Fatal("failed to panic for invalid Type.Size()")
}

func TestTypeString(t *testing.T) {
	if Pad.String() != "pad" {
		t.Fatal("type string representation failed")
	}
}
