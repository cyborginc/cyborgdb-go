# EncryptedIndex

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexName** | Pointer to **string** |  | [optional] 
**IndexType** | Pointer to **string** |  | [optional] 
**Config** | Pointer to [**IndexConfig**](IndexConfig.md) |  | [optional] 

## Methods

### NewEncryptedIndex

`func NewEncryptedIndex() *EncryptedIndex`

NewEncryptedIndex instantiates a new EncryptedIndex object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEncryptedIndexWithDefaults

`func NewEncryptedIndexWithDefaults() *EncryptedIndex`

NewEncryptedIndexWithDefaults instantiates a new EncryptedIndex object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexName

`func (o *EncryptedIndex) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *EncryptedIndex) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *EncryptedIndex) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.

### HasIndexName

`func (o *EncryptedIndex) HasIndexName() bool`

HasIndexName returns a boolean if a field has been set.

### GetIndexType

`func (o *EncryptedIndex) GetIndexType() string`

GetIndexType returns the IndexType field if non-nil, zero value otherwise.

### GetIndexTypeOk

`func (o *EncryptedIndex) GetIndexTypeOk() (*string, bool)`

GetIndexTypeOk returns a tuple with the IndexType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexType

`func (o *EncryptedIndex) SetIndexType(v string)`

SetIndexType sets IndexType field to given value.

### HasIndexType

`func (o *EncryptedIndex) HasIndexType() bool`

HasIndexType returns a boolean if a field has been set.

### GetConfig

`func (o *EncryptedIndex) GetConfig() IndexConfig`

GetConfig returns the Config field if non-nil, zero value otherwise.

### GetConfigOk

`func (o *EncryptedIndex) GetConfigOk() (*IndexConfig, bool)`

GetConfigOk returns a tuple with the Config field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfig

`func (o *EncryptedIndex) SetConfig(v IndexConfig)`

SetConfig sets Config field to given value.

### HasConfig

`func (o *EncryptedIndex) HasConfig() bool`

HasConfig returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


