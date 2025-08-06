// Experimental utility to "encode" a go struct into another, using json encoding as intermediary.
//
// Intended to translate between API objects and internal system abstractions, providing capabilities to configure how
// each field should be mapped.
//
// # Usage
//
// The most basic usage is to annotate your struct fields with the `sm` tag, specifying the "jsonpath" to the field in
// the destination object.
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
// By default types wont be checked, but you can specify type matching option like `types<SomeType>`. Multiple types can
// be annotated for each field using `|` as separator.
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
// You can specify a different path for each type by appending the path to the type using `:` as separator in the
// `types<>` option.
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
// By default fields that are structs will inherit the parent path, but you can dismiss this by using the `->` operator
// as field name in order for the path to be fully processed
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
	"reflect"
)

const (
	// field tag to parse
	FIELD_TAG_KEY = "sm"
	// type separator when encoding to multiple types from a single source, eg sm:"example,types<Struct1|Struct2>"
	TYPES_SPLIT = "|"
	// type path separator when setting per type path, eg sm:"+,types<Struct1:path.one|Struct2:path.name>"
	TYPES_PATH_SPLIT = ":"
	// path to be used when dismissing path nesting
	DISMISS_NESTED = "->"
	// path name to be used when setting per type path, eg sm:"+,types<Struct1:path.one|Struct2:path.name>"
	MULTI_TYPE_NAME = "+"

	ERROR_PER_TYPE_PATH_IS_NOT_VALID = "main path should be '+' when using per-type path matching"

	TYPE_OPTS_REGEX = `^types<([^>]+)>$`
)

func getTypeName(t interface{}) string {
	val := reflect.ValueOf(t)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val.Type().Name()
}

// Unmarshal marshals the given source and then unmarshals into the jsonpath compatible destination.
// This function is intended to convert between the provided API object and the system internal definitions.
func Unmarshal(src interface{}, dst interface{}) (err error) {
	decoder := &StructDecoder{}
	if err := decoder.Init(src, dst); err != nil {
		return err
	}

	return decoder.Run()
}

// Marshal marshals the given jsonpath compatible source to a JSON byte slice,
// and then unmarshals it into the given destination interface{}.
// This function is intended to convert between system internal definitions and the destined API object.
func Marshal(src interface{}, dst interface{}) error {
	encoder := &StructEncoder{}
	if err := encoder.Init(src, dst); err != nil {
		return err
	}
	return encoder.Run()
}
