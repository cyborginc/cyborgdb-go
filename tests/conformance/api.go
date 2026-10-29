package main

import (
	"context"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// api is the SDK as the adapter uses it, behind interfaces so the forwarding test can record the
// calls (ADAPTERS.md: "Go swaps the binding behind an interface").
type api interface {
	NewClient(baseURL, apiKey string) (client, error)
}

type client interface {
	ListIndexes(ctx context.Context) ([]string, error)
	CreateIndex(ctx context.Context, params *cyborgdb.CreateIndexParams) (index, error)
	LoadIndex(ctx context.Context, indexName string, indexKey []byte) (index, error)
	GetHealth(ctx context.Context) (map[string]string, error)
	GetTrainingStatus(ctx context.Context) (*cyborgdb.TrainingStatus, error)
	ListEmbeddingModels(ctx context.Context) ([]cyborgdb.EmbeddingModel, error)
}

// index is exactly *cyborgdb.EncryptedIndex's method set the adapter calls.
type index interface {
	Describe(ctx context.Context) (*cyborgdb.IndexDescription, error)
	GetNumVectors(ctx context.Context) (int32, error)
	Upsert(ctx context.Context, input cyborgdb.UpsertInput) error
	Query(ctx context.Context, input cyborgdb.QueryInput) (*cyborgdb.QueryResponse, error)
	Get(ctx context.Context, ids []string, include []string) (*cyborgdb.GetResponse, error)
	Delete(ctx context.Context, ids []string) error
	ListIDs(ctx context.Context) (*cyborgdb.ListIDsResponse, error)
	Train(ctx context.Context, params cyborgdb.TrainParams) error
	QueryMetadata(ctx context.Context, params cyborgdb.QueryMetadataParams) (*cyborgdb.QueryMetadataResponse, error)
	DeleteIndex(ctx context.Context) error
	CreateUser(ctx context.Context, permissions []string) (*cyborgdb.CreatedUser, error)
	ListUsers(ctx context.Context) ([]cyborgdb.UserInfo, error)
	DeleteUser(ctx context.Context, userID string) error
}

var _ index = (*cyborgdb.EncryptedIndex)(nil)

// sdk is the real api.
type sdk struct{}

func (sdk) NewClient(baseURL, apiKey string) (client, error) {
	c, err := cyborgdb.NewClient(baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	return realClient{c}, nil
}

type realClient struct{ *cyborgdb.Client }

func (c realClient) CreateIndex(ctx context.Context, params *cyborgdb.CreateIndexParams) (index, error) {
	ix, err := c.Client.CreateIndex(ctx, params)
	if err != nil {
		return nil, err
	}
	return ix, nil
}

func (c realClient) LoadIndex(ctx context.Context, indexName string, indexKey []byte) (index, error) {
	ix, err := c.Client.LoadIndex(ctx, indexName, indexKey)
	if err != nil {
		return nil, err
	}
	return ix, nil
}
