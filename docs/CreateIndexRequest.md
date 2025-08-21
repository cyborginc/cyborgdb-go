# CreateIndexRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexName** | **string** |  | 
**IndexKey** | **string** |  | 
**Config** | [**IndexConfig**](IndexConfig.md) |  | 

## Methods

### NewCreateIndexRequest

`func NewCreateIndexRequest(indexName string, indexKey string, config IndexConfig, ) *CreateIndexRequest`

NewCreateIndexRequest instantiates a new CreateIndexRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateIndexRequestWithDefaults

`func NewCreateIndexRequestWithDefaults() *CreateIndexRequest`

NewCreateIndexRequestWithDefaults instantiates a new CreateIndexRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

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


### GetConfig

`func (o *CreateIndexRequest) GetConfig() IndexConfig`

GetConfig returns the Config field if non-nil, zero value otherwise.

### GetConfigOk

`func (o *CreateIndexRequest) GetConfigOk() (*IndexConfig, bool)`

GetConfigOk returns a tuple with the Config field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfig

`func (o *CreateIndexRequest) SetConfig(v IndexConfig)`

SetConfig sets Config field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


