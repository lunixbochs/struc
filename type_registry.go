package struc

import (
	"fmt"
	"reflect"
	"sync"
)

// TypeRegistry 是类型系统注册器
// 提供动态注册和查找类型的功能，支持自定义类型扩展
type TypeRegistry struct {
	mu sync.RWMutex

	// 类型名称到 Type 枚举的映射
	nameToType map[string]Type

	// Type 枚举到类型名称的映射
	typeToName map[Type]string

	// reflect.Kind 到 Type 枚举的映射
	kindToType map[reflect.Kind]Type

	// 自定义类型注册表
	customTypes map[reflect.Type]CustomTypeInfo
}

// CustomTypeInfo 包含自定义类型的注册信息
type CustomTypeInfo struct {
	Name       string
	SizeFunc   func(options *Options) int
	PackFunc   func(value reflect.Value, buffer []byte, options *Options) (int, error)
	UnpackFunc func(buffer []byte, value reflect.Value, options *Options) error
}

// GlobalTypeRegistry 全局类型注册器实例
var GlobalTypeRegistry = NewTypeRegistry()

// NewTypeRegistry 创建一个新的类型注册器
func NewTypeRegistry() *TypeRegistry {
	registry := &TypeRegistry{
		nameToType:  make(map[string]Type),
		typeToName:  make(map[Type]string),
		kindToType:  make(map[reflect.Kind]Type),
		customTypes: make(map[reflect.Type]CustomTypeInfo),
	}

	// 初始化内置类型
	registry.initBuiltinTypes()

	return registry
}

// initBuiltinTypes 初始化内置类型
func (r *TypeRegistry) initBuiltinTypes() {
	// 注册内置类型名称映射
	builtinTypes := map[string]Type{
		"pad":     Pad,
		"bool":    Bool,
		"byte":    Uint8,
		"int8":    Int8,
		"uint8":   Uint8,
		"int16":   Int16,
		"uint16":  Uint16,
		"int32":   Int32,
		"uint32":  Uint32,
		"int64":   Int64,
		"uint64":  Uint64,
		"float32": Float32,
		"float64": Float64,
		"size_t":  SizeType,
		"off_t":   OffType,
	}

	for name, typ := range builtinTypes {
		r.nameToType[name] = typ
		r.typeToName[typ] = name
	}

	// 注册内置类型字符串映射
	additionalTypeNames := map[Type]string{
		Invalid:    "invalid",
		String:     "string",
		Struct:     "struct",
		Ptr:        "ptr",
		CustomType: "custom",
	}

	for typ, name := range additionalTypeNames {
		r.typeToName[typ] = name
	}

	// 注册 reflect.Kind 到 Type 的映射
	builtinKindMappings := map[reflect.Kind]Type{
		reflect.Bool:    Bool,
		reflect.Int8:    Int8,
		reflect.Int16:   Int16,
		reflect.Int:     Int32,
		reflect.Int32:   Int32,
		reflect.Int64:   Int64,
		reflect.Uint8:   Uint8,
		reflect.Uint16:  Uint16,
		reflect.Uint:    Uint32,
		reflect.Uint32:  Uint32,
		reflect.Uint64:  Uint64,
		reflect.Float32: Float32,
		reflect.Float64: Float64,
		reflect.String:  String,
		reflect.Struct:  Struct,
		reflect.Ptr:     Ptr,
	}

	for kind, typ := range builtinKindMappings {
		r.kindToType[kind] = typ
	}
}

// RegisterType 注册一个新的类型
func (r *TypeRegistry) RegisterType(name string, typ Type) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.nameToType[name]; exists {
		return fmt.Errorf("type name '%s' already registered to type %v", name, existing)
	}

	if existing, exists := r.typeToName[typ]; exists {
		return fmt.Errorf("type %v already registered with name '%s'", typ, existing)
	}

	r.nameToType[name] = typ
	r.typeToName[typ] = name

	return nil
}

// RegisterKindMapping 注册 reflect.Kind 到 Type 的映射
func (r *TypeRegistry) RegisterKindMapping(kind reflect.Kind, typ Type) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.kindToType[kind]; exists {
		return fmt.Errorf("kind %v already mapped to type %v", kind, existing)
	}

	r.kindToType[kind] = typ
	return nil
}

// RegisterCustomType 注册自定义类型
func (r *TypeRegistry) RegisterCustomType(typ reflect.Type, info CustomTypeInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.customTypes[typ]; exists {
		return fmt.Errorf("custom type %v already registered", typ)
	}

	r.customTypes[typ] = info
	return nil
}

// LookupTypeByName 根据类型名称查找 Type
func (r *TypeRegistry) LookupTypeByName(name string) (Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typ, exists := r.nameToType[name]
	return typ, exists
}

// LookupNameByType 根据 Type 查找类型名称
func (r *TypeRegistry) LookupNameByType(typ Type) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name, exists := r.typeToName[typ]
	return name, exists
}

// LookupTypeByKind 根据 reflect.Kind 查找 Type
func (r *TypeRegistry) LookupTypeByKind(kind reflect.Kind) (Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typ, exists := r.kindToType[kind]
	return typ, exists
}

// LookupCustomType 查找自定义类型信息
func (r *TypeRegistry) LookupCustomType(typ reflect.Type) (CustomTypeInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.customTypes[typ]
	return info, exists
}

// GetRegisteredTypes 返回所有已注册的类型名称
func (r *TypeRegistry) GetRegisteredTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.nameToType))
	for name := range r.nameToType {
		names = append(names, name)
	}
	return names
}

// GetRegisteredKinds 返回所有已注册的 reflect.Kind 映射
func (r *TypeRegistry) GetRegisteredKinds() map[reflect.Kind]Type {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本以避免并发访问问题
	result := make(map[reflect.Kind]Type)
	for kind, typ := range r.kindToType {
		result[kind] = typ
	}
	return result
}

// GetRegisteredCustomTypes 返回所有已注册的自定义类型
func (r *TypeRegistry) GetRegisteredCustomTypes() map[reflect.Type]CustomTypeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本以避免并发访问问题
	result := make(map[reflect.Type]CustomTypeInfo)
	for typ, info := range r.customTypes {
		result[typ] = info
	}
	return result
}
