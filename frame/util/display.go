package util

import (
	"fmt"
	"reflect"
	"strings"
)

func printValue(v reflect.Value, b *strings.Builder) {
	switch v.Kind() {
	case reflect.Invalid:
		b.WriteString("<invalid> ")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b.WriteString(fmt.Sprintf("%d ", v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		b.WriteString(fmt.Sprintf("%d ", v.Uint()))
	case reflect.Float32, reflect.Float64:
		b.WriteString(fmt.Sprintf("%f ", v.Float()))
	case reflect.String:
		b.WriteString(fmt.Sprintf("%s ", v.String()))
	case reflect.Bool:
		b.WriteString(fmt.Sprintf("%t ", v.Bool()))
	case reflect.Ptr:
		if !v.IsNil() {
			printValue(v.Elem(), b)
		} else {
			b.WriteString("nil ")
		}
	case reflect.Slice, reflect.Array:
		b.WriteString("[")
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				b.WriteString(" ")
			}
			printValue(v.Index(i), b)
		}
		b.WriteString("] ")
	case reflect.Map:
		b.WriteString("{")
		for _, key := range v.MapKeys() {
			if b.Len() > 1 {
				b.WriteString(" ")
			}
			printValue(key, b)
			b.WriteString(": ")
			value := v.MapIndex(key)
			if !value.IsValid() {
				b.WriteString("<invalid> ")
			} else {
				printValue(value, b)
			}
		}
		b.WriteString("} ")
	case reflect.Struct:
		b.WriteString("{")
		for i := 0; i < v.NumField(); i++ {
			if i > 0 {
				b.WriteString(" ")
			}
			fieldType := v.Type().Field(i)
			b.WriteString(fmt.Sprintf("%s:", fieldType.Name))
			fieldValue := v.Field(i)
			if !fieldValue.IsValid() {
				b.WriteString("<invalid> ")
			} else {
				printValue(fieldValue, b)
			}
		}
		b.WriteString("} ")
	case reflect.Interface:
		if !v.IsNil() {
			elem := v.Elem()
			printValue(elem, b)
		} else {
			b.WriteString("nil ")
		}
	default:
		b.WriteString(fmt.Sprintf("unsupported type %s ", v.Type()))
	}
}

func Display(any interface{}) string {
	//var b strings.Builder
	//printValue(reflect.ValueOf(any), &b)
	//return b.String()
	return fmt.Sprint(any)
}
