package compose

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/generic"
)

type Mapping struct {
	From string

	FromField  string
	FromMapKey string

	ToField  string
	ToMapKey string
}

func (m *Mapping) empty() bool {
	return len(m.FromField) == 0 && len(m.FromMapKey) == 0 && len(m.ToField) == 0 && len(m.ToMapKey) == 0
}

func takeOne(input any, m Mapping) (any, error) {
	if len(m.FromField) == 0 && len(m.FromMapKey) == 0 {
		return input, nil
	}

	if len(m.FromField) > 0 && len(m.FromMapKey) > 0 {
		return nil, fmt.Errorf("mapping has both FromField and FromMapKey, m=%+v", m)
	}

	typedInput := reflect.ValueOf(input)
	inputType := typedInput.Type()

	if inputType.Kind() == reflect.Ptr {
		inputType = inputType.Elem()
		typedInput = typedInput.Elem()
	}

	if len(m.FromField) > 0 {
		if inputType.Kind() != reflect.Struct {
			return nil, fmt.Errorf("mapping has FromField but input is not struct or struct ptr, type= %v", inputType)
		}

		f, ok := inputType.FieldByName(m.FromField)
		if !ok {
			return nil, fmt.Errorf("mapping has FromField not found. field=%v, input=%v", m.FromField, input)
		}

		if !f.IsExported() {
			return nil, fmt.Errorf("mapping has FromField not exported. field= %v, input=%v", m.FromField, input)
		}

		return typedInput.FieldByName(m.FromField).Interface(), nil
	}

	if inputType.Kind() != reflect.Map {
		return nil, fmt.Errorf("mapping has FromKey but input is not a map, type=%v", inputType)
	}

	if !reflect.TypeOf(m.FromMapKey).AssignableTo(inputType.Key()) {
		return nil, fmt.Errorf("mapping has FromKey but input is not a map with string key, type=%v", inputType)
	}

	v := typedInput.MapIndex(reflect.ValueOf(m.FromMapKey))
	if !v.IsValid() {
		return nil, fmt.Errorf("mapping FromMapKey not found in input. key=%s, input= %v", m.FromMapKey, input)
	}
	return v.Interface(), nil
}

func assignOne[T any](dest T, taken any, m Mapping) (T, error) {
	outputType := generic.TypeOf[T]()
	toAssign := reflect.ValueOf(dest)

	if !toAssign.CanAddr() {
		toAssign = reflect.ValueOf(&dest).Elem()
	}

	if len(m.ToField) == 0 && len(m.ToMapKey) == 0 { // assign to output directly
		toSet := reflect.ValueOf(taken)
		if !toSet.Type().AssignableTo(outputType) {
			return dest, fmt.Errorf("mapping entire value has a mismatched type. from=%v, to=%v", toSet.Type(), outputType)
		}

		toAssign.Set(toSet)

		return toAssign.Interface().(T), nil
	}

	if len(m.ToField) > 0 && len(m.ToMapKey) > 0 {
		return dest, fmt.Errorf("mapping has both ToField and ToMapKey, m=%+v", m)
	}

	if len(m.ToField) > 0 {
		realToAssign := toAssign

		if outputType.Kind() == reflect.Ptr {
			outputType = outputType.Elem()
			realToAssign = realToAssign.Elem()
		}

		if outputType.Kind() != reflect.Struct {
			return dest, fmt.Errorf("mapping has ToField but output is not a struct, type=%v", outputType)
		}

		field := realToAssign.FieldByName(m.ToField)
		if !field.IsValid() {
			return dest, fmt.Errorf("mapping has ToField not found. field=%v, outputType=%v", m.ToField, outputType)
		}

		if !field.CanSet() {
			return dest, fmt.Errorf("mapping has ToField not exported. field=%v, outputType=%v", m.ToField, outputType)
		}

		toSet := reflect.ValueOf(taken)
		if !toSet.Type().AssignableTo(field.Type()) {
			return dest, fmt.Errorf("mapping ToField has a mismatched type. field=%s, from=%v, to=%v", m.ToField, toSet.Type(), field.Type())
		}

		field.Set(toSet)
		return toAssign.Interface().(T), nil
	}

	if outputType.Kind() != reflect.Map {
		return dest, fmt.Errorf("mapping has ToMapKey but output is not a map, type=%v", outputType)
	}

	if !reflect.TypeOf(m.ToMapKey).AssignableTo(outputType.Key()) {
		return dest, fmt.Errorf("mapping has ToMapKey but output is not a map with string key, type=%v", outputType)
	}

	toSet := reflect.ValueOf(taken)
	if !toSet.Type().AssignableTo(outputType.Elem()) {
		return dest, fmt.Errorf("mapping ToMapKey has a mismatched type. key=%s, from=%v, to=%v", m.ToMapKey, toSet.Type(), outputType.Elem())
	}

	toAssign.SetMapIndex(reflect.ValueOf(m.ToMapKey), toSet)
	return toAssign.Interface().(T), nil
}

type fieldMapper[T any] struct {
	mappings []Mapping
}

func (f *fieldMapper[T]) mapFrom(input any) (T, error) {
	t := generic.NewInstance[T]()

	if len(f.mappings) == 0 {
		return t, errors.New("mapper has no Mappings")
	}

	from := f.mappings[0].From
	for _, mapping := range f.mappings {
		if len(mapping.ToField) == 0 && len(mapping.ToMapKey) == 0 {
			if len(f.mappings) > 1 {
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

func (f *fieldMapper[T]) fieldMap() func(any) (any, error) {
	return func(input any) (any, error) {
		return f.mapFrom(input)
	}
}

func (f *fieldMapper[T]) streamFieldMap() func(input streamReader) streamReader {
	return func(input streamReader) streamReader {
		converted := schema.StreamReaderWithConvert(input.toAnyStreamReader(), func(v any) (T, error) {
			return f.mapFrom(v)
		})

		return packStreamReader(converted)
	}
}
