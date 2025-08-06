package pkg

import (
	"encoding/json"
	"errors"
	"reflect"
)

func toMap(from interface{}, into map[string]interface{}) (err error) {
	bytes, err := json.Marshal(from)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, &into)
	return err
}

func assertNonNilPointer(check interface{}) (err error) {
	rd := reflect.ValueOf(check)
	if rd.Kind() != reflect.Pointer || rd.IsNil() {
		err = errors.New("must be a non-nil pointer")
	}
	return err
}

// StructDecoder is a struct that provides a way to build a new struct from an existing interface{}.
// It takes a source interface{} and a destination pointer, and generates a new struct
// representation of the destination based on the values in the source.
type StructDecoder struct {
	src          interface{}
	dst          interface{}
	typeRestrain string
}

// Init initializes the StructBuilder with the provided source and destination interfaces.
// It first checks that the dst interface is a non-nil pointer, and returns an error if it is not.
// It then sets the src, dst, and typeRestrain fields of the StructBuilder.
// The typeRestrain field is set to the type name of the src interface.
// This function returns an error if the dst interface is not a non-nil pointer.
func (sb *StructDecoder) Init(src interface{}, dst interface{}) (err error) {
	if err = assertNonNilPointer(dst); err != nil {
		return errors.New("dst must be a non-nil pointer")
	}

	sb.src = src
	sb.dst = dst
	sb.typeRestrain = getTypeName(sb.src)

	return err
}

// Run generates a map[string]interface{} representation of the dst struct, using the values from the src interface{}.
// It first converts the src interface{} to a map[string]interface{} using the toMap function.
// It then recursively generates the map[string]interface{} representation of the dst struct by calling the generate
// function.
// Finally, it marshals the generated map[string]interface{} to JSON and unmarshals it into the dst pointer.
// The function returns an error if any errors occur during the generation process.
func (sb StructDecoder) Run() (err error) {
	input := map[string]interface{}{}
	if err := toMap(sb.src, input); err != nil {
		return err
	}

	reflectedDst := reflect.ValueOf(sb.dst)
	if reflectedDst.Kind() == reflect.Ptr {
		reflectedDst = reflectedDst.Elem()
	}

	out := map[string]interface{}{}
	if err = sb.generate(input, sb.typeRestrain, reflectedDst, out); err != nil {
		return err
	}

	if outEnc, err := json.Marshal(out); err != nil {
		return err
	} else {
		return json.Unmarshal(outEnc, sb.dst)
	}
}

// generate recursively generates a map[string]interface{} representation of the dst struct, using the values from the
// src map[string]interface{}.
// It iterates through each field in the dst struct, and for each field:
// - If the field is a struct, it recursively calls generate() to generate a map[string]interface{} for that struct.
// - If the field is a slice of structs, it calls generateSlice() to generate a slice of map[string]interface{} for that
// slice.
// - Otherwise, it gets the value for that field from the src map and adds it to the into map.
// The function returns an error if any errors occur during the generation process.
func (sb StructDecoder) generate(
	src map[string]interface{},
	typeRestrain string,
	dst reflect.Value,
	into map[string]interface{},
	parents ...string,
) error {
	if dst.Kind() == reflect.Ptr {
		dst = dst.Elem()
	}

	for i := range dst.NumField() {
		field := &Field{}
		if err := field.Init(i, dst, sb.typeRestrain); err != nil {
			return err
		}

		if field.Skip {
			continue
		}
		field.ChRoot(parents)

		var value any
		if field.IsStruct() {
			val := map[string]interface{}{}
			if err := sb.generate(src, typeRestrain, field.Value, val, field.GetPathAsParent()...); err != nil {
				return err
			} else {
				value = val
			}
		} else {
			value = field.GetValueFromMap(src)
		}
		if value == nil {
			continue
		}

		if field.IsStructSlice() {
			val := []any{}
			if err := sb.generateSlice(value.([]interface{}), field, typeRestrain, &val); err != nil {
				return err
			} else {
				value = val
			}
		}
		into[field.stfield.Name] = value
	}

	return nil
}

// generateSlice recursively generates a slice of map[string]interface{} representations of the elements in the value
// slice, using the dst struct type specified by the field parameter.
// It iterates through each element in the value slice, and for each element:
// - It creates a new map[string]interface{} to hold the representation of the element.
// - It calls the generate() function to recursively generate the map[string]interface{} representation of the element,
// using the element's map[string]interface{} value and the dst struct type.
// - It appends the generated map[string]interface{} to the out slice.
// The function returns an error if any errors occur during the generation process.
func (sb StructDecoder) generateSlice(value []interface{}, field *Field, typeRestrain string, out *[]any) error {
	dstType := field.Value.Type().Elem()
	for i := range value {
		val := map[string]interface{}{}
		if err := sb.generate(value[i].(map[string]interface{}), typeRestrain, reflect.New(dstType).Elem(), val); err != nil {
			return err
		}
		*out = append(*out, val)
	}
	return nil
}
