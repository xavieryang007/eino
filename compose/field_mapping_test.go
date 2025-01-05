package compose

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldMapping(t *testing.T) {
	t.Run("whole mapped to whole", func(t *testing.T) {
		m := &fieldMapper[string]{
			mappings: []Mapping{
				{
					From: "upper",
				},
			},
		}

		out, err := m.mapFrom("correct input")
		assert.NoError(t, err)
		assert.Equal(t, "correct input", out)

		out, err = m.mapFrom("")
		assert.NoError(t, err)
		assert.Equal(t, "", out)

		_, err = m.mapFrom(1)
		assert.ErrorContains(t, err, "mismatched type")

		m1 := &fieldMapper[any]{
			mappings: []Mapping{
				{
					From: "upper",
				},
			},
		}

		out1, err := m1.mapFrom("correct input")
		assert.NoError(t, err)
		assert.Equal(t, "correct input", out1)
	})

	t.Run("field mapped to whole", func(t *testing.T) {
		type up struct {
			F1 string
			f2 string
			F3 int
		}

		m := &fieldMapper[string]{
			mappings: []Mapping{
				{
					From:      "upper",
					FromField: "F1",
				},
			},
		}

		out, err := m.mapFrom(&up{F1: "field1"})
		assert.NoError(t, err)
		assert.Equal(t, "field1", out)

		out, err = m.mapFrom(&up{F1: ""})
		assert.NoError(t, err)
		assert.Equal(t, "", out)

		out, err = m.mapFrom(up{F1: "field1"})
		assert.NoError(t, err)
		assert.Equal(t, "field1", out)

		m.mappings[0].FromField = "f2"
		_, err = m.mapFrom(&up{f2: "f2"})
		assert.ErrorContains(t, err, "not exported")

		m.mappings[0].FromField = "field3"
		_, err = m.mapFrom(&up{F3: 3})
		assert.ErrorContains(t, err, "FromField not found")

		m.mappings[0].FromField = "F3"
		_, err = m.mapFrom(&up{F3: 3})
		assert.ErrorContains(t, err, "mismatched type")

		m1 := &fieldMapper[any]{
			mappings: []Mapping{
				{
					From:      "upper",
					FromField: "F1",
				},
			},
		}

		out1, err := m1.mapFrom(&up{F1: "field1"})
		assert.NoError(t, err)
		assert.Equal(t, "field1", out1)
	})

	t.Run("map key mapped to whole", func(t *testing.T) {
		m := &fieldMapper[string]{
			mappings: []Mapping{
				{
					From:       "upper",
					FromMapKey: "key1",
				},
			},
		}

		out, err := m.mapFrom(map[string]string{"key1": "value1"})
		assert.NoError(t, err)
		assert.Equal(t, "value1", out)

		out, err = m.mapFrom(map[string]string{"key1": ""})
		assert.NoError(t, err)
		assert.Equal(t, "", out)

		out, err = m.mapFrom(map[string]string{"key2": "value2"})
		assert.ErrorContains(t, err, "FromMapKey not found")

		out, err = m.mapFrom(map[string]int{"key1": 1})
		assert.ErrorContains(t, err, "mismatched type")

		type mock string
		out, err = m.mapFrom(map[mock]string{"key1": "value1"})
		assert.ErrorContains(t, err, "not a map with string key")

		m1 := &fieldMapper[any]{
			mappings: []Mapping{
				{
					From:       "upper",
					FromMapKey: "key1",
				},
			},
		}

		out1, err := m1.mapFrom(map[string]string{"key1": "value1"})
		assert.NoError(t, err)
		assert.Equal(t, "value1", out1)
	})

	t.Run("whole mapped to field", func(t *testing.T) {
		type down struct {
			F1 string
			f3 string
		}

		m := &fieldMapper[down]{
			mappings: []Mapping{
				{
					From:    "upper",
					ToField: "F1",
				},
			},
		}

		out, err := m.mapFrom("from")
		assert.NoError(t, err)
		assert.Equal(t, down{F1: "from"}, out)

		out, err = m.mapFrom(1)
		assert.ErrorContains(t, err, "mismatched type")

		m.mappings[0].ToField = "f2"
		_, err = m.mapFrom("from")
		assert.ErrorContains(t, err, "ToField not found")

		m.mappings[0].ToField = "f3"
		_, err = m.mapFrom("from")
		assert.ErrorContains(t, err, "not exported")

		m1 := &fieldMapper[*down]{
			mappings: []Mapping{
				{
					From:    "upper",
					ToField: "F1",
				},
			},
		}

		out1, err := m1.mapFrom("from")
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: "from"}, out1)
	})

	t.Run("whole mapped to map key", func(t *testing.T) {
		m := &fieldMapper[map[string]string]{
			mappings: []Mapping{
				{
					From:     "upper",
					ToMapKey: "key1",
				},
			},
		}

		out, err := m.mapFrom("from")
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"key1": "from"}, out)
		out, err = m.mapFrom(1)
		assert.ErrorContains(t, err, "mismatched type")

		type mockKey string
		m1 := &fieldMapper[map[mockKey]string]{
			mappings: []Mapping{
				{
					From:     "upper",
					ToMapKey: "key1",
				},
			},
		}

		_, err = m1.mapFrom("from")
		assert.ErrorContains(t, err, "mapping has ToMapKey but output is not a map with string key")
	})

	t.Run("field mapped to field", func(t *testing.T) {
		type inner struct {
			in string
		}

		type up struct {
			F1 *inner
		}

		type down struct {
			F1 *inner
		}

		m := &fieldMapper[*down]{
			mappings: []Mapping{
				{
					From:      "upper",
					FromField: "F1",
					ToField:   "F1",
				},
			},
		}

		out, err := m.mapFrom(&up{F1: &inner{in: "in"}})
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: &inner{in: "in"}}, out)
	})

	t.Run("field mapped to map key", func(t *testing.T) {
		type up struct {
			F1 []string
		}

		m := &fieldMapper[map[string]any]{
			mappings: []Mapping{
				{
					From:      "upper",
					FromField: "F1",
					ToMapKey:  "key1",
				},
			},
		}

		out, err := m.mapFrom(&up{F1: []string{"in"}})
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"key1": []string{"in"}}, out)
	})

	t.Run("map key mapped to map key", func(t *testing.T) {
		m := &fieldMapper[map[string]any]{
			mappings: []Mapping{
				{
					From:       "upper",
					FromMapKey: "key1",
					ToMapKey:   "key2",
				},
			},
		}

		out, err := m.mapFrom(map[string]any{"key1": "value1"})
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"key2": "value1"}, out)
	})

	t.Run("map key mapped to field", func(t *testing.T) {
		type down struct {
			F1 io.Reader
		}

		m := &fieldMapper[*down]{
			mappings: []Mapping{
				{
					From:       "upper",
					FromMapKey: "key1",
					ToField:    "F1",
				},
			},
		}

		out, err := m.mapFrom(map[string]any{"key1": &bytes.Buffer{}})
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: &bytes.Buffer{}}, out)
	})

	t.Run("multiple mappings", func(t *testing.T) {
		type down struct {
			F1 string
			F2 int
		}

		m := &fieldMapper[*down]{
			mappings: []Mapping{
				{
					From:       "upper",
					FromMapKey: "key1",
					ToField:    "F1",
				},
				{
					From:       "upper",
					FromMapKey: "key2",
					ToField:    "F2",
				},
			},
		}

		out, err := m.mapFrom(map[string]any{"key1": "v1", "key2": 2})
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: "v1", F2: 2}, out)

		m.mappings[0].From = "different_upper"
		out, err = m.mapFrom(map[string]any{"key1": "v1", "key2": 2})
		assert.ErrorContains(t, err, "multiple mappings from the same node have different keys")

		m.mappings[0].From = "upper"
		m.mappings[0].ToField = ""
		out, err = m.mapFrom(map[string]any{"key1": "v1", "key2": 2})
		assert.ErrorContains(t, err, "one of the mapping maps to entire input, conflict")

		m.mappings = []Mapping{}
		out, err = m.mapFrom(map[string]any{"key1": "v1", "key2": 2})
		assert.ErrorContains(t, err, "mapper has no Mappings")
	})
}
