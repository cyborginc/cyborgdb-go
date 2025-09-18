# DeleteRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 
**Ids** | **[]string** |  | 

## Methods

### NewDeleteRequest

`func NewDeleteRequest(indexKey string, indexName string, ids []string, ) *DeleteRequest`

NewDeleteRequest instantiates a new DeleteRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteRequestWithDefaults

`func NewDeleteRequestWithDefaults() *DeleteRequest`

NewDeleteRequestWithDefaults instantiates a new DeleteRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexKey

`func (o *DeleteRequest) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *DeleteRequest) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *DeleteRequest) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *DeleteRequest) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *DeleteRequest) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *DeleteRequest) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetIds

`func (o *DeleteRequest) GetIds() []string`

GetIds returns the Ids field if non-nil, zero value otherwise.

### GetIdsOk

`func (o *DeleteRequest) GetIdsOk() (*[]string, bool)`

GetIdsOk returns a tuple with the Ids field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIds

`func (o *DeleteRequest) SetIds(v []string)`

SetIds sets Ids field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


