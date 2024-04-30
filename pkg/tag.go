package pkg

import (
	"reflect"
	"regexp"
	"strings"
)

type TypeMatch struct {
	Path []string
	Name string
}

type TagOpts struct {
	MatchTypes []TypeMatch
}

type FieldTag struct {
	RawOpts []string
	RawPath string
	Path    []string
	Opts    TagOpts
}

// check naming convention when using "type matching" tag option
// return values can either be:
// - nil if no match is found but there are type-matching options set in this tag field
// - an empty TypeMatch if no match is found and no type-matching options set in this tag field
// - the TypeMatch description if a match is found
func (t *FieldTag) findTypeMatch(typeName string) *TypeMatch {
	matched := &TypeMatch{}
	if len(t.Opts.MatchTypes) > 0 && typeName != "" {
		matched = nil
		for _, match := range t.Opts.MatchTypes {
			if match.Name == typeName {
				matched = &match
				break
			}
		}
	}
	return matched
}

// parseTag parses a field tag string into a FieldTag struct. The field tag string
// is expected to be in the format "path,opt1,opt2,...". The path is split on
// periods to create the Path field of the FieldTag struct. The remaining comma-
// separated values are parsed into the Opts field of the FieldTag struct.
//
// If the field tag string is empty, the function returns a FieldTag with skip
// set to true.
func parseTag(field reflect.StructField) (FieldTag, bool) {
	var skip bool
	var tag FieldTag
	rawString := field.Tag.Get(FIELD_TAG_KEY)
	if rawString == "" {
		return tag, true
	}

	tagParts := strings.Split(rawString, ",")
	tag.Path = strings.Split(tagParts[0], ".")
	tag.RawOpts = tagParts[1:]

	if len(tagParts) > 1 {
		tag.Opts = parseTagOpts(tagParts[1:])
	}

	return tag, skip
}

// parseTagOpts parses a list of tag options into a TagOpts struct.
// The options are expected to be in the format "opt1,opt2,...".
// The resulting TagOpts will contain a list of TypeMatch structs, one for each type option.
func parseTagOpts(opts []string) TagOpts {
	options := TagOpts{}
	matchTypeRegEx := regexp.MustCompile(TYPE_OPTS_REGEX)
	for _, opt := range opts {
		typeMatches := matchTypeRegEx.FindStringSubmatch(opt)
		if len(typeMatches) > 0 {
			options.MatchTypes = parseTypeMatches(typeMatches[1])
		}
	}
	return options
}

// parseTypeMatches parses a string representation of type matches into a slice of TypeMatch structs.
// The input string is expected to be in the format "typeName1:fieldPath1|typeName2:fieldPath2|...".
// Each type match consists of a type name and an optional field path, separated by a colon.
// The field paths are split on periods to create the Path field of the TypeMatch struct.
// The resulting slice contains one TypeMatch struct for each type match in the input string.
func parseTypeMatches(data string) []TypeMatch {
	matchDescription := []TypeMatch{}
	parts := strings.Split(data, "|")
	for _, typeOpt := range parts {
		var fieldPath []string
		typeParts := strings.Split(typeOpt, ":")
		typeName := typeParts[0]
		if len(typeParts) > 1 {
			fieldPath = strings.Split(typeParts[1], ".")
		}
		matchDescription = append(matchDescription, TypeMatch{
			Name: typeName,
			Path: fieldPath,
		})
	}

	return matchDescription
}
