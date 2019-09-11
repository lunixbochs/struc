package test_pack_init

import (
	"bytes"
	"github.com/lunixbochs/struc"
	"sync"
	"testing"
)

type Example struct {
	I int `struc:int`
}

// TestParallelPack checks whether Pack is goroutine-safe. Run it with -race flag.
// Keep it as a single test in package since it is likely to be triggered on initialization
// of global objects reported as a data race by race detector.
func TestParallelPack(t *testing.T) {
	var wg sync.WaitGroup
	val := Example{}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			_ = struc.Pack(&buf, &val)
		}()
	}
	wg.Wait()
}
