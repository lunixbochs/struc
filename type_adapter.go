package struc

import (
	"reflect"
)

// TypeAdapter 提供类型系统的适配层
// 将现有的硬编码类型系统迁移到新的注册器系统
type TypeAdapter struct {
	registry *TypeRegistry
}

// NewTypeAdapter 创建一个新的类型适配器
func NewTypeAdapter() *TypeAdapter {
	return &TypeAdapter{
		registry: GlobalTypeRegistry,
	}
}

// String 返回类型的字符串表示
// 使用注册器查找类型名称
func (a *TypeAdapter) String(t Type) string {
	if name, exists := a.registry.LookupNameByType(t); exists {
		return name
	}
	return "unknown"
}

// ParseType 解析类型字符串
// 使用注册器查找类型枚举
func (a *TypeAdapter) ParseType(typeStr string) (Type, bool) {
	return a.registry.LookupTypeByName(typeStr)
}

// KindToType 将 reflect.Kind 转换为 Type
// 使用注册器查找类型映射
func (a *TypeAdapter) KindToType(kind reflect.Kind) (Type, bool) {
	return a.registry.LookupTypeByKind(kind)
}

// RegisterType 注册新类型（便捷方法）
func (a *TypeAdapter) RegisterType(name string, typ Type) error {
	return a.registry.RegisterType(name, typ)
}

// RegisterKindMapping 注册类型映射（便捷方法）
func (a *TypeAdapter) RegisterKindMapping(kind reflect.Kind, typ Type) error {
	return a.registry.RegisterKindMapping(kind, typ)
}

// RegisterCustomType 注册自定义类型（便捷方法）
func (a *TypeAdapter) RegisterCustomType(typ reflect.Type, info CustomTypeInfo) error {
	return a.registry.RegisterCustomType(typ, info)
}

// GetRegisteredTypes 获取已注册的类型列表
func (a *TypeAdapter) GetRegisteredTypes() []string {
	return a.registry.GetRegisteredTypes()
}

// GetRegisteredKinds 获取已注册的类型映射
func (a *TypeAdapter) GetRegisteredKinds() map[reflect.Kind]Type {
	return a.registry.GetRegisteredKinds()
}

// GetRegisteredCustomTypes 获取已注册的自定义类型
func (a *TypeAdapter) GetRegisteredCustomTypes() map[reflect.Type]CustomTypeInfo {
	return a.registry.GetRegisteredCustomTypes()
}

// ==================== 向后兼容的全局函数 ====================

// GlobalTypeAdapter 全局类型适配器实例
var GlobalTypeAdapter = NewTypeAdapter()

// TypeToString 全局类型到字符串转换函数
func TypeToString(t Type) string {
	return GlobalTypeAdapter.String(t)
}

// ParseTypeString 全局类型字符串解析函数
func ParseTypeString(typeStr string) (Type, bool) {
	return GlobalTypeAdapter.ParseType(typeStr)
}

// KindToTypeString 全局类型映射函数
func KindToTypeString(kind reflect.Kind) (Type, bool) {
	return GlobalTypeAdapter.KindToType(kind)
}
