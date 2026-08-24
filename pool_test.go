package struc

import (
	"bytes"
	"sync"
	"testing"
)

// concurrentBorrowMessage 是 ERC#P0-2 回归测试的消息结构。
// Text(string) 与 Data([]byte) 都会走 unpackBasicType 的零拷贝借用路径，
// 从全局 unpackBasicTypeSlicePool 借用底层内存。
type concurrentBorrowMessage struct {
	TextLen int `struc:"int32,sizeof=Text"`
	Text    string
	DataLen int `struc:"int32,sizeof=Data"`
	Data    []byte
}

// TestBytesSlicePoolConcurrentGetSliceNoOverlap 验证并发调用 GetSlice 时，
// 不同调用者获得的切片互不重叠。
//
// 修复前：快路径先 offset.CompareAndSwap 成功、之后才 bytesPtr.Load() 取底层数组，
// 与慢路径"换新块 + offset 归零"交错时，会用旧偏移切进新块，导致两个调用者
// 持有重叠内存，互相覆盖对方数据。本测试为每个 goroutine 的借用切片写入唯一
// 标记，最终校验所有切片内容仍为自身标记，任何重叠都会导致校验失败。
func TestBytesSlicePoolConcurrentGetSliceNoOverlap(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 8
		iterations = 500
		sliceSize  = 1024 // 4KB 块容量下每块只够 4 次分配，频繁触发换代
	)

	pool := NewBytesSlicePool(0) // 0 -> 默认 4096 字节块

	type borrowed struct {
		buf []byte
		tag byte
	}
	results := make([][]borrowed, goroutines)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-start
			tag := byte('A' + g)
			local := make([]borrowed, 0, iterations)
			for i := 0; i < iterations; i++ {
				buf := pool.GetSlice(sliceSize)
				if len(buf) != sliceSize {
					t.Errorf("GetSlice returned wrong length: got %d, want %d", len(buf), sliceSize)
					return
				}
				// 拿到切片后立即写满自己的标记，最大化重叠时的冲突窗口
				for j := range buf {
					buf[j] = tag
				}
				local = append(local, borrowed{buf: buf, tag: tag})
			}
			results[g] = local
		}(g)
	}
	close(start)
	wg.Wait()

	// 借用是永久性的（池只换新块、不复用旧块），因此所有切片的标记应一直保留；
	// 若发生过重叠，后写入者会覆盖先写入者的标记。
	for g, local := range results {
		for i, b := range local {
			for j, v := range b.buf {
				if v != b.tag {
					t.Fatalf("slice overlap detected: goroutine %d iteration %d, byte %d overwritten: got %q, want %q",
						g, i, j, v, b.tag)
				}
			}
		}
	}
}

// TestUnpackConcurrentBorrowFields 是 ERC#P0-2 的端到端回归测试：
// N=8 个 goroutine 并发 Unpack 含 ~1KB string 字段的消息，零拷贝借用的
// string/[]byte 内容不得被其他 goroutine 的解包破坏。
// 修复前该测试在 -race 下会立即报 DATA RACE（fields.go 中 io.ReadFull
// 写入被重叠借用的 buffer）。
func TestUnpackConcurrentBorrowFields(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 8
		iterations = 200
		payloadLen = 1024
	)

	// 每个 goroutine 使用互不相同的预期内容：一旦发生借用内存重叠，
	// 解包结果会被其他 goroutine 的数据覆盖，与预期内容不一致。
	payloads := make([][]byte, goroutines)
	packed := make([][]byte, goroutines)
	for g := range goroutines {
		payloads[g] = make([]byte, payloadLen)
		for i := range payloads[g] {
			payloads[g][i] = byte('A'+g) ^ byte(i%251)
		}
		msg := &concurrentBorrowMessage{
			TextLen: payloadLen,
			Text:    string(payloads[g]),
			DataLen: payloadLen,
			Data:    payloads[g],
		}
		var buf bytes.Buffer
		if err := Pack(&buf, msg); err != nil {
			t.Fatalf("pack message for goroutine %d: %v", g, err)
		}
		packed[g] = buf.Bytes()
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-start
			wantText := string(payloads[g])
			for i := 0; i < iterations; i++ {
				var out concurrentBorrowMessage
				if err := Unpack(bytes.NewReader(packed[g]), &out); err != nil {
					t.Errorf("goroutine %d iteration %d: unpack: %v", g, i, err)
					return
				}
				if out.TextLen != payloadLen || out.Text != wantText {
					t.Errorf("goroutine %d iteration %d: text corrupted, got len %d, want len %d",
						g, i, len(out.Text), payloadLen)
					return
				}
				if out.DataLen != payloadLen || !bytes.Equal(out.Data, payloads[g]) {
					t.Errorf("goroutine %d iteration %d: data corrupted, got len %d, want len %d",
						g, i, len(out.Data), payloadLen)
					return
				}
			}
		}(g)
	}
	close(start)
	wg.Wait()
}
