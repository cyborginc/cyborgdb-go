# IndexOperationRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 

## Methods

### NewIndexOperationRequest

`func NewIndexOperationRequest(indexKey string, indexName string, ) *IndexOperationRequest`

NewIndexOperationRequest instantiates a new IndexOperationRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexOperationRequestWithDefaults

`func NewIndexOperationRequestWithDefaults() *IndexOperationRequest`

NewIndexOperationRequestWithDefaults instantiates a new IndexOperationRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexKey

`func (o *IndexOperationRequest) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *IndexOperationRequest) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *IndexOperationRequest) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *IndexOperationRequest) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *IndexOperationRequest) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *IndexOperationRequest) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


