/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compose

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/generic"
)

func takeOne(input any, m *Mapping) (any, error) {
	if len(m.fromField) == 0 && len(m.fromMapKey) == 0 {
		return input, nil
	}

	if len(m.fromField) > 0 && len(m.fromMapKey) > 0 {
		return nil, fmt.Errorf("mapping has both fromField and fromMapKey, m=%s", m)
	}

	inputValue := reflect.ValueOf(input)

	if len(m.fromField) > 0 {
		f, err := checkAndExtractFromField(m.fromField, inputValue)
		if err != nil {
			return nil, err
		}

		return f.Interface(), nil
	}

	v, err := checkAndExtractFromMapKey(m.fromMapKey, inputValue)
	if err != nil {
		return nil, err
	}

	return v.Interface(), nil
}

func assignOne[T any](dest T, taken any, m *Mapping) (T, error) {
	destValue := reflect.ValueOf(dest)

	if !destValue.CanAddr() {
		destValue = reflect.ValueOf(&dest).Elem()
	}

	if len(m.toField) == 0 && len(m.toMapKey) == 0 { // assign to output directly
		toSet := reflect.ValueOf(taken)
		if !toSet.Type().AssignableTo(destValue.Type()) {
			return dest, fmt.Errorf("mapping entire value has a mismatched type. from=%v, to=%v", toSet.Type(), destValue.Type())
		}

		destValue.Set(toSet)

		return destValue.Interface().(T), nil
	}

	if len(m.toField) > 0 && len(m.toMapKey) > 0 {
		return dest, fmt.Errorf("mapping has both toField and toMapKey, m=%s", m)
	}

	toSet := reflect.ValueOf(taken)

	if len(m.toField) > 0 {
		field, err := checkAndExtractToField(m.toField, destValue, toSet)
		if err != nil {
			return dest, err
		}

		field.Set(toSet)
		return destValue.Interface().(T), nil
	}

	key, err := checkAndExtractToMapKey(m.toMapKey, destValue, toSet)
	if err != nil {
		return dest, err
	}

	destValue.SetMapIndex(key, toSet)
	return destValue.Interface().(T), nil
}

func mapFrom[T any](input any, mappings []*Mapping) (T, error) {
	t := generic.NewInstance[T]()

	if len(mappings) == 0 {
		return t, errors.New("mapper has no Mappings")
	}

	from := mappings[0].fromNodeKey
	for _, mapping := range mappings {
		if len(mapping.toField) == 0 && len(mapping.toMapKey) == 0 {
			if len(mappings) > 1 {
				return t, fmt.Errorf("one of the mapping maps to entire input, conflict")
			}
		}

		if mapping.fromNodeKey != from {
			return t, fmt.Errorf("multiple mappings from the same node have different keys: %s, %s", mapping.fromNodeKey, from)
		}

		taken, err := takeOne(input, mapping)
		if err != nil {
			return t, err
		}

		t, err = assignOne(t, taken, mapping)
		if err != nil {
			return t, err
		}
	}

	return t, nil
}

type fieldMapFn func(any) (any, error)
type streamFieldMapFn func(streamReader) streamReader

type defaultFieldMapper[T any] struct{}

func (d defaultFieldMapper[T]) fieldMap(mappings []*Mapping) fieldMapFn {
	return func(input any) (any, error) {
		return mapFrom[T](input, mappings)
	}
}

func (d defaultFieldMapper[T]) streamFieldMap(mappings []*Mapping) streamFieldMapFn {
	return func(input streamReader) streamReader {
		converted := schema.StreamReaderWithConvert(input.toAnyStreamReader(), func(v any) (T, error) {
			return mapFrom[T](v, mappings)
		})

		return packStreamReader(converted)
	}
}

type fieldMapper interface {
	streamFieldMap(mappings []*Mapping) streamFieldMapFn
	fieldMap(mappings []*Mapping) fieldMapFn
}

var (
	anyType = reflect.TypeOf((*any)(nil)).Elem()
	strType = reflect.TypeOf("")
)

func checkAndExtractFromField(fromField string, input reflect.Value) (reflect.Value, error) {
	if input.Kind() == reflect.Ptr {
		input = input.Elem()
	}

	if input.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("mapping has fromField but input is not struct or struct ptr, type= %v", input.Type())
	}

	f := input.FieldByName(fromField)
	if !f.IsValid() {
		return reflect.Value{}, fmt.Errorf("mapping has fromField not found. field=%v, inputType=%v", fromField, input.Type())
	}

	if !f.CanInterface() {
		return reflect.Value{}, fmt.Errorf("mapping has fromField not exported. field= %v, inputType=%v", fromField, input.Type())
	}

	return f, nil
}

func checkAndExtractFromMapKey(fromMapKey string, input reflect.Value) (reflect.Value, error) {
	if input.Kind() != reflect.Map {
		return reflect.Value{}, fmt.Errorf("mapping has FromKey but input is not a map, type=%v", input.Type())
	}

	if !reflect.TypeOf(fromMapKey).AssignableTo(input.Type().Key()) {
		return reflect.Value{}, fmt.Errorf("mapping has FromKey but input is not a map with string key, type=%v", input.Type())
	}

	v := input.MapIndex(reflect.ValueOf(fromMapKey))
	if !v.IsValid() {
		return reflect.Value{}, fmt.Errorf("mapping fromMapKey not found in input. key=%s, inputType= %v", fromMapKey, input.Type())
	}

	return v, nil
}

func checkAndExtractToField(toField string, output, toSet reflect.Value) (reflect.Value, error) {
	if output.Kind() == reflect.Ptr {
		output = output.Elem()
	}

	if output.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("mapping has toField but output is not a struct, type=%v", output.Type())
	}

	field := output.FieldByName(toField)
	if !field.IsValid() {
		return reflect.Value{}, fmt.Errorf("mapping has toField not found. field=%v, outputType=%v", toField, output.Type())
	}

	if !field.CanSet() {
		return reflect.Value{}, fmt.Errorf("mapping has toField not exported. field=%v, outputType=%v", toField, output.Type())
	}

	if !toSet.Type().AssignableTo(field.Type()) {
		return reflect.Value{}, fmt.Errorf("mapping toField has a mismatched type. field=%s, from=%v, to=%v", toField, toSet.Type(), field.Type())
	}

	return field, nil
}

func checkAndExtractToMapKey(toMapKey string, output, toSet reflect.Value) (reflect.Value, error) {
	if output.Kind() != reflect.Map {
		return reflect.Value{}, fmt.Errorf("mapping has toMapKey but output is not a map, type=%v", output.Type())
	}

	if !reflect.TypeOf(toMapKey).AssignableTo(output.Type().Key()) {
		return reflect.Value{}, fmt.Errorf("mapping has toMapKey but output is not a map with string key, type=%v", output.Type())
	}

	if !toSet.Type().AssignableTo(output.Type().Elem()) {
		return reflect.Value{}, fmt.Errorf("mapping toMapKey has a mismatched type. key=%s, from=%v, to=%v", toMapKey, toSet.Type(), output.Type().Elem())
	}

	return reflect.ValueOf(toMapKey), nil
}

func checkAndExtractMapValueType(typ reflect.Type) (reflect.Type, error) {
	if typ.Kind() != reflect.Map {
		return nil, fmt.Errorf("type[%v] is not a map", typ)
	}

	if typ.Key() != strType {
		return nil, fmt.Errorf("type[%v] is not a map with string key", typ)
	}

	return typ.Elem(), nil
}

func checkAndExtractFieldType(field string, typ reflect.Type) (reflect.Type, error) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type[%v] is not a struct", typ)
	}

	f, ok := typ.FieldByName(field)
	if !ok {
		return nil, fmt.Errorf("type[%v] has no field[%s]", typ, field)
	}

	if !f.IsExported() {
		return nil, fmt.Errorf("type[%v] has an unexported field[%s]", typ.String(), field)
	}

	return f.Type, nil
}

func checkMappingGroup(mappings []*Mapping) error {
	if len(mappings) <= 1 {
		return nil
	}

	var (
		fromMapKeyFlag, toMapKeyFlag, fromFieldFlag, toFieldFlag bool
		fromMap                                                  = make(map[string]bool, len(mappings))
		toMap                                                    = make(map[string]bool, len(mappings))
	)

	for _, mapping := range mappings {
		if mapping.empty() {
			return errors.New("multiple mappings have an empty mapping")
		}

		if len(mapping.fromField) == 0 && len(mapping.fromMapKey) == 0 {
			return fmt.Errorf("multiple mappings have a mapping from entire input, mapping= %s", mapping)
		}

		if len(mapping.toField) == 0 && len(mapping.toMapKey) == 0 {
			return fmt.Errorf("multiple mappings have a mapping to entire output, mapping= %s", mapping)
		}

		if len(mapping.fromMapKey) > 0 {
			if fromFieldFlag {
				return fmt.Errorf("multiple mappings have both FromField and FromMapKey, mappings=%v", mappings)
			}

			if _, ok := fromMap[mapping.fromMapKey]; ok {
				return fmt.Errorf("multiple mappings have the same FromMapKey = %s, mappings=%v", mapping.fromMapKey, mappings)
			}

			fromMapKeyFlag = true
			fromMap[mapping.fromMapKey] = true
		}

		if len(mapping.fromField) > 0 {
			if fromMapKeyFlag {
				return fmt.Errorf("multiple mappings have both FromField and FromMapKey, mappings=%v", mappings)
			}

			if _, ok := fromMap[mapping.fromField]; ok {
				return fmt.Errorf("multiple mappings have the same FromField = %s, mappings=%v", mapping.fromField, mappings)
			}

			fromFieldFlag = true
			fromMap[mapping.fromField] = true
		}

		if len(mapping.toMapKey) > 0 {
			if toFieldFlag {
				return fmt.Errorf("multiple mappings have both ToField and ToMapKey, mappings=%v", mappings)
			}

			if _, ok := toMap[mapping.toMapKey]; ok {
				return fmt.Errorf("multiple mappings have the same ToMapKey = %s, mappings=%v", mapping.toMapKey, mappings)
			}

			toMapKeyFlag = true
			toMap[mapping.toMapKey] = true
		}

		if len(mapping.toField) > 0 {
			if toMapKeyFlag {
				return fmt.Errorf("multiple mappings have both ToField and ToMapKey, mappings=%v", mappings)
			}

			if _, ok := toMap[mapping.toField]; ok {
				return fmt.Errorf("multiple mappings have the same ToField = %s, mappings=%v", mapping.toField, mappings)
			}

			toFieldFlag = true
			toMap[mapping.toField] = true
		}
	}

	return nil
}
