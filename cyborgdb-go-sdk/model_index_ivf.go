package cyborgdb

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// IndexIVFModel configures an IVF (Inverted File) index.
type IndexIVFModel struct {
	Dimension int32  `json:"dimension"`
	NLists    int32  `json:"n_lists"`
	Metric    string `json:"metric"`
	Type      string `json:"type"` // default: "ivf"
}

type _IndexIVFModel IndexIVFModel

// NewIndexIVFModel instantiates a new IVF index model with required fields.
func NewIndexIVFModel(dimension int32, metric string, nLists int32) *IndexIVFModel {
	return &IndexIVFModel{
		Dimension: dimension,
		Metric:    metric,
		NLists:    nLists,
		Type:      "ivf",
	}
}

// NewIndexIVFModelWithDefaults instantiates a new IVF index model with default type set.
func NewIndexIVFModelWithDefaults() *IndexIVFModel {
	return &IndexIVFModel{
		Type: "ivf",
	}
}

func (o IndexIVFModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return nil, err
	}
	return json.Marshal(toSerialize)
}

func (o IndexIVFModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{
		"dimension": o.Dimension,
		"metric":    o.Metric,
		"n_lists":   o.NLists,
		"type":      o.Type,
	}
	return toSerialize, nil
}

func (o *IndexIVFModel) UnmarshalJSON(data []byte) error {
	requiredProps := []string{"dimension", "metric", "n_lists"}
	var allProps map[string]interface{}
	err := json.Unmarshal(data, &allProps)
	if err != nil {
		return err
	}

	for _, prop := range requiredProps {
		if _, found := allProps[prop]; !found {
			return fmt.Errorf("required field %s is missing", prop)
		}
	}

	var temp _IndexIVFModel
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&temp); err != nil {
		return err
	}

	*o = IndexIVFModel(temp)

	if o.Type == "" {
		o.Type = "ivf"
	}

	return nil
}

// Nullable wrapper
type NullableIndexIVFModel struct {
	value *IndexIVFModel
	isSet bool
}

func (v NullableIndexIVFModel) Get() *IndexIVFModel {
	return v.value
}

func (v *NullableIndexIVFModel) Set(val *IndexIVFModel) {
	v.value = val
	v.isSet = true
}

func (v NullableIndexIVFModel) IsSet() bool {
	return v.isSet
}

func (v *NullableIndexIVFModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableIndexIVFModel(val *IndexIVFModel) *NullableIndexIVFModel {
	return &NullableIndexIVFModel{value: val, isSet: true}
}

func (v NullableIndexIVFModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableIndexIVFModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

func (m *IndexIVFModel) isIndexModel() {}
