package utils

import (
	"reflect"
	"strconv"
	"strings"
)

// UpdatesFromPtrDTO builds a map[string]any containing only non-nil *fields from a pointer DTO.
// It uses the `json` tag (before any comma options) as the column/key name.
// Optionally provide a renames map to translate json->db column (e.g., {"customer_id":"c_id"}).
func UpdatesFromPtrDTO(dto any, renames map[string]string) map[string]any {
	res := make(map[string]any)
	v := reflect.ValueOf(dto)
	if v.Kind() != reflect.Ptr {
		return res
	}
	s := v.Elem()
	if s.Kind() != reflect.Struct {
		return res
	}
	t := s.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := s.Field(i)
		if fv.Kind() != reflect.Ptr || fv.IsNil() {
			continue
		}
		jsonTag := sf.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		if renames != nil {
			if alt, ok := renames[name]; ok && alt != "" {
				name = alt
			}
		}
		res[name] = fv.Elem().Interface()
	}
	return res
}

func ParseIntDefault(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 0 {
		return v
	}
	return def
}
