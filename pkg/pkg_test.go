package pkg_test

import (
	"testing"

	"github.com/ilexPar/struct-marshal/pkg"

	"github.com/stretchr/testify/assert"
)

// Mock a struct internal to an application
type SystemDeepNested struct {
	Direction string `sm:"direction2"`
}
type SystemNested struct {
	Direction   string           `sm:"direction"`
	DeeepNested SystemDeepNested `sm:"deepnested"`
}
type SystemStruct struct {
	Name        string       `sm:"metadata.namefield"`
	Count       int          `sm:"config.somecount"`
	Flag        bool         `sm:"metadata.flag"`
	Nested      SystemNested `sm:"config.somelist[0].config"`
	ListedStuff []string     `sm:"config.somelist[0].list"`
}

// Mock a struct that differs in structure from our internal struct, probably belonging to another API
type APIObject struct {
	Metadata APIMetadata `json:"metadata"`
	Config   APIConfig   `json:"config"`
}
type APIDeepNested struct {
	Direction2 string `json:"direction2"`
}
type APIListedObjConfig struct {
	DeepNested APIDeepNested `json:"deepnested"`
	Direction  string        `json:"direction"`
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
		assert.NotNil(t, err, "Expected error when destination interface is nil")
	})
	t.Run("should error when destination interface is not a pointer", func(t *testing.T) {
		var dst1 SystemStruct
		err := pkg.UnmarshalJSONPath([]byte{}, dst1)
		assert.NotNil(t, err, "Expected error when destination interface is not a pointer")
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
							DeepNested: APIDeepNested{
								Direction2: direction,
							},
						},
					},
				},
			},
		}

		err := pkg.StructUnmarshal(src, &dst)

		assert.Nil(t, err)
		assert.Equal(t, dst.Name, name)
		assert.Equal(t, dst.Count, count)
		assert.Equal(t, dst.Flag, flag)
		assert.Equal(t, dst.Nested.Direction, direction)
		assert.Equal(t, dst.ListedStuff, list)
		assert.Equal(t, dst.Nested.DeeepNested.Direction, direction)
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
		err := pkg.StructMarshal(src, dst)

		assert.Nil(t, err)
		assert.Equal(t, dst.Metadata.NameField, name)
		assert.Equal(t, dst.Metadata.Flag, flag)
		assert.Equal(t, dst.Config.SomeCount, count)
		assert.Equal(t, dst.Config.SomeList[0].Config.Direction, direction)
		assert.Equal(t, dst.Config.SomeList[0].List, list)
	})
}
