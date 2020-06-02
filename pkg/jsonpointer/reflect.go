// TODO: GET RID OF ME
package jsonpointer

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	oldjsonpointer "github.com/dustin/go-jsonpointer"
)

func Reflect(o interface{}, path string) interface{} {
	if path[0] != '/' {
		path = "/" + path
	}
	return oldjsonpointer.Reflect(o, path)
}

func SetReflect(o interface{}, path string, value interface{}) {
	if path == "" {
		return
	}
	parts := strings.Split(strings.TrimLeft(path, "/"), "/")

	dst := reflect.ValueOf(o)

OUTER:
	for idx, p := range parts {
		isLast := false
		if idx == len(parts)-1 {
			isLast = true
		}
		dst = reflect.Indirect(dst)

		if dst.Kind() == reflect.Struct {
			typ := dst.Type()
			for i := 0; i < typ.NumField(); i++ {
				sf := typ.Field(i)
				tag := sf.Tag.Get("json")
				name := parseJSONTagName(tag)
				if (name != "" && name == p) || sf.Name == p {
					dst = dst.Field(i)
					continue OUTER
				}
			}
			// Found no matching field.
			return
		} else if dst.Kind() == reflect.Map {
			mapKey := reflect.ValueOf(p).Convert(dst.Type().Key())
			if isLast {
				dst.SetMapIndex(mapKey, reflect.ValueOf(value))
				return
			} else {
				dst = dst.MapIndex(mapKey)
				// TODO: if value is not a pointer, add to a stack
				// that replaces with SetMapIndex after value is set
			}
		} else if dst.Kind() == reflect.Slice || dst.Kind() == reflect.Array {
			i, err := strconv.Atoi(p)
			if err == nil && i < dst.Len() {
				dst = dst.Index(i)
			} else {
				return
			}
		} else {
			return
		}
	}

	if value == nil {
		dst.Set(reflect.Zero(dst.Type()))
	} else {
		rv := reflect.ValueOf(value)
		var nv reflect.Value
		if rv.Type().Kind() == reflect.Slice {
			fmt.Println(dst.Type())
			nv = reflect.MakeSlice(dst.Type(), 0, 0)
			for i := 0; i < rv.Len(); i++ {
				v := rv.Index(i)
				nv = reflect.Append(nv, v.Convert(nv.Type().Elem()))
			}
		} else {
			nv = rv.Convert(dst.Type())
		}
		dst.Set(nv)
	}
	return
}

// ReflectListPointers lists all possible pointers from the given struct.
func ReflectListPointers(o interface{}) ([]string, error) {
	return reflectListPointersRecursive(o, ""), nil
}

func reflectListPointersRecursive(o interface{}, prefix string) []string {
	rv := []string{prefix + ""}

	val := reflect.ValueOf(o)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {

		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			child := val.Field(i).Interface()
			sf := typ.Field(i)
			tag := sf.Tag.Get("json")
			name := parseJSONTagName(tag)
			if name != "" {
				// use the tag name
				childReults := reflectListPointersRecursive(child, prefix+name)
				rv = append(rv, childReults...)
			} else {
				// use the original field name
				childResults := reflectListPointersRecursive(child, prefix+sf.Name)
				rv = append(rv, childResults...)
			}
		}

	} else if val.Kind() == reflect.Map {
		for _, k := range val.MapKeys() {
			child := val.MapIndex(k).Interface()
			mapKeyName := makeMapKeyName(k)
			childReults := reflectListPointersRecursive(child, prefix+mapKeyName)
			rv = append(rv, childReults...)
		}
	} else if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		for i := 0; i < val.Len(); i++ {
			child := val.Index(i).Interface()
			childResults := reflectListPointersRecursive(child, prefix+strconv.Itoa(i))
			rv = append(rv, childResults...)
		}
	}
	return rv
}

// makeMapKeyName takes a map key value and creates a string representation
func makeMapKeyName(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		fv := v.Float()
		return strconv.FormatFloat(fv, 'f', -1, v.Type().Bits())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		iv := v.Int()
		return strconv.FormatInt(iv, 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		iv := v.Uint()
		return strconv.FormatUint(iv, 10)
	default:
		return v.String()
	}
}

// parseJSONTagName extracts the JSON field name from a struct tag
func parseJSONTagName(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}
