package internal

import (
	"encoding/json"
	"bytes"
	"fmt"
)

// checks if the TrainRequest type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &TrainRequest{}

// TrainRequest represents the payload to train an encrypted index
type TrainRequest struct {
	IndexKey   string   `json:"index_key"`             // Required: hex string
	IndexName  string   `json:"index_name"`            // Required
	BatchSize  *int32   `json:"batch_size,omitempty"`  // Optional, default: 2048
	MaxIters   *int32   `json:"max_iters,omitempty"`   // Optional, default: 100
	Tolerance  *float64 `json:"tolerance,omitempty"`   // Optional, default: 1e-6
	MaxMemory  *int32   `json:"max_memory,omitempty"`  // Optional, default: 0
}

type _TrainRequest TrainRequest

func NewTrainRequest(indexKey string, indexName string) *TrainRequest {
	return &TrainRequest{
		IndexKey:  indexKey,
		IndexName: indexName,
	}
}

func (o TrainRequest) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return nil, err
	}
	return json.Marshal(toSerialize)
}

func (o TrainRequest) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{
		"index_key":  o.IndexKey,
		"index_name": o.IndexName,
	}
	if o.BatchSize != nil {
		toSerialize["batch_size"] = o.BatchSize
	}
	if o.MaxIters != nil {
		toSerialize["max_iters"] = o.MaxIters
	}
	if o.Tolerance != nil {
		toSerialize["tolerance"] = o.Tolerance
	}
	if o.MaxMemory != nil {
		toSerialize["max_memory"] = o.MaxMemory
	}
	return toSerialize, nil
}

func (o *TrainRequest) UnmarshalJSON(data []byte) error {
	required := []string{"index_key", "index_name"}
	var all map[string]interface{}
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for _, k := range required {
		if _, ok := all[k]; !ok {
			return fmt.Errorf("missing required field %q", k)
		}
	}

	var aux _TrainRequest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&aux); err != nil {
		return err
	}
	*o = TrainRequest(aux)
	return nil
}
