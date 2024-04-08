// struct-marshal is an experimental utility to translate between two go structs using json encoding in the background.
//
// Intended to translate between API objects and the internal system definitions using "jsonpath" tag to map fields.
//
// A real world example would be to translate between Kubernetes API objects and a custom struct internal to your application.
//
// Example translating an internal object to a deployment:
//
//	import (
//		sm "github.com/ilexPar/struct-marshal/pkg"
//		apps "k8s.io/api/apps/v1"
//	)
//
//	type InternalObject struct {
//		Name   string `jsonpath:"metadata.name"`
//		Image  string `jsonpath:"spec.template.spec.containers[0].image"`
//		Memory string `jsonpath:"spec.template.spec.containers[0].resources.limits.memory"`
//	}
//
//	func main() {
//		src := &apps.Deployment{}
//		dst := &InternalObject{}
//		sm.StructUnmarshal(src, dst) // now dst should have been populated with the expected values from src
//	}
//
// Example translating a deployment to an internal object:
//
//		import (
//			sm "github.com/ilexPar/struct-marshal/pkg"
//			apps "k8s.io/api/apps/v1"
//		)
//
//		type InternalObject struct {
//			Name   string `jsonpath:"metadata.name"`
//			Image  string `jsonpath:"spec.template.spec.containers[0].image"`
//			Memory string `jsonpath:"spec.template.spec.containers[0].resources.limits.memory"`
//		}
//
//	func main() {
//		dst := &apps.Deployment{}
//		src := &InternalObject{
//			Name: "my-dpl",
//			Image: "nginx",
//			Memory: "128Mi",
//		}
//		sm.StructMmarshal(src, dst) // now dst should have been populated with the expected values from src
//	}
package pkg

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// StructMarshal marshals the given the jsonpath compatible source to a JSON byte slice, and then unmarshals it into the given destination interface{}.
// This function is intended to convert between system internal definitions and the destined API object.
func StructMarshal(src interface{}, dst interface{}) error {
	b, _ := MarshalJSONPath(src)
	err := json.Unmarshal(b, &dst)
	return err
}

// StructUnmarshal marshals the given source and then unmarshals into the jsonpath compatible destination.
// This function is intended to convert between the provided API object and the system internal definitions.
func StructUnmarshal(src interface{}, dst interface{}) error {
	b, _ := json.Marshal(src)
	err := UnmarshalJSONPath(b, dst)
	return err
}

// UnmarshalJSONPath unmarshals the given JSON byte slice into the provided destination interface.
// The destination interface must be a non-nil pointer. The function uses the "jsonpath" struct tags
// on the destination interface fields to map the JSON data to the appropriate fields.
func UnmarshalJSONPath(src []byte, dst interface{}) error {
	rd := reflect.ValueOf(dst)
	if rd.Kind() != reflect.Pointer || rd.IsNil() {
		return fmt.Errorf("dst must be a non-nil pointer")
	}

	srcData := map[string]interface{}{}
	dstData := map[string]interface{}{}
	json.Unmarshal(src, &srcData)

	v := reflect.ValueOf(dst).Elem()
	for i := range v.NumField() {
		tag := v.Type().Field(i).Tag.Get("jsonpath")
		if tag == "" {
			continue
		}
		tagParts := strings.Split(tag, ",")
		srcPath := strings.Split(tagParts[0], ".")
		value := extractValueFromPath(srcPath, srcData)
		if value == nil {
			continue
		}
		fieldName := v.Type().Field(i).Name
		dstPath := []string{strings.ToLower(fieldName)}
		setValueIntoPath(dstPath, value, dstData)
	}

	jsonData, _ := json.Marshal(dstData)
	return json.Unmarshal(jsonData, dst)
}

func setValueIntoPath(path []string, value any, dst map[string]interface{}) {
	if len(path) == 1 {
		dst[path[0]] = value
		return
	}
	setValueIntoPath(path[1:], value, dst[path[0]].(map[string]interface{}))
}

func extractValueFromPath(path []string, src map[string]interface{}) any {
	handler := func() any {
		return extractValueFromPath(path[1:], src[path[0]].(map[string]interface{}))
	}
	arrayHandler := func(idx int, field string, data any) any {
		if data == nil {
			return nil // ignore missing data
		}
		return extractValueFromPath(
			path[1:],
			data.([]interface{})[idx].(map[string]interface{}),
		)
	}

	if len(path) == 1 {
		return src[path[0]]
	}
	if len(path) >= 2 {
		return handleNestedPaths(path, src, handler, arrayHandler)
	}
	panic("well well, how did we get here?")
}

// MarshalJSONPath converts the given source interface{} to a JSON-encoded byte slice,
// using the JSON field tags defined on the source struct to map the fields to the
// resulting JSON object.
func MarshalJSONPath(src interface{}) ([]byte, error) {
	data := map[string]interface{}{}
	processJSONFields(src, data)
	return json.Marshal(data)
}

func processJSONFields(src interface{}, dst map[string]interface{}, parentPath ...string) {
	v := reflect.ValueOf(src)
	for i := range v.NumField() {
		tag := v.Type().Field(i).Tag.Get("jsonpath")
		if tag == "" {
			continue
		}
		tagParts := strings.Split(tag, ",")
		pathParts := strings.Split(tagParts[0], ".")
		if parentPath != nil {
			pathParts = append(parentPath, pathParts...)
		}
		field := v.Field(i)
		if field.IsZero() {
			continue
		}
		placeFieldValueIntoPath(pathParts, field, dst)
	}
}

func placeFieldValueIntoPath(path []string, field reflect.Value, dst map[string]interface{}) {
	handler := func() any {
		if dst[path[0]] == nil {
			dst[path[0]] = map[string]interface{}{}
		}
		placeFieldValueIntoPath(path[1:], field, dst[path[0]].(map[string]interface{}))
		return nil
	}
	arrayHandler := func(idx int, fieldName string, data any) any {
		if data == nil {
			dst[fieldName] = make([]map[string]interface{}, 1)
			dst[fieldName].([]map[string]interface{})[idx] = map[string]interface{}{}
		}
		placeFieldValueIntoPath(
			path[1:],
			field,
			dst[fieldName].([]map[string]interface{})[idx],
		)
		return nil
	}

	if len(path) == 1 {
		dst[path[0]] = getFieldValue(field)
		return
	}
	if len(path) >= 2 {
		handleNestedPaths(path, dst, handler, arrayHandler)
	}
}

func getFieldValue(field reflect.Value) any {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int:
		return field.Int()
	case reflect.Bool:
		return field.Bool()
	case reflect.Slice:
		list := []any{}
		for i := range field.Len() {
			list = append(list, getFieldValue(field.Index(i)))
		}
		return list
	case reflect.Map:
		iter := field.MapRange()
		result := map[string]any{}
		for iter.Next() {
			key := iter.Key().String()
			value := getFieldValue(iter.Value())
			result[key] = value
		}
		return result
	case reflect.Struct:
		result := map[string]any{}
		processJSONFields(field.Interface(), result)
		return result
	default:
		msg := fmt.Sprintf("unsupported type: %s", field.Kind().String())
		panic(msg)
	}
}

// Callback to handler array paths
// will provide array index, the corresponding field name holding the array, and the data that is being
// held in the array index
type arrayCb func(idx int, field string, data any) any

// Callback to handler non-array paths
type handlerCb func() any

func handleNestedPaths(
	path []string,
	src map[string]interface{},
	handler handlerCb,
	arrayHandler arrayCb,
) any {
	const matchArrayExp = "^(.*)\\[([0-9]*)\\]$"
	isPathArray := regexp.MustCompile(matchArrayExp).FindStringSubmatch(path[0])
	if isPathArray != nil {
		idx, _ := strconv.Atoi(isPathArray[2])
		fieldName := isPathArray[1]
		data := src[fieldName]
		return arrayHandler(idx, fieldName, data)
	} else {
		return handler()
	}
}
