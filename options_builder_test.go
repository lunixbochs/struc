package struc

import (
	"testing"
)

// TestDefaultOptionsReturnsCopy 回归 P1-4：DefaultOptions 必须返回独立副本，
// 调用方修改返回值不得污染全局默认配置
func TestDefaultOptionsReturnsCopy(t *testing.T) {
	opts := DefaultOptions()
	if opts == defaultPackingOptions {
		t.Fatal("DefaultOptions returned the shared global instance instead of a copy")
	}

	opts.ByteAlign = 4
	if got := DefaultOptions().ByteAlign; got != defaultPackingOptions.ByteAlign {
		t.Fatalf("mutation leaked into global defaults: DefaultOptions().ByteAlign = %d, want %d", got, defaultPackingOptions.ByteAlign)
	}
}
