# UpsertRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 
**Items** | [**[]VectorItem**](VectorItem.md) |  | 

## Methods

### NewUpsertRequest

`func NewUpsertRequest(indexKey string, indexName string, items []VectorItem, ) *UpsertRequest`

NewUpsertRequest instantiates a new UpsertRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpsertRequestWithDefaults

`func NewUpsertRequestWithDefaults() *UpsertRequest`

NewUpsertRequestWithDefaults instantiates a new UpsertRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexKey

`func (o *UpsertRequest) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *UpsertRequest) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *UpsertRequest) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *UpsertRequest) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *UpsertRequest) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *UpsertRequest) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetItems

`func (o *UpsertRequest) GetItems() []VectorItem`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *UpsertRequest) GetItemsOk() (*[]VectorItem, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *UpsertRequest) SetItems(v []VectorItem)`

SetItems sets Items field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


