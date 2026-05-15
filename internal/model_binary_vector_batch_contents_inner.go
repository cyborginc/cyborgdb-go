package internal

import (
	"encoding/json"
	"fmt"
)

type BinaryVectorBatchContentsInner struct {
	Bytes  *[]byte
	String *string
}

func (dst *BinaryVectorBatchContentsInner) UnmarshalJSON(data []byte) error {
	if string(data) == "" || string(data) == "{}" || string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		dst.String = &s
		return nil
	}
	return fmt.Errorf("data failed to match schemas in anyOf(BinaryVectorBatchContentsInner)")
}

func (src BinaryVectorBatchContentsInner) MarshalJSON() ([]byte, error) {
	if src.Bytes != nil {
		return json.Marshal(src.Bytes)
	}
	if src.String != nil {
		return json.Marshal(src.String)
	}
	return []byte("null"), nil
}

type NullableBinaryVectorBatchContentsInner struct {
	value *BinaryVectorBatchContentsInner
	isSet bool
}

func (v NullableBinaryVectorBatchContentsInner) Get() *BinaryVectorBatchContentsInner { return v.value }
func (v *NullableBinaryVectorBatchContentsInner) Set(val *BinaryVectorBatchContentsInner) { v.value = val; v.isSet = true }
func (v NullableBinaryVectorBatchContentsInner) IsSet() bool { return v.isSet }
func (v *NullableBinaryVectorBatchContentsInner) Unset() { v.value = nil; v.isSet = false }
func NewNullableBinaryVectorBatchContentsInner(val *BinaryVectorBatchContentsInner) *NullableBinaryVectorBatchContentsInner { return &NullableBinaryVectorBatchContentsInner{value: val, isSet: true} }
func (v NullableBinaryVectorBatchContentsInner) MarshalJSON() ([]byte, error) { return json.Marshal(v.value) }
func (v *NullableBinaryVectorBatchContentsInner) UnmarshalJSON(src []byte) error { v.isSet = true; return json.Unmarshal(src, &v.value) }
