package struc

import (
	"testing"
)

func TestTypeString(t *testing.T) {
	if Pad.String() != "pad" {
		t.Fatal("type string representation failed")
	}
}
