package internal

type GetRequest struct {
	IndexName string   `json:"index_name"`
	IndexKey  string   `json:"index_key"`
	Ids       []string `json:"ids"`
	Include   []string `json:"include,omitempty"`
}

