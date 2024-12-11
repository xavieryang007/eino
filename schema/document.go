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

package schema

const (
	metaDataKeySubIndexes      = "_sub_indexes"
	metaDataKeyScore           = "_score"
	metaDataKeyVikingExtraInfo = "_viking_extra_info"
	metaDataKeyVikingDSL       = "_viking_dsl"
	metaDataKeyVector          = "_vector"
)

// Document is a piece of text with metadata.
type Document struct {
	// ID is the unique identifier of the document.
	ID string `json:"id"`
	// Content is the content of the document.
	Content string `json:"content"`
	// MetaData is the metadata of the document, can be used to store extra information.
	MetaData map[string]any `json:"meta_data"`
}

// String returns the content of the document.
func (d *Document) String() string {
	return d.Content
}

// WithSubIndexes sets the sub indexes of the document.
// can use doc.SubIndexes() to get the sub indexes, useful for search engine to use sub indexes to search.
func (d *Document) WithSubIndexes(indexes []string) *Document {
	if d.MetaData == nil {
		d.MetaData = make(map[string]any)
	}

	d.MetaData[metaDataKeySubIndexes] = indexes

	return d
}

// SubIndexes returns the sub indexes of the document.
// can use doc.WithSubIndexes() to set the sub indexes.
func (d *Document) SubIndexes() []string {
	if d.MetaData == nil {
		return nil
	}

	indexes, ok := d.MetaData[metaDataKeySubIndexes].([]string)
	if ok {
		return indexes
	}

	return nil
}

// WithScore sets the score of the document.
// can use doc.Score() to get the score.
func (d *Document) WithScore(score float64) *Document {
	if d.MetaData == nil {
		d.MetaData = make(map[string]any)
	}

	d.MetaData[metaDataKeyScore] = score

	return d
}

// Score returns the score of the document.
// can use doc.WithScore() to set the score.
func (d *Document) Score() float64 {
	if d.MetaData == nil {
		return 0
	}

	score, ok := d.MetaData[metaDataKeyScore].(float64)
	if ok {
		return score
	}

	return 0
}

// WithVikingExtraInfo sets the extra info of the document.
// can use doc.VikingExtraInfo() to get the extra info.
func (d *Document) WithVikingExtraInfo(extraInfo string) *Document {
	if d.MetaData == nil {
		d.MetaData = make(map[string]any)
	}

	d.MetaData[metaDataKeyVikingExtraInfo] = extraInfo

	return d
}

// VikingExtraInfo returns the extra info of the document.
// can use doc.WithVikingExtraInfo() to set the extra info.
func (d *Document) VikingExtraInfo() string {
	if d.MetaData == nil {
		return ""
	}

	extraInfo, ok := d.MetaData[metaDataKeyVikingExtraInfo].(string)
	if ok {
		return extraInfo
	}

	return ""
}

// WithVikingDSLInfo sets the dsl info of the document.
// can use doc.VikingDSLInfo() to get the dsl info.
func (d *Document) WithVikingDSLInfo(dslInfo map[string]any) *Document {
	if d.MetaData == nil {
		d.MetaData = make(map[string]any)
	}

	d.MetaData[metaDataKeyVikingDSL] = dslInfo

	return d
}

// VikingDSLInfo returns the dsl info of the document.
// can use doc.WithVikingDSLInfo() to set the dsl info.
func (d *Document) VikingDSLInfo() map[string]any {
	if d.MetaData == nil {
		return nil
	}

	dslInfo, ok := d.MetaData[metaDataKeyVikingDSL].(map[string]any)
	if ok {
		return dslInfo
	}

	return nil
}

// WithVector sets the vector of the document.
// can use doc.Vector() to get the vector.
func (d *Document) WithVector(vector []float64) *Document {
	if d.MetaData == nil {
		d.MetaData = make(map[string]any)
	}

	d.MetaData[metaDataKeyVector] = vector

	return d
}

// Vector returns the vector of the document.
// can use doc.WithVector() to set the vector.
func (d *Document) Vector() []float64 {
	if d.MetaData == nil {
		return nil
	}

	vector, ok := d.MetaData[metaDataKeyVector].([]float64)
	if ok {
		return vector
	}

	return nil
}
