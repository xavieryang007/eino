package compose

import (
	"fmt"
	"reflect"
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

func (m *Mapping) take(input any) (any, error) {
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
		return nil, fmt.Errorf("mapping has FromKey but input is not a map with string typed key, type=%v", inputType)
	}

	v := typedInput.MapIndex(reflect.ValueOf(m.FromMapKey))
	if !v.IsValid() {
		return nil, fmt.Errorf("mapping FromMapKey not found in input. key=%s, input= %v", m.FromMapKey, input)
	}
	return v.Interface(), nil
}

func (m *Mapping) assignTo(output reflect.Value, taken any) error {
	if len(m.ToField) == 0 && len(m.ToMapKey) == 0 {
		return nil
	}

	if len(m.ToField) > 0 && len(m.ToMapKey) > 0 {
		return fmt.Errorf("mapping has both ToField and ToMapKey, m=%+v", m)
	}

	outputType := output.Type()
	if outputType.Kind() == reflect.Ptr {
		outputType = outputType.Elem()
		output = output.Elem()
	}

	if len(m.ToField) > 0 {
		if outputType.Kind() != reflect.Struct {
			return fmt.Errorf("mapping has ToField but output is not a struct, type=%v", outputType)
		}

		field := output.FieldByName(m.ToField)
		if !field.IsValid() {
			return fmt.Errorf("mapping has ToField not found. field=%v, outputType=%v", m.ToField, outputType)
		}

		if !field.CanSet() {
			return fmt.Errorf("mapping has ToField not exported. field=%v, outputType=%v", m.ToField, outputType)
		}

		toSet := reflect.ValueOf(taken)
		if !toSet.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("mapping ToField has a mismatched type. field=%s, from=%v, to=%v", m.ToField, toSet.Type(), field.Type())
		}

		field.Set(toSet)
		return nil
	}

	if outputType.Kind() != reflect.Map {
		return fmt.Errorf("mapping has ToMapKey but output is not a map, type=%v", outputType)
	}

	if !reflect.TypeOf(m.ToMapKey).AssignableTo(outputType.Key()) {
		return fmt.Errorf("mapping has ToMapKey but output is not a map with string typed key, type=%v", outputType)
	}

	toSet := reflect.ValueOf(taken)
	if !toSet.Type().AssignableTo(outputType.Elem()) {
		return fmt.Errorf("mapping ToMapKey has a mismatched type. key=%s, from=%v, to=%v", m.ToMapKey, toSet.Type(), outputType.Elem())
	}

	output.SetMapIndex(reflect.ValueOf(m.ToMapKey), toSet)
	return nil
}

type fieldMapper struct {
	mappings []Mapping
	fType    reflect.Type
	rType    reflect.Type
}

func (m *fieldMapper) fieldMap(input any) (any, error) {
	if len(m.mappings) == 0 {
		return input, nil
	}

	typedInput := reflect.ValueOf(input)
	if !typedInput.Type().AssignableTo(m.fType) {
		return nil, fmt.Errorf("input type mismatch, expected: %v, got: %v", m.fType, typedInput.Type())
	}

	var (
		fType = typedInput.Type()
		rType = m.rType
	)

	fIsPtr := fType.Kind() == reflect.Ptr
	rIsPtr := rType.Kind() == reflect.Ptr

	if !fIsPtr {
		fType = m.fType
	} else {
		fType = m.fType.Elem()
	}

	if !rIsPtr {
		rType = m.rType
	} else {
		rType = m.rType.Elem()
	}

	mapped := reflect.New(rType).Elem()
	from := m.mappings[0].From
	for _, mapping := range m.mappings {
		if mapping.empty() {
			return nil, fmt.Errorf("mapping is empty")
		}

		if len(mapping.ToField) == 0 && len(mapping.ToMapKey) == 0 {
			if len(m.mappings) > 1 {
				return nil, fmt.Errorf("one of the mapping maps to entire input, conflict")
			}
		}

		if mapping.From != from {
			return nil, fmt.Errorf("multiple mappings from the same node have different keys: %s, %s", mapping.From, from)
		}

		taken, err := mapping.take(input)
		if err != nil {
			return nil, err
		}

		err = mapping.assignTo(mapped, taken)
		if err != nil {
			return nil, err
		}
	}

	if rIsPtr {
		return mapped.Addr().Interface(), nil
	}

	return mapped.Interface(), nil
}

func (m *fieldMapper) streamFieldMap(input streamReader) streamReader {
	if len(m.mappings) == 0 {
		return input
	}

	panic("not implemented")
}
