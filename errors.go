package struc

import (
	"fmt"
)

// Error 是 struc 包的自定义错误类型
// 提供统一的错误处理接口
type Error struct {
	Code    ErrorCode
	Message string
	Context map[string]interface{}
	wrapped error
}

// ErrorCode 定义了错误代码枚举
type ErrorCode int

const (
	// ErrInvalidType 无效类型错误
	ErrInvalidType ErrorCode = iota + 1

	// ErrBufferTooSmall 缓冲区太小错误
	ErrBufferTooSmall

	// ErrUnsupportedType 不支持的类型错误
	ErrUnsupportedType

	// ErrInvalidOptions 无效配置选项错误
	ErrInvalidOptions

	// ErrFieldMismatch 字段不匹配错误
	ErrFieldMismatch

	// ErrCustomTypeFailed 自定义类型处理失败错误
	ErrCustomTypeFailed

	// ErrTypeRegistration 类型注册失败错误
	ErrTypeRegistration

	// ErrSizeCalculation 大小计算错误
	ErrSizeCalculation

	// ErrPackingFailed 打包失败错误
	ErrPackingFailed

	// ErrUnpackingFailed 解包失败错误
	ErrUnpackingFailed
)

// errorMessages 定义了错误代码对应的错误消息
var errorMessages = map[ErrorCode]string{
	ErrInvalidType:      "invalid type",
	ErrBufferTooSmall:   "buffer too small",
	ErrUnsupportedType:  "unsupported type",
	ErrInvalidOptions:   "invalid options",
	ErrFieldMismatch:    "field mismatch",
	ErrCustomTypeFailed: "custom type operation failed",
	ErrTypeRegistration: "type registration failed",
	ErrSizeCalculation:  "size calculation failed",
	ErrPackingFailed:    "packing failed",
	ErrUnpackingFailed:  "unpacking failed",
}

// NewError 创建一个新的错误
func NewError(code ErrorCode, message string, context ...map[string]interface{}) *Error {
	err := &Error{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}

	// 如果有上下文信息，合并到 Context 中
	if len(context) > 0 && context[0] != nil {
		for k, v := range context[0] {
			err.Context[k] = v
		}
	}

	return err
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("struc: %s", e.Message)
	}
	if msg, exists := errorMessages[e.Code]; exists {
		return fmt.Sprintf("struc: %s", msg)
	}
	return fmt.Sprintf("struc: unknown error (code: %d)", e.Code)
}

// WithContext 添加上下文信息
func (e *Error) WithContext(key string, value interface{}) *Error {
	e.Context[key] = value
	return e
}

// Unwrap 支持错误链
func (e *Error) Unwrap() error {
	return e.wrapped
}

// ==================== 便捷错误创建函数 ====================

// ErrInvalidTypef 创建无效类型错误
func ErrInvalidTypef(format string, args ...interface{}) *Error {
	return NewError(ErrInvalidType, fmt.Sprintf(format, args...))
}

// ErrBufferTooSmallf 创建缓冲区太小错误
func ErrBufferTooSmallf(format string, args ...interface{}) *Error {
	return NewError(ErrBufferTooSmall, fmt.Sprintf(format, args...))
}

// ErrUnsupportedTypef 创建不支持的类型错误
func ErrUnsupportedTypef(format string, args ...interface{}) *Error {
	return NewError(ErrUnsupportedType, fmt.Sprintf(format, args...))
}

// ErrInvalidOptionsf 创建无效配置选项错误
func ErrInvalidOptionsf(format string, args ...interface{}) *Error {
	return NewError(ErrInvalidOptions, fmt.Sprintf(format, args...))
}

// ErrFieldMismatchf 创建字段不匹配错误
func ErrFieldMismatchf(format string, args ...interface{}) *Error {
	return NewError(ErrFieldMismatch, fmt.Sprintf(format, args...))
}

// ErrCustomTypeFailedf 创建自定义类型处理失败错误
func ErrCustomTypeFailedf(format string, args ...interface{}) *Error {
	return NewError(ErrCustomTypeFailed, fmt.Sprintf(format, args...))
}

// ErrTypeRegistrationf 创建类型注册失败错误
func ErrTypeRegistrationf(format string, args ...interface{}) *Error {
	return NewError(ErrTypeRegistration, fmt.Sprintf(format, args...))
}

// ErrSizeCalculationf 创建大小计算错误
func ErrSizeCalculationf(format string, args ...interface{}) *Error {
	return NewError(ErrSizeCalculation, fmt.Sprintf(format, args...))
}

// ErrPackingFailedf 创建打包失败错误
func ErrPackingFailedf(format string, args ...interface{}) *Error {
	return NewError(ErrPackingFailed, fmt.Sprintf(format, args...))
}

// ErrUnpackingFailedf 创建解包失败错误
func ErrUnpackingFailedf(format string, args ...interface{}) *Error {
	return NewError(ErrUnpackingFailed, fmt.Sprintf(format, args...))
}

// ==================== 错误检查工具函数 ====================

// IsInvalidType 检查是否为无效类型错误
func IsInvalidType(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrInvalidType
	}
	return false
}

// IsBufferTooSmall 检查是否为缓冲区太小错误
func IsBufferTooSmall(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrBufferTooSmall
	}
	return false
}

// IsUnsupportedType 检查是否为不支持的类型错误
func IsUnsupportedType(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrUnsupportedType
	}
	return false
}

// IsInvalidOptions 检查是否为无效配置选项错误
func IsInvalidOptions(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrInvalidOptions
	}
	return false
}

// IsFieldMismatch 检查是否为字段不匹配错误
func IsFieldMismatch(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrFieldMismatch
	}
	return false
}

// IsCustomTypeFailed 检查是否为自定义类型处理失败错误
func IsCustomTypeFailed(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCustomTypeFailed
	}
	return false
}

// ==================== 错误包装函数 ====================

// WrapError 包装现有错误为 struc 错误
func WrapError(code ErrorCode, err error, message string) *Error {
	if message == "" {
		message = err.Error()
	}
	e := NewError(code, message)
	e.Context["wrapped_error"] = err
	e.wrapped = err
	return e
}

// WrapErrorf 包装现有错误为 struc 错误（格式化）
func WrapErrorf(code ErrorCode, err error, format string, args ...interface{}) *Error {
	message := fmt.Sprintf(format, args...)
	return WrapError(code, err, message)
}
