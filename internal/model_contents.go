package internal

import (
	"encoding/json"
	"fmt"
)

type Contents struct {
	Bytes  *[]byte
	String *string
}

func (dst *Contents) UnmarshalJSON(data []byte) error {
	if string(data) == "" || string(data) == "{}" || string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		dst.String = &s
		return nil
	}
	return fmt.Errorf("data failed to match schemas in anyOf(Contents)")
}

func (src Contents) MarshalJSON() ([]byte, error) {
	if src.Bytes != nil {
		return json.Marshal(src.Bytes)
	}
	if src.String != nil {
		return json.Marshal(src.String)
	}
	return []byte("null"), nil
}

type NullableContents struct {
	value *Contents
	isSet bool
}

func (v NullableContents) Get() *Contents     { return v.value }
func (v *NullableContents) Set(val *Contents) { v.value = val; v.isSet = true }
func (v NullableContents) IsSet() bool        { return v.isSet }
func (v *NullableContents) Unset()            { v.value = nil; v.isSet = false }
func NewNullableContents(val *Contents) *NullableContents {
	return &NullableContents{value: val, isSet: true}
}
func (v NullableContents) MarshalJSON() ([]byte, error) { return json.Marshal(v.value) }
func (v *NullableContents) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
