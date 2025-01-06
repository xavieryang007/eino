/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compose

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudwego/eino/utils/generic"
)

func TestFieldMapping(t *testing.T) {
	t.Run("whole mapped to whole", func(t *testing.T) {
		m := []*Mapping{NewMapping("1")}

		out, err := mapFrom[string]("correct input", m)
		assert.NoError(t, err)
		assert.Equal(t, "correct input", out)

		out, err = mapFrom[string]("", m)
		assert.NoError(t, err)
		assert.Equal(t, "", out)

		_, err = mapFrom[string](1, m)
		assert.ErrorContains(t, err, "mismatched type")

		out1, err := mapFrom[any]("correct input", m)
		assert.NoError(t, err)
		assert.Equal(t, "correct input", out1)
	})

	t.Run("field mapped to whole", func(t *testing.T) {
		type up struct {
			F1 string
			f2 string
			F3 int
		}

		m := []*Mapping{NewMapping("1").FromField("F1")}

		out, err := mapFrom[string](&up{F1: "field1"}, m)
		assert.NoError(t, err)
		assert.Equal(t, "field1", out)

		out, err = mapFrom[string](&up{F1: ""}, m)
		assert.NoError(t, err)
		assert.Equal(t, "", out)

		out, err = mapFrom[string](up{F1: "field1"}, m)
		assert.NoError(t, err)
		assert.Equal(t, "field1", out)

		m[0].fromField = "f2"
		_, err = mapFrom[string](&up{f2: "f2"}, m)
		assert.ErrorContains(t, err, "not exported")

		m[0].fromField = "field3"
		_, err = mapFrom[string](&up{F3: 3}, m)
		assert.ErrorContains(t, err, "fromField not found")

		m[0].fromField = "F3"
		_, err = mapFrom[string](&up{F3: 3}, m)
		assert.ErrorContains(t, err, "mismatched type")

		m = []*Mapping{NewMapping("1").FromField("F1")}
		out1, err := mapFrom[any](&up{F1: "field1"}, m)
		assert.NoError(t, err)
		assert.Equal(t, "field1", out1)
	})

	t.Run("map key mapped to whole", func(t *testing.T) {
		m := []*Mapping{NewMapping("1").FromMapKey("key1")}

		out, err := mapFrom[string](map[string]string{"key1": "value1"}, m)
		assert.NoError(t, err)
		assert.Equal(t, "value1", out)

		out, err = mapFrom[string](map[string]string{"key1": ""}, m)
		assert.NoError(t, err)
		assert.Equal(t, "", out)

		out, err = mapFrom[string](map[string]string{"key2": "value2"}, m)
		assert.ErrorContains(t, err, "fromMapKey not found")

		out, err = mapFrom[string](map[string]int{"key1": 1}, m)
		assert.ErrorContains(t, err, "mismatched type")

		type mock string
		out, err = mapFrom[string](map[mock]string{"key1": "value1"}, m)
		assert.ErrorContains(t, err, "not a map with string key")

		out1, err := mapFrom[any](map[string]string{"key1": "value1"}, m)
		assert.NoError(t, err)
		assert.Equal(t, "value1", out1)
	})

	t.Run("whole mapped to field", func(t *testing.T) {
		type down struct {
			F1 string
			f3 string
		}

		m := []*Mapping{NewMapping("1").ToField("F1")}

		out, err := mapFrom[down]("from", m)
		assert.NoError(t, err)
		assert.Equal(t, down{F1: "from"}, out)

		out, err = mapFrom[down](1, m)
		assert.ErrorContains(t, err, "mismatched type")

		m[0].toField = "f2"
		_, err = mapFrom[down]("from", m)
		assert.ErrorContains(t, err, "toField not found")

		m[0].toField = "f3"
		_, err = mapFrom[down]("from", m)
		assert.ErrorContains(t, err, "not exported")

		m = []*Mapping{NewMapping("1").ToField("F1")}
		out1, err := mapFrom[*down]("from", m)
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: "from"}, out1)
	})

	t.Run("whole mapped to map key", func(t *testing.T) {
		m := []*Mapping{NewMapping("1").ToMapKey("key1")}

		out, err := mapFrom[map[string]string]("from", m)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"key1": "from"}, out)
		out, err = mapFrom[map[string]string](1, m)
		assert.ErrorContains(t, err, "mismatched type")

		type mockKey string
		_, err = mapFrom[map[mockKey]string]("from", m)
		assert.ErrorContains(t, err, "mapping has toMapKey but output is not a map with string key")
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

		m := []*Mapping{NewMapping("1").FromField("F1").ToField("F1")}

		out, err := mapFrom[*down](&up{F1: &inner{in: "in"}}, m)
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: &inner{in: "in"}}, out)
	})

	t.Run("field mapped to map key", func(t *testing.T) {
		type up struct {
			F1 []string
		}

		m := []*Mapping{NewMapping("1").FromField("F1").ToMapKey("key1")}

		out, err := mapFrom[map[string]any](&up{F1: []string{"in"}}, m)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"key1": []string{"in"}}, out)
	})

	t.Run("map key mapped to map key", func(t *testing.T) {
		m := []*Mapping{NewMapping("1").FromMapKey("key1").ToMapKey("key2")}

		out, err := mapFrom[map[string]any](map[string]any{"key1": "value1"}, m)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"key2": "value1"}, out)
	})

	t.Run("map key mapped to field", func(t *testing.T) {
		type down struct {
			F1 io.Reader
		}

		m := []*Mapping{NewMapping("1").FromMapKey("key1").ToField("F1")}

		out, err := mapFrom[*down](map[string]any{"key1": &bytes.Buffer{}}, m)
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: &bytes.Buffer{}}, out)
	})

	t.Run("multiple mappings", func(t *testing.T) {
		type down struct {
			F1 string
			F2 int
		}

		m := []*Mapping{
			NewMapping("1").FromMapKey("key1").ToField("F1"),
			NewMapping("1").FromMapKey("key2").ToField("F2"),
		}

		out, err := mapFrom[*down](map[string]any{"key1": "v1", "key2": 2}, m)
		assert.NoError(t, err)
		assert.Equal(t, &down{F1: "v1", F2: 2}, out)

		m[0].fromNodeKey = "different_upper"
		out, err = mapFrom[*down](map[string]any{"key1": "v1", "key2": 2}, m)
		assert.ErrorContains(t, err, "multiple mappings from the same node have different keys")

		m[0].fromNodeKey = "1"
		m[0].toField = ""
		out, err = mapFrom[*down](map[string]any{"key1": "v1", "key2": 2}, m)
		assert.ErrorContains(t, err, "one of the mapping maps to entire input, conflict")

		m = []*Mapping{}
		out, err = mapFrom[*down](map[string]any{"key1": "v1", "key2": 2}, m)
		assert.ErrorContains(t, err, "mapper has no Mappings")
	})

	t.Run("invalid mapping", func(t *testing.T) {
		m := []*Mapping{NewMapping("1").FromMapKey("key1").FromField("F1")}
		_, err := mapFrom[string](map[string]any{"key1": "v1", "key2": 2}, m)
		assert.ErrorContains(t, err, "mapping has both fromField and fromMapKey")

		m = []*Mapping{NewMapping("1").ToMapKey("key1").ToField("F1")}
		_, err = mapFrom[string]("input", m)
		assert.ErrorContains(t, err, "mapping has both toField and toMapKey")

		m = []*Mapping{NewMapping("1").FromField("F1")}
		_, err = mapFrom[string](generic.PtrOf("input"), m)
		assert.ErrorContains(t, err, "mapping has fromField but input is not struct or struct ptr")

		m = []*Mapping{NewMapping("1").FromMapKey("key1")}
		_, err = mapFrom[string](generic.PtrOf("input"), m)
		assert.ErrorContains(t, err, "mapping has FromKey but input is not a map")

		m = []*Mapping{NewMapping("1").ToField("F1")}
		_, err = mapFrom[string]("input", m)
		assert.ErrorContains(t, err, "mapping has toField but output is not a struct")

		m = []*Mapping{NewMapping("1").ToMapKey("key1")}
		_, err = mapFrom[string]("input", m)
		assert.ErrorContains(t, err, "mapping has toMapKey but output is not a map")
	})
}
