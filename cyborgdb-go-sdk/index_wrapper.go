package cyborgdb

type IndexWrapper struct {
	client    *Client
	indexName string
	indexKey  []byte
	config    IndexConfig
}