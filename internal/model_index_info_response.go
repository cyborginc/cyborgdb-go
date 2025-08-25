package internal

// IndexInfoResponseModel represents the response from the describe index endpoint
type IndexInfoResponseModel struct {
	// Name of the index
	IndexName string `json:"index_name"`
	
	// Type of the index (ivf, ivfpq, ivfflat)
	IndexType string `json:"index_type"`
	
	// Whether the index has been trained
	IsTrained bool `json:"is_trained"`
	
	// Configuration of the index
	IndexConfig IndexConfig `json:"index_config"`
}