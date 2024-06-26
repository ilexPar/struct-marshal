// Experimental tility to "encode" a go struct into another, using json encoding as intermediary.
//
// Intended to translate between API objects and internal system abstractions, providing capabilities to configure how each field should be mapped.
//
// # Usage
//
// The most basic usage is to annotate your struct fields with the `sm` tag, specifying the "jsonpath" to the field in the destination object.
//
//	type MyStruct struct {
//	    Name string `sm:metadata.name`
//	}
//
// And now you just need to call `Marshal` or `Unmarshal` to translate between your structs.
//
//	// load your struct with values from an external object
//	dst := &MyStruct{}
//	src := getSomeThirdPartyData()
//	sm.Unarshal(src, dst)
//
//	// load an third party struct from you internal abstraction
//	dst := &module.SomeStruct{}
//	src := &MyStruct{
//	    Name: "pepito"
//	}
//	sm.Marshal(src, dst)
//
// # Advanced
//
// # Type Matching
//
// By default types wont be checked, but you can specify type matching option like `types<SomeType>`. Multiple types can be annotated for each field using `|` as separator.
// When types don't match field will be skipped
//
// Example:
//
//	type MyStruct struct {
//	    Name string `sm:metadata.name,types<SomeStruct|OtherStruct>`
//	    Flag bool `sm:metadata.name,types<SomeStruct>`
//	}
//
// # Per Type Path
//
// You can specify a different path for each type by appending the path to the type using `:` as separator in the `types<>` option.
//
// Keep in mind:
//
//   - Is mandatory the main path for the field is set to `+` character when using this feature
//   - the `types<>` option will perform the match only based on the type you expect to "encode/decode",
//     so there's no need to know the types of fields disregarding the depth
//
// Example:
//
//	type MyStruct struct {
//	    Name string `sm:+,types<SomeStruct:meta.name|OtherStruct:info.name>`
//	}
//
// # Nesting
//
// By default fields that are structs will inherit the parent path, but you can dismiss this by using the `->` operator as field name in order for the path to be fully processed
//
// Example:
//
//	type PreservedParent struct {
//	    Name `sm:name`
//	}
//	type DismissParent struct {
//	    SomeProperty string `sm:some.path.to.property`
//	}
//
//	type MyStruct struct {
//	    Child1 PreservedParent `sm:some.path.to.use`
//	    Child2 DismissParent `->`
//	}
package pkg

import (
	"encoding/json"
	"fmt"
	"reflect"
)

const (
	FIELD_TAG_KEY    = "sm" // field tag to parse
	TYPES_SPLIT      = "|"  // type separator when encoding to multiple types from a single source, eg sm:"example,types<Struct1|Struct2>"
	TYPES_PATH_SPLIT = ":"  // type path separator when setting per type path, eg sm:"+,types<Struct1:path.one|Struct2:path.name>"
	DISMISS_NESTED   = "->" // path to be used when dismissing path nesting
	MULTI_TYPE_NAME  = "+"  // path name to be used when setting per type path, eg sm:"+,types<Struct1:path.one|Struct2:path.name>"

	ERROR_PER_TYPE_PATH_IS_NOT_VALID = "main path should be '+' when using per-type path matching"

	TYPE_OPTS_REGEX = `^types<([^>]+)>$`
)

// Marshal marshals the given the jsonpath compatible source to a JSON byte slice, and then unmarshals it into the given destination interface{}.
// This function is intended to convert between system internal definitions and the destined API object.
func Marshal(src interface{}, dst interface{}) error {
	dstType := getTypeName(dst)
	b, err := MarshalJSONPath(src, dstType)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &dst)
	return err
}

// Unmarshal marshals the given source and then unmarshals into the jsonpath compatible destination.
// This function is intended to convert between the provided API object and the system internal definitions.
func Unmarshal(src interface{}, dst interface{}) error {
	b, _ := json.Marshal(src)
	srcType := getTypeName(src)
	err := UnmarshalJSONPath(b, dst, srcType)
	return err
}

func getTypeName(t interface{}) string {
	val := reflect.ValueOf(t)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val.Type().Name()
}

// UnmarshalJSONPath unmarshals the given JSON byte slice into the provided destination interface.
// The destination interface must be a non-nil pointer. The function uses the "jsonpath" struct tags
// on the destination interface fields to map the JSON data to the appropriate fields.
func UnmarshalJSONPath(src []byte, dst interface{}, srcTypeName string) error {
	rd := reflect.ValueOf(dst)
	if rd.Kind() != reflect.Pointer || rd.IsNil() {
		return fmt.Errorf("dst must be a non-nil pointer")
	}

	srcData := map[string]interface{}{}
	err := json.Unmarshal(src, &srcData)
	if err != nil {
		return err
	}
	dstData, err := populateStructFromMap(srcData, dst, srcTypeName)
	if err != nil {
		return err
	}
	jsonData, _ := json.Marshal(dstData)
	return json.Unmarshal(jsonData, dst)
}

func populateStructFromMap(
	src map[string]interface{},
	dst interface{},
	srcTypeName string,
	parents ...string,
) (map[string]interface{}, error) {
	dstData := map[string]interface{}{}
	v := reflect.ValueOf(dst)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := range v.NumField() {
		field, err := NewField(i, v, srcTypeName)
		if err != nil {
			return dstData, err
		}
		if field.Skip {
			continue
		}

		field.ChRoot(parents)
		var value any
		if field.IsStruct() {
			// handle nested structs
			value, err = populateStructFromMap(
				src,
				field.Value.Interface(),
				srcTypeName,
				field.GetPathAsParent()...,
			)
			if err != nil {
				return dstData, err
			}
		} else {
			value = field.GetValueFromMap(src)
		}
		if value == nil {
			continue
		}
		if field.IsStructSlice() {
			structList := []any{}
			reflectedStruct := field.Value.Type().Elem()
			for i := range value.([]interface{}) {
				ifaceVal := value.([]interface{})[i].(map[string]interface{})
				v, err := populateStructFromMap(
					ifaceVal, reflect.New(reflectedStruct).Interface(), srcTypeName,
				)
				if err != nil {
					return dstData, err
				}
				structList = append(structList, v)
			}
			value = structList
		}
		dstData[field.stfield.Name] = value
	}
	return dstData, nil
}

// MarshalJSONPath converts the given source interface{} to a JSON-encoded byte slice,
// using the JSON field tags defined on the source struct to map the fields to the
// resulting JSON object.
func MarshalJSONPath(src interface{}, dstTypeName string) ([]byte, error) {
	data := map[string]interface{}{}
	err := populateMapFromStruct(src, data, dstTypeName)
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func populateMapFromStruct(
	src interface{},
	dst map[string]interface{},
	dstTypeName string,
) error {
	v := reflect.ValueOf(src)
	for i := range v.NumField() {
		field, err := NewField(i, v, dstTypeName)
		if err != nil {
			return err
		}
		field.SkipIfEmpty()
		if field.Skip {
			continue
		}
		if field.IsStruct() && field.DissmisNesting(field.Path) {
			// if dismiss nesting then treat the child struct fields as if they
			// were defined in the parent struct
			populateMapFromStruct(field.Value.Interface(), dst, dstTypeName)
		} else {
			field.SetValueIntoMap(dst)
		}
	}
	return nil
}
