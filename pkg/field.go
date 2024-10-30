package pkg

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

type Field struct {
	tag     FieldTag
	stfield reflect.StructField
	Target  string
	Value   reflect.Value
	Kind    reflect.Kind
	Skip    bool
	Path    []string
}

// Configures a Field instance from the provided struct value and root struct name.
// It parses the field tag, resolves the field path, and sets the field's Kind, Skip, and tag properties.
// If an error occurs during field path resolution, it is returned.
func (f *Field) Init(idx int, structValue reflect.Value, rootStruct string) (err error) {
	f.Value = structValue.Field(idx)
	f.Target = rootStruct
	f.stfield = structValue.Type().Field(idx)
	f.Kind = f.Value.Kind()
	tag, skip := parseTag(f.stfield)
	f.tag = tag

	if skip {
		f.Skip = skip
		return err
	}

	return f.resolvePath()
}

// SkipIfEmpty sets the Skip field to true if the Value field is the zero value.
// This is a utility method to easily skip fields that have no value.
func (f *Field) SkipIfEmpty() {
	if f.Value.IsZero() {
		f.Skip = true
	}
}

// will return nil if field path should be dismissed in favour
// of the child one
func (f *Field) GetPathAsParent() []string {
	var path []string
	if !f.DissmisNesting(f.Path) {
		path = f.Path
	}
	return path
}

func (f *Field) IsStruct() bool {
	if f.Kind == reflect.Ptr {
		return f.Value.Elem().Kind() == reflect.Struct
	} else {
		return f.Kind == reflect.Struct
	}
}

func (f *Field) IsStructSlice() bool {
	return f.Kind == reflect.Slice && f.Value.Type().Elem().Kind() == reflect.Struct
}

func (f Field) DissmisNesting(path []string) bool {
	return path[0] == DISMISS_NESTED
}

// SetValueIntoMap sets the value of the field into the provided map at the given path.
// If the path is nil, it uses the field's Path.
// If the path has only one element, it sets the field's value directly in the map.
// If the path has two or more elements, it recursively sets the value in the nested map.
func (f *Field) SetValueIntoMap(dst map[string]interface{}, path ...string) {
	if path == nil {
		path = f.Path
	}

	if len(path) == 1 {
		if f.DissmisNesting(path) {
			dst = f.getFieldValue(f.Value).(map[string]interface{})
		} else {
			dst[path[0]] = f.getFieldValue(f.Value)
		}
	}
	if len(path) >= 2 {
		nested := parseNestedPath(dst, path)
		data := initEmptyNestedMapField(nested, dst)
		f.SetValueIntoMap(data, path[1:]...)
	}
}

func initEmptyNestedMapField(nested NestedPath, from map[string]interface{}) map[string]interface{} {
	if nested.data == nil {
		if nested.isArray {
			from[nested.field] = make([]interface{}, 1)
			from[nested.field].([]interface{})[nested.idx] = map[string]interface{}{}
			return from[nested.field].([]interface{})[nested.idx].(map[string]interface{})
		} else {
			from[nested.field] = map[string]interface{}{}
			return from[nested.field].(map[string]interface{})
		}
	} else {
		return nested.data
	}
}

// GetValueFromMap retrieves the value from the provided map at the given path.
// If the path is not provided, it defaults to the Field's Path.
// If the path has only one element, it returns the value directly from the map.
// If the path has two or more elements, it recursively calls GetValueFromMap on the nested data.
// If the nested data is nil, it returns the nested data.
// If the path is invalid, it panics.
func (f *Field) GetValueFromMap(src map[string]interface{}, path ...string) any {
	if path == nil {
		path = f.Path
	}

	if len(path) == 1 {
		return src[path[0]]
	}
	if len(path) >= 2 {
		nested := parseNestedPath(src, path)
		if nested.data != nil {
			return f.GetValueFromMap(nested.data, path[1:]...)
		} else {
			return nested.data
		}
	}
	panic("well well, how did we get here?")
}

func (f *Field) ChRoot(root []string) {
	if root != nil {
		f.Path = append(root, f.Path...)
	}
}

func (f *Field) resolvePath() error {
	var err error
	f.Path = f.tag.Path // default to tag main path

	match := f.tag.findTypeMatch(f.Target)
	if match.Matches {
		err = f.checkPerTypePathNaming(match)
		if len(match.Path) > 0 {
			// replace tag main path with type-matching path
			f.Path = match.Path
		}
	} else {
		// no match found, but there are type-matching options set in this tag field
		// so set the field to be skipped
		f.Skip = true
	}

	return err
}

// checkPerTypePathNaming checks if the path specified in the TypeMatch
// is valid for the current Field. It returns an error if the path is not valid.
// The path is not valid if:
// 1. The match has a path and the main path in the Field tag does not have the MULTI_TYPE_NAME value.
// 2. The match has a path and the main path in the Field tag has more than one element.
func (f *Field) checkPerTypePathNaming(match TypeMatch) error {
	var err error
	var failedCheck bool

	matchHasPath := len(match.Path) > 0
	mainPathMatches := f.tag.Path[0] != MULTI_TYPE_NAME
	mainPathMaxLen := len(f.tag.Path) > 1

	if matchHasPath && mainPathMatches {
		failedCheck = true
	}

	if matchHasPath && mainPathMaxLen {
		failedCheck = true
	}

	if failedCheck {
		err = errors.New(ERROR_PER_TYPE_PATH_IS_NOT_VALID)
	}

	return err
}

// getFieldValue returns the value of the given field as an interface{} value.
//
// It handles various field types, including strings, integers, booleans, slices, maps, and structs.
// For slices, it recursively calls getFieldValue on each element.
// For maps, it recursively calls getFieldValue on each value.
// For structs, it populates a map[string]interface{} with the struct field values.
// If the field type is not supported, it panics with an error message.
//
// field: the reflect.Value of the field to get the value from.
// any: the value of the field.
func (f *Field) getFieldValue(field reflect.Value) any {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int()
	case reflect.Bool:
		return field.Bool()
	case reflect.Slice:
		list := []any{}
		for i := range field.Len() {
			list = append(list, f.getFieldValue(field.Index(i)))
		}
		return list
	case reflect.Map:
		iter := field.MapRange()
		result := map[string]any{}
		for iter.Next() {
			key := iter.Key().String()
			value := f.getFieldValue(iter.Value())
			result[key] = value
		}
		return result
	case reflect.Struct:
		result := map[string]any{}
		builder := &StructEncoder{}
		builder.typeRestrain = f.Target
		builder.generate(field.Interface(), result)
		return result
	case reflect.Ptr:
		return f.getFieldValue(field.Elem())
	default:
		msg := fmt.Sprintf("unsupported type: %s", field.Kind().String())
		panic(msg)
	}
}

type NestedPath struct {
	idx     int
	field   string
	isArray bool
	data    map[string]interface{}
}

func parseNestedPath(src map[string]interface{}, path []string) NestedPath {
	const matchArrayExp = "^(.*)\\[([0-9]*)\\]$"
	isPathArray := regexp.MustCompile(matchArrayExp).FindStringSubmatch(path[0])
	if isPathArray != nil {
		idx, _ := strconv.Atoi(isPathArray[2])
		fieldName := isPathArray[1]
		var data map[string]interface{}

		if src[fieldName] != nil {
			data = src[fieldName].([]interface{})[idx].(map[string]interface{})
		}
		return NestedPath{
			idx:     idx,
			field:   fieldName,
			isArray: true,
			data:    data,
		}
	} else {
		fieldName := path[0]
		var data map[string]interface{}

		srcData := src[fieldName]
		if srcData != nil {
			data = srcData.(map[string]interface{})
		}

		return NestedPath{
			field: fieldName,
			data:  data,
		}
	}
}
