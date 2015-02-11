package struc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// struc:"int32,big,sizeof=Data"

var tagWordsRe = regexp.MustCompile(`(\[|\b)[^"]+\b+$`)

type strucTag struct {
	Type   string
	Order  binary.ByteOrder
	Sizeof string
}

func parseStrucTag(tag reflect.StructTag) (*strucTag, error) {
	t := &strucTag{
		Order: binary.BigEndian,
	}
	for _, s := range strings.Split(tag.Get("struc"), ",") {
		if strings.HasPrefix(s, "sizeof=") {
			tmp := strings.SplitN(s, "=", 2)
			t.Sizeof = tmp[1]
		} else if s == "big" {
			t.Order = binary.BigEndian
		} else if s == "little" {
			t.Order = binary.LittleEndian
		} else {
			t.Type = s
		}
	}
	return t, nil
}

var typeLenRe = regexp.MustCompile(`^\[(\d*)\]`)

func parseField(f reflect.StructField) (fd *Field, err error) {
	tag, err := parseStrucTag(f.Tag)
	if err != nil {
		return nil, err
	}
	var ok bool
	fd = &Field{
		Name:   f.Name,
		Len:    1,
		Order:  tag.Order,
		Slice:  false,
		kind:   f.Type.Kind(),
		offset: f.Offset,
	}
	switch fd.kind {
	case reflect.Array:
		fd.Slice = true
		fd.Len = f.Type.Len()
		fd.kind = f.Type.Elem().Kind()
	case reflect.Slice:
		fd.Slice = true
		fd.Len = -1
		fd.kind = f.Type.Elem().Kind()
	case reflect.Struct:
		panic("struc: struct nesting is not yet supported")
	}
	// find a type in the struct tag
	pureType := typeLenRe.ReplaceAllLiteralString(tag.Type, "")
	if fd.Type, ok = typeLookup[pureType]; ok {
		fd.Len = 1
		match := typeLenRe.FindAllStringSubmatch(tag.Type, -1)
		if len(match) > 0 && len(match[0]) > 1 {
			fd.Slice = true
			first := match[0][1]
			// Field.Len = -1 indicates a []slice
			if first == "" {
				fd.Len = -1
			} else {
				fd.Len, err = strconv.Atoi(first)
			}
		}
		return
	}
	// the user didn't specify a type, or used an unknown type
	if fd.Type, ok = reflectTypeMap[fd.kind]; ok {
		return
	}
	err = errors.New("struc: Could not find field type.")
	return
}

var fieldCache = make(map[reflect.Type]Fields)

func parseFields(v reflect.Value) (Fields, error) {
	t := v.Type()
	if cached, ok := fieldCache[t]; ok {
		return cached, nil
	}
	if v.NumField() < 1 {
		return nil, errors.New("struc: Struct has no fields.")
	}
	sizeofMap := make(map[string][]int)
	fields := make(Fields, 0, v.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		f, err := parseField(field)
		if err != nil {
			return nil, err
		}
		f.CanSet = v.Field(i).CanSet()
		f.Index = i
		tag, err := parseStrucTag(field.Tag)
		if err != nil {
			return nil, err
		}
		if tag.Sizeof != "" {
			target, ok := t.FieldByName(tag.Sizeof)
			if !ok {
				return nil, fmt.Errorf("struc: `sizeof=%s` field does not exist", tag.Sizeof)
			}
			f.Sizeof = target.Index
			sizeofMap[tag.Sizeof] = field.Index
		}
		if sizefrom, ok := sizeofMap[field.Name]; ok {
			f.Sizefrom = sizefrom
		}
		if f.Len == -1 && f.Sizefrom == nil {
			return nil, fmt.Errorf("struc: field `%s` is a slice with no length or Sizeof field", field.Name)
		}
		fields = append(fields, f)
	}
	fieldCache[t] = fields
	return fields, nil
}
