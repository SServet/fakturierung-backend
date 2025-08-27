package utils

import (
	"reflect"
	"strings"
)

// NormalizePtrDTO trims *string fields and rounds *float64 fields on a pointer-to-struct DTO.
// Only non-nil pointer fields are touched; nils stay nil so GORM won't update them.
func NormalizePtrDTO(dto any) {
	v := reflect.ValueOf(dto)
	if v.Kind() != reflect.Ptr {
		return
	}
	s := v.Elem()
	if s.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.Kind() != reflect.Ptr || f.IsNil() {
			continue
		}
		ef := f.Elem()
		switch ef.Kind() {
		case reflect.String:
			ef.SetString(strings.TrimSpace(ef.String()))
		case reflect.Float64:
			ef.SetFloat(Round2(ef.Float()))
		}
	}
}

// NormalizeDTO trims string fields and rounds float64 fields on a pointer-to-struct DTO.
// Useful for create DTOs that use non-pointer fields.
func NormalizeDTO(dto any) {
	v := reflect.ValueOf(dto)
	if v.Kind() != reflect.Ptr {
		return
	}
	s := v.Elem()
	if s.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		switch f.Kind() {
		case reflect.String:
			if f.CanSet() {
				f.SetString(strings.TrimSpace(f.String()))
			}
		case reflect.Float64:
			if f.CanSet() {
				f.SetFloat(Round2(f.Float()))
			}
		}
	}
}
