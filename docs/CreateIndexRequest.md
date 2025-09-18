# CreateIndexRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexConfig** | Pointer to [**NullableIndexConfig**](IndexConfig.md) |  | [optional] 
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 
**EmbeddingModel** | Pointer to **NullableString** |  | [optional] 
**Metric** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewCreateIndexRequest

`func NewCreateIndexRequest(indexKey string, indexName string, ) *CreateIndexRequest`

NewCreateIndexRequest instantiates a new CreateIndexRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateIndexRequestWithDefaults

`func NewCreateIndexRequestWithDefaults() *CreateIndexRequest`

NewCreateIndexRequestWithDefaults instantiates a new CreateIndexRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexConfig

`func (o *CreateIndexRequest) GetIndexConfig() IndexConfig`

GetIndexConfig returns the IndexConfig field if non-nil, zero value otherwise.

### GetIndexConfigOk

`func (o *CreateIndexRequest) GetIndexConfigOk() (*IndexConfig, bool)`

GetIndexConfigOk returns a tuple with the IndexConfig field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexConfig

`func (o *CreateIndexRequest) SetIndexConfig(v IndexConfig)`

SetIndexConfig sets IndexConfig field to given value.

### HasIndexConfig

`func (o *CreateIndexRequest) HasIndexConfig() bool`

HasIndexConfig returns a boolean if a field has been set.

### SetIndexConfigNil

`func (o *CreateIndexRequest) SetIndexConfigNil(b bool)`

 SetIndexConfigNil sets the value for IndexConfig to be an explicit nil

### UnsetIndexConfig
`func (o *CreateIndexRequest) UnsetIndexConfig()`

UnsetIndexConfig ensures that no value is present for IndexConfig, not even an explicit nil
### GetIndexKey

`func (o *CreateIndexRequest) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *CreateIndexRequest) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *CreateIndexRequest) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *CreateIndexRequest) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *CreateIndexRequest) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *CreateIndexRequest) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetEmbeddingModel

`func (o *CreateIndexRequest) GetEmbeddingModel() string`

GetEmbeddingModel returns the EmbeddingModel field if non-nil, zero value otherwise.

### GetEmbeddingModelOk

`func (o *CreateIndexRequest) GetEmbeddingModelOk() (*string, bool)`

GetEmbeddingModelOk returns a tuple with the EmbeddingModel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmbeddingModel

`func (o *CreateIndexRequest) SetEmbeddingModel(v string)`

SetEmbeddingModel sets EmbeddingModel field to given value.

### HasEmbeddingModel

`func (o *CreateIndexRequest) HasEmbeddingModel() bool`

HasEmbeddingModel returns a boolean if a field has been set.

### SetEmbeddingModelNil

`func (o *CreateIndexRequest) SetEmbeddingModelNil(b bool)`

 SetEmbeddingModelNil sets the value for EmbeddingModel to be an explicit nil

### UnsetEmbeddingModel
`func (o *CreateIndexRequest) UnsetEmbeddingModel()`

UnsetEmbeddingModel ensures that no value is present for EmbeddingModel, not even an explicit nil
### GetMetric

`func (o *CreateIndexRequest) GetMetric() string`

GetMetric returns the Metric field if non-nil, zero value otherwise.

### GetMetricOk

`func (o *CreateIndexRequest) GetMetricOk() (*string, bool)`

GetMetricOk returns a tuple with the Metric field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetric

`func (o *CreateIndexRequest) SetMetric(v string)`

SetMetric sets Metric field to given value.

### HasMetric

`func (o *CreateIndexRequest) HasMetric() bool`

HasMetric returns a boolean if a field has been set.

### SetMetricNil

`func (o *CreateIndexRequest) SetMetricNil(b bool)`

 SetMetricNil sets the value for Metric to be an explicit nil

### UnsetMetric
`func (o *CreateIndexRequest) UnsetMetric()`

UnsetMetric ensures that no value is present for Metric, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


