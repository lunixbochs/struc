package struc

import (
	"bytes"
	"reflect"
	"testing"
)

func parseTest(data interface{}) error {
	_, err := parseFields(reflect.ValueOf(data))
	return err
}

type empty struct{}

func TestEmptyStruc(t *testing.T) {
	if err := parseTest(&empty{}); err == nil {
		t.Fatal("failed to error on empty struct")
	}
}

type chanStruct struct {
	Test chan int
}

func TestChanError(t *testing.T) {
	if err := parseTest(&chanStruct{}); err == nil {
		// TODO: should probably ignore channel fields
		t.Fatal("failed to error on struct containing channel")
	}
}

type badSizeof struct {
	Size int `struc:"sizeof=Bad"`
}

func TestBadSizeof(t *testing.T) {
	if err := parseTest(&badSizeof{}); err == nil {
		t.Fatal("failed to error on missing Sizeof target")
	}
}

type missingSize struct {
	Test []byte
}

func TestMissingSize(t *testing.T) {
	if err := parseTest(&missingSize{}); err == nil {
		t.Fatal("failed to error on missing field size")
	}
}

type badNested struct {
	Empty empty
}

func TestNestedParseError(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, &badNested{}); err == nil {
		t.Fatal("failed to error on bad nested struct")
	}
}

// 测试忽略字段的结构体
// Test struct with ignored fields
type ignoreFieldsStruct struct {
	Public  int     `struc:"int32"`
	Ignored bool    `struc:"-"`
	Compat  string  `struct:"-"`
	Private float64 `struc:"float64"`
}

// TestIgnoreFields 测试 struc:"-" 标签的功能
// 验证带有 "-" 标签的字段会被正确忽略
//
// TestIgnoreFields tests the struc:"-" tag functionality
// Verifies that fields with "-" tag are correctly ignored
func TestIgnoreFields(t *testing.T) {
	val := reflect.ValueOf(&ignoreFieldsStruct{})
	fields, err := parseFields(val.Elem())
	if err != nil {
		t.Fatalf("Failed to parse struct with ignored fields: %v", err)
	}

	// 检查字段数量是否正确（应该只有 Public 和 Private 两个字段）
	// Check if the number of fields is correct (should only have Public and Private fields)
	expectedFields := 2
	actualFields := 0
	for _, field := range fields {
		if field != nil {
			actualFields++
		}
	}
	if actualFields != expectedFields {
		t.Errorf("Expected %d fields, got %d fields", expectedFields, actualFields)
	}

	// 验证 Ignored 和 Compat 字段被正确标记为跳过
	// Verify that Ignored and Compat fields are correctly marked as skipped
	if fields[1] != nil || fields[2] != nil {
		t.Error("Fields marked with '-' tag were not properly ignored")
	}

	// 验证 Public 和 Private 字段被正确解析
	// Verify that Public and Private fields are correctly parsed
	if fields[0] == nil || fields[0].Type != Int32 {
		t.Error("Public field was not properly parsed")
	}
	if fields[3] == nil || fields[3].Type != Float64 {
		t.Error("Private field was not properly parsed")
	}
}
