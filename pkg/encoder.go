package pkg

import (
	"encoding/json"
	"reflect"
)

// StructEncoder is a struct that provides a way to convert an arbitrary Go struct
// data into another struct. It can be used to easily
// marshal complex Go data structures into other Go data structure.
type StructEncoder struct {
	src          interface{}
	dst          interface{}
	typeRestrain string
}

func (mb *StructEncoder) Init(src interface{}, dst interface{}) error {
	mb.src = src
	mb.dst = dst
	mb.typeRestrain = getTypeName(dst)
	return nil
}

// Run generates a map[string]interface{} from the source object provided to the MapBuilder,
// and then marshals that map to JSON and unmarshals it into the destination object.
// This allows converting arbitrary Go structs into a flat map representation.
func (mb StructEncoder) Run() error {
	out := map[string]interface{}{}
	if err := mb.generate(mb.src, out); err != nil {
		return err
	}

	if outEnc, err := json.Marshal(out); err != nil {
		return err
	} else {
		return json.Unmarshal(outEnc, &mb.dst)
	}
}

// generate recursively traverses the src interface{} and populates the into map[string]interface{}
// with the values from the src. It handles nested structs by either recursively calling generate
// on them, or by flattening their fields into the into map if the DissmisNesting flag is set.
// Any fields that are skipped (e.g. empty values) are not added to the into map.
func (mb StructEncoder) generate(src interface{}, into map[string]interface{}) error {
	data := reflect.ValueOf(src)
	if data.Kind() == reflect.Ptr {
		data = data.Elem()
	}
	for i := range data.NumField() {
		field := &Field{}
		if err := field.Init(i, data, mb.typeRestrain); err != nil {
			return err
		}
		field.SkipIfEmpty()
		if field.Skip {
			continue
		}

		if field.IsStruct() && field.DissmisNesting(field.Path) {
			// if dismiss nesting then treat the child struct fields as if they
			// were defined in the parent struct
			mb.generate(field.Value.Interface(), into)
		} else {
			field.SetValueIntoMap(into)
		}
	}

	return nil
}
