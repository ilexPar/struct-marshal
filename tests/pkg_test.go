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
	Name          string        `sm:"metadata.namefield"`
	Count         int           `sm:"config.somecount"`
	Flag          bool          `sm:"metadata.flag"`
	Nested        SystemNested  `sm:"config.somelist[0].config"`
	NestedPointer *SystemNested `sm:"config.somelist2[0].config"`
	ListedStuff   []string      `sm:"config.somelist[0].list"`
}

// Mock a struct taking advantage of multiple destination types
type SystemStructWithMultipleDestination struct {
	Name          string                               `sm:"metadata.namefield,types<APIObject|SecondaryAPIObject>"`         // multiple type checks using unified path
	Flag          bool                                 `sm:"+,types<APIObject:metadata.flag|SecondaryAPIObject:configflag>"` // multiple type checks using per type path
	Blackhole     string                               `sm:"dismiss,types<InexistentType>"`                                  // this should be ignored
	DismissNested NestedStructWithMultipleDestinations `sm:"->"`                                                             // let the nested fields declare the full destination path
}
type NestedStructWithMultipleDestinations struct {
	Direction string `sm:"child.direction,types<SecondaryAPIObject>"`
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
	SomeCount int             `json:"somecount"`
	SomeList  []APIListedObj  `json:"somelist"`
	SomeList2 []*APIListedObj `json:"somelist2"`
}

// Mock another struct that differs in structure from both our internal struct and the APIObject
// to test multiple types compatibility
type SecondaryAPIObjectChild struct {
	Direction string `json:"direction"`
}
type SecondaryAPIObject struct {
	Metadata   APIMetadata             `json:"metadata"`
	ConfigFlag bool                    `json:"configflag"`
	Child      SecondaryAPIObjectChild `json:"child"`
}

func TestStructUnmarshal(t *testing.T) {
	t.Run("should error when destination interface is nil", func(t *testing.T) {
		var emptyDst *SystemStruct
		err := pkg.Unmarshal(APIObject{}, emptyDst)
		assert.NotNil(t, err, "Expected error when destination interface is nil")
	})
	t.Run("should error when destination interface is not a pointer", func(t *testing.T) {
		var dst1 SystemStruct
		src := APIObject{}
		err := pkg.Unmarshal(src, dst1)
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
				SomeList2: []*APIListedObj{
					{
						Config: APIListedObjConfig{
							Direction: direction,
						},
					},
				},
			},
		}

		err := pkg.Unmarshal(src, &dst)

		assert.Nil(t, err)
		assert.Equal(t, dst.Name, name)
		assert.Equal(t, dst.Count, count)
		assert.Equal(t, dst.Flag, flag)
		assert.Equal(t, dst.Nested.Direction, direction)
		assert.Equal(t, dst.ListedStuff, list)
		assert.Equal(t, dst.Nested.DeeepNested.Direction, direction)
		assert.Equal(t, dst.NestedPointer.Direction, direction)
	})
	t.Run("should ignore fields that doesn't match type matching option", func(t *testing.T) {
		dst := struct {
			Name string `sm:"metadata.name,types<SecondaryAPIObject>"`
			Flag bool   `sm:"configflag,types<APIObject>"`
		}{}
		src := SecondaryAPIObject{
			Metadata: APIMetadata{
				NameField: "test",
			},
			ConfigFlag: true,
		}

		err := pkg.Unmarshal(src, &dst)

		assert.Nil(t, err)
		assert.False(t, dst.Flag)
	})
	t.Run("should accept a list of types to assert", func(t *testing.T) {
		dst := &SystemStructWithMultipleDestination{}
		src1 := SecondaryAPIObject{
			Metadata: APIMetadata{
				NameField: "test",
			},
		}
		src2 := APIObject{
			Metadata: APIMetadata{
				NameField: "test2",
			},
		}

		err1 := pkg.Unmarshal(src1, dst)
		assert.Nil(t, err1)
		assert.Equal(t, src1.Metadata.NameField, dst.Name)

		err2 := pkg.Unmarshal(src2, dst)
		assert.Nil(t, err2)
		assert.Equal(t, src2.Metadata.NameField, dst.Name)
	})
	t.Run("should accept per-type path when using type matching", func(t *testing.T) {
		dst1 := &SystemStructWithMultipleDestination{}
		dst2 := &SystemStructWithMultipleDestination{}
		src1 := SecondaryAPIObject{
			ConfigFlag: true,
		}
		src2 := APIObject{
			Metadata: APIMetadata{
				Flag: true,
			},
		}

		err1 := pkg.Unmarshal(src1, dst1)
		err2 := pkg.Unmarshal(src2, dst2)

		assert.Nil(t, err1)
		assert.Nil(t, err2)
		assert.Equal(t, src1.ConfigFlag, dst1.Flag)
		assert.Equal(t, src2.Metadata.Flag, dst2.Flag)
	})
	t.Run(
		"should error when using per-type path matching and main path is not '+'",
		func(t *testing.T) {
			type Destination struct {
				Flag bool `sm:"metadata.flag,types<APIObject:metadata.flag|SecondaryAPIObject:configflag>"`
			}
			dst := &Destination{}
			src := APIObject{
				Metadata: APIMetadata{
					Flag: true,
				},
			}

			err := pkg.Unmarshal(src, dst)
			assert.ErrorContains(t, err, pkg.ERROR_PER_TYPE_PATH_IS_NOT_VALID)
		},
	)
	t.Run("should use the full path of nested fields when using '->'", func(t *testing.T) {
		dst := &SystemStructWithMultipleDestination{}
		src := SecondaryAPIObject{
			Child: SecondaryAPIObjectChild{
				Direction: "up",
			},
		}

		err := pkg.Unmarshal(src, dst)
		assert.Nil(t, err)
		assert.Equal(t, src.Child.Direction, dst.DismissNested.Direction)
	})
	t.Run("do nothing when fields don't have the right tag", func(t *testing.T) {
		dst := struct {
			Name string `wrong:"metadata.namefield"`
		}{}
		src := SecondaryAPIObject{
			Metadata: APIMetadata{
				NameField: "test",
			},
		}

		err := pkg.Unmarshal(src, &dst)

		assert.Nil(t, err)
		assert.Empty(t, dst.Name)
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
			NestedPointer: &SystemNested{
				Direction: direction,
			},
			ListedStuff: list,
		}

		dst := &APIObject{}
		err := pkg.Marshal(src, dst)

		assert.Nil(t, err)
		assert.Equal(t, dst.Metadata.NameField, name)
		assert.Equal(t, dst.Metadata.Flag, flag)
		assert.Equal(t, dst.Config.SomeCount, count)
		assert.Equal(t, dst.Config.SomeList[0].Config.Direction, direction)
		assert.Equal(t, dst.Config.SomeList[0].List, list)
		assert.Equal(t, dst.Config.SomeList2[0].Config.Direction, direction)
	})
	t.Run("should ignore fields that doesn't match type matching option", func(t *testing.T) {
		src := struct {
			Name string `sm:"metadata.name,types<SecondaryAPIObject>"`
			Flag bool   `sm:"configflag,types<APIObject>"`
		}{
			Name: "test",
			Flag: true,
		}
		dst := SecondaryAPIObject{}

		err := pkg.Marshal(src, &dst)

		assert.Nil(t, err)
		assert.False(t, dst.ConfigFlag)
	})
	t.Run("should accept a list of types to assert", func(t *testing.T) {
		src := SystemStructWithMultipleDestination{
			Name: "test",
		}
		dst1 := &SecondaryAPIObject{}
		dst2 := &APIObject{}

		err1 := pkg.Marshal(src, dst1)
		assert.Nil(t, err1)
		assert.Equal(t, src.Name, dst1.Metadata.NameField)

		err2 := pkg.Marshal(src, dst2)
		assert.Nil(t, err2)
		assert.Equal(t, src.Name, dst2.Metadata.NameField)
	})
	t.Run("should accept per-type path when using type matching", func(t *testing.T) {
		src1 := SystemStructWithMultipleDestination{
			Flag: true,
		}
		dst1 := &SecondaryAPIObject{}
		dst2 := &APIObject{}

		err1 := pkg.Marshal(src1, dst1)
		err2 := pkg.Marshal(src1, dst2)

		assert.Nil(t, err1)
		assert.Nil(t, err2)
		assert.Equal(t, src1.Flag, dst1.ConfigFlag)
		assert.Equal(t, src1.Flag, dst2.Metadata.Flag)
	})
	t.Run(
		"should error when using per-type path matching and main path is not '+'",
		func(t *testing.T) {
			type Destination struct {
				Flag bool `sm:"metadata.flag,types<APIObject:metadata.flag|SecondaryAPIObject:configflag>"`
			}
			src := Destination{
				Flag: true,
			}
			dst := &APIObject{}

			err := pkg.Marshal(src, dst)
			assert.ErrorContains(t, err, pkg.ERROR_PER_TYPE_PATH_IS_NOT_VALID)
		},
	)
	t.Run("should use the full path of nested fields when using '->'", func(t *testing.T) {
		src := SystemStructWithMultipleDestination{
			DismissNested: NestedStructWithMultipleDestinations{
				Direction: "up",
			},
		}
		dst := &SecondaryAPIObject{}

		err := pkg.Marshal(src, dst)

		assert.Nil(t, err)
		assert.Equal(t, src.DismissNested.Direction, dst.Child.Direction)
	})
	t.Run("do nothing when fields don't have the right tag", func(t *testing.T) {
		src := struct {
			Name string `wrong:"metadata.namefield"`
		}{
			Name: "test",
		}
		dst := &SecondaryAPIObject{}

		err := pkg.Marshal(src, &dst)

		assert.Nil(t, err)
		assert.Empty(t, dst.Metadata.NameField)
	})
}
