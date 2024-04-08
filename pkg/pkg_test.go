package pkg_test

import (
	"testing"

	"github.com/ilexPar/struct-marshal/pkg"

	cmp "github.com/google/go-cmp/cmp"
)

// Mock a struct internal to an application
type SystemNested struct {
	Direction string `jsonpath:"direction"`
}
type SystemStruct struct {
	Name        string       `jsonpath:"metadata.namefield"`
	Count       int          `jsonpath:"config.somecount"`
	Flag        bool         `jsonpath:"metadata.flag"`
	Nested      SystemNested `jsonpath:"config.somelist[0].config"`
	ListedStuff []string     `jsonpath:"config.somelist[0].list"`
}

// Mock a struct that differs in structure from our internal struct, probably belonging to another API
type APIObject struct {
	Metadata APIMetadata `json:"metadata"`
	Config   APIConfig   `json:"config"`
}
type APIListedObjConfig struct {
	Direction string `json:"direction"`
}
type APIListedObj struct {
	List   []string           `json:"list"`
	Config APIListedObjConfig `json:"config"`
}
type APIMetadata struct {
	NameField string `json:"namefield"`
	Flag      bool   `json:"flag"`
}
type APIConfig struct {
	SomeCount int            `json:"somecount"`
	SomeList  []APIListedObj `json:"somelist"`
}

func TestStructUnmarshal(t *testing.T) {
	t.Run("should error when destination interface is nil", func(t *testing.T) {
		var emptyDst *SystemStruct
		err := pkg.StructUnmarshal(APIObject{}, emptyDst)
		if err == nil {
			t.Error("Expected error when destination interface is nil")
		}
	})
	t.Run("should error when destination interface is not a pointer", func(t *testing.T) {
		var dst1 SystemStruct
		err := pkg.UnmarshalJSONPath([]byte{}, dst1)
		if err == nil {
			t.Error("Expected error when destination interface is not a pointer")
		}
	})
	t.Run("correct unmarshal", func(t *testing.T) {
		name := "test"
		count := 999
		flag := true
		dst := SystemStruct{}
		direction := "up"
		list := []string{"a", "b", "c"}

		src := APIObject{
			Metadata: APIMetadata{
				NameField: name,
				Flag:      flag,
			},
			Config: APIConfig{
				SomeCount: count,
				SomeList: []APIListedObj{
					{
						List: list,
						Config: APIListedObjConfig{
							Direction: direction,
						},
					},
				},
			},
		}

		if err := pkg.StructUnmarshal(src, &dst); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if dst.Name != name || dst.Count != count || dst.Flag != flag ||
			dst.Nested.Direction != "up" || !cmp.Equal(dst.ListedStuff, list) {
			t.Errorf("Expected %v, got %v", src, dst)
		}
	})
}

func TestStructMarshal(t *testing.T) {
	t.Run("should marshal source into destination", func(t *testing.T) {
		name := "test"
		count := 999
		flag := true
		direction := "up"
		list := []string{"a", "b", "c"}
		src := SystemStruct{
			Name:  name,
			Count: count,
			Flag:  flag,
			Nested: SystemNested{
				Direction: direction,
			},
			ListedStuff: list,
		}

		dst := &APIObject{}

		if err := pkg.StructMarshal(src, dst); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if dst.Metadata.NameField != name || dst.Metadata.Flag != flag ||
			dst.Config.SomeCount != count || dst.Config.SomeList[0].Config.Direction != direction ||
			!cmp.Equal(dst.Config.SomeList[0].List, list) {
			t.Errorf("Expected %v, got %v", src, dst)
		}

	})
}
