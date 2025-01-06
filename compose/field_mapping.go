package compose

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/generic"
)

func takeOne(input any, m Mapping) (any, error) {
	if len(m.FromField) == 0 && len(m.FromMapKey) == 0 {
		return input, nil
	}

	if len(m.FromField) > 0 && len(m.FromMapKey) > 0 {
		return nil, fmt.Errorf("mapping has both FromField and FromMapKey, m=%+v", m)
	}

	inputValue := reflect.ValueOf(input)

	if len(m.FromField) > 0 {
		f, err := checkAndExtractFromField(m.FromField, inputValue)
		if err != nil {
			return nil, err
		}

		return f.Interface(), nil
	}

	v, err := checkAndExtractFromMapKey(m.FromMapKey, inputValue)
	if err != nil {
		return nil, err
	}

	return v.Interface(), nil
}

func assignOne[T any](dest T, taken any, m Mapping) (T, error) {
	destValue := reflect.ValueOf(dest)

	if !destValue.CanAddr() {
		destValue = reflect.ValueOf(&dest).Elem()
	}

	if len(m.ToField) == 0 && len(m.ToMapKey) == 0 { // assign to output directly
		toSet := reflect.ValueOf(taken)
		if !toSet.Type().AssignableTo(destValue.Type()) {
			return dest, fmt.Errorf("mapping entire value has a mismatched type. from=%v, to=%v", toSet.Type(), destValue.Type())
		}

		destValue.Set(toSet)

		return destValue.Interface().(T), nil
	}

	if len(m.ToField) > 0 && len(m.ToMapKey) > 0 {
		return dest, fmt.Errorf("mapping has both ToField and ToMapKey, m=%+v", m)
	}

	toSet := reflect.ValueOf(taken)

	if len(m.ToField) > 0 {
		field, err := checkAndExtractToField(m.ToField, destValue, toSet)
		if err != nil {
			return dest, err
		}

		field.Set(toSet)
		return destValue.Interface().(T), nil
	}

	key, err := checkAndExtractToMapKey(m.ToMapKey, destValue, toSet)
	if err != nil {
		return dest, err
	}

	destValue.SetMapIndex(key, toSet)
	return destValue.Interface().(T), nil
}

func mapFrom[T any](input any, mappings []Mapping) (T, error) {
	t := generic.NewInstance[T]()

	if len(mappings) == 0 {
		return t, errors.New("mapper has no Mappings")
	}

	from := mappings[0].From
	for _, mapping := range mappings {
		if len(mapping.ToField) == 0 && len(mapping.ToMapKey) == 0 {
			if len(mappings) > 1 {
				return t, fmt.Errorf("one of the mapping maps to entire input, conflict")
			}
		}

		if mapping.From != from {
			return t, fmt.Errorf("multiple mappings from the same node have different keys: %s, %s", mapping.From, from)
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

type defaultFieldMapper[T any] struct{}

func (d defaultFieldMapper[T]) fieldMap(mappings []Mapping) func(input any) (any, error) {
	return func(input any) (any, error) {
		return mapFrom[T](input, mappings)
	}
}

func (d defaultFieldMapper[T]) streamFieldMap(mappings []Mapping) func(input streamReader) streamReader {
	return func(input streamReader) streamReader {
		converted := schema.StreamReaderWithConvert(input.toAnyStreamReader(), func(v any) (T, error) {
			return mapFrom[T](v, mappings)
		})

		return packStreamReader(converted)
	}
}

type fieldMapper interface {
	streamFieldMap(mappings []Mapping) func(input streamReader) streamReader
	fieldMap(mappings []Mapping) func(input any) (any, error)
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
		return reflect.Value{}, fmt.Errorf("mapping has FromField but input is not struct or struct ptr, type= %v", input.Type())
	}

	f := input.FieldByName(fromField)
	if !f.IsValid() {
		return reflect.Value{}, fmt.Errorf("mapping has FromField not found. field=%v, inputType=%v", fromField, input.Type())
	}

	if !f.CanInterface() {
		return reflect.Value{}, fmt.Errorf("mapping has FromField not exported. field= %v, inputType=%v", fromField, input.Type())
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
		return reflect.Value{}, fmt.Errorf("mapping FromMapKey not found in input. key=%s, inputType= %v", fromMapKey, input.Type())
	}

	return v, nil
}

func checkAndExtractToField(toField string, output, toSet reflect.Value) (reflect.Value, error) {
	if output.Kind() == reflect.Ptr {
		output = output.Elem()
	}

	if output.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("mapping has ToField but output is not a struct, type=%v", output.Type())
	}

	field := output.FieldByName(toField)
	if !field.IsValid() {
		return reflect.Value{}, fmt.Errorf("mapping has ToField not found. field=%v, outputType=%v", toField, output.Type())
	}

	if !field.CanSet() {
		return reflect.Value{}, fmt.Errorf("mapping has ToField not exported. field=%v, outputType=%v", toField, output.Type())
	}

	if !toSet.Type().AssignableTo(field.Type()) {
		return reflect.Value{}, fmt.Errorf("mapping ToField has a mismatched type. field=%s, from=%v, to=%v", toField, toSet.Type(), field.Type())
	}

	return field, nil
}

func checkAndExtractToMapKey(toMapKey string, output, toSet reflect.Value) (reflect.Value, error) {
	if output.Kind() != reflect.Map {
		return reflect.Value{}, fmt.Errorf("mapping has ToMapKey but output is not a map, type=%v", output.Type())
	}

	if !reflect.TypeOf(toMapKey).AssignableTo(output.Type().Key()) {
		return reflect.Value{}, fmt.Errorf("mapping has ToMapKey but output is not a map with string key, type=%v", output.Type())
	}

	if !toSet.Type().AssignableTo(output.Type().Elem()) {
		return reflect.Value{}, fmt.Errorf("mapping ToMapKey has a mismatched type. key=%s, from=%v, to=%v", toMapKey, toSet.Type(), output.Type().Elem())
	}

	return reflect.ValueOf(toMapKey), nil
}

func checkAndExtractMapValueType(mapKey string, typ reflect.Type) (reflect.Type, error) {
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

	from := mappings[0].From
	var fromMapKeyFlag, toMapKeyFlag, fromFieldFlag, toFieldFlag bool
	fromMap := make(map[string]bool, len(mappings))
	toMap := make(map[string]bool, len(mappings))
	for _, mapping := range mappings {
		if mapping.From != from {
			return fmt.Errorf("multiple mappings from the same group have from node keys: %s, %s", mapping.From, from)
		}

		if len(mapping.FromMapKey) > 0 {
			if fromFieldFlag {
				return fmt.Errorf("multiple mappings from the same group have both from field and from map key, mappings=%v", mappings)
			}

			if _, ok := fromMap[mapping.FromMapKey]; ok {
				return fmt.Errorf("multiple mappings from the same group have the same from map key = %s, mappings=%v", mapping.FromMapKey, mappings)
			}

			fromMapKeyFlag = true
			fromMap[mapping.FromMapKey] = true
		}

		if len(mapping.FromField) > 0 {
			if fromMapKeyFlag {
				return fmt.Errorf("multiple mappings from the same group have both from field and from map key, mappings=%v", mappings)
			}

			if _, ok := fromMap[mapping.FromField]; ok {
				return fmt.Errorf("multiple mappings from the same group have the same from field = %s, mappings=%v", mapping.FromField, mappings)
			}

			fromFieldFlag = true
			fromMap[mapping.FromField] = true
		}

		if len(mapping.ToMapKey) > 0 {
			if toFieldFlag {
				return fmt.Errorf("multiple mappings from the same group have both to field and to map key, mappings=%v", mappings)
			}

			if _, ok := toMap[mapping.ToMapKey]; ok {
				return fmt.Errorf("multiple mappings from the same group have the same to map key = %s, mappings=%v", mapping.ToMapKey, mappings)
			}

			toMapKeyFlag = true
			toMap[mapping.ToMapKey] = true
		}

		if len(mapping.ToField) > 0 {
			if toMapKeyFlag {
				return fmt.Errorf("multiple mappings from the same group have both to field and to map key, mappings=%v", mappings)
			}

			if _, ok := toMap[mapping.ToField]; ok {
				return fmt.Errorf("multiple mappings from the same group have the same to field = %s, mappings=%v", mapping.ToField, mappings)
			}

			toFieldFlag = true
			toMap[mapping.ToField] = true
		}
	}

	return nil
}
