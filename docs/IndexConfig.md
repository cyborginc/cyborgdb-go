# IndexConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dimension** | Pointer to **int32** |  | [optional] 
**Type** | Pointer to **string** |  | [optional] [default to "ivfflat"]
**PqDim** | **int32** |  | 
**PqBits** | **int32** |  | 

## Methods

### NewIndexConfig

`func NewIndexConfig(pqDim int32, pqBits int32, ) *IndexConfig`

NewIndexConfig instantiates a new IndexConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexConfigWithDefaults

`func NewIndexConfigWithDefaults() *IndexConfig`

NewIndexConfigWithDefaults instantiates a new IndexConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDimension

`func (o *IndexConfig) GetDimension() int32`

GetDimension returns the Dimension field if non-nil, zero value otherwise.

### GetDimensionOk

`func (o *IndexConfig) GetDimensionOk() (*int32, bool)`

GetDimensionOk returns a tuple with the Dimension field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDimension

`func (o *IndexConfig) SetDimension(v int32)`

SetDimension sets Dimension field to given value.

### HasDimension

`func (o *IndexConfig) HasDimension() bool`

HasDimension returns a boolean if a field has been set.

### GetType

`func (o *IndexConfig) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *IndexConfig) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *IndexConfig) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *IndexConfig) HasType() bool`

HasType returns a boolean if a field has been set.

### GetPqDim

`func (o *IndexConfig) GetPqDim() int32`

GetPqDim returns the PqDim field if non-nil, zero value otherwise.

### GetPqDimOk

`func (o *IndexConfig) GetPqDimOk() (*int32, bool)`

GetPqDimOk returns a tuple with the PqDim field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPqDim

`func (o *IndexConfig) SetPqDim(v int32)`

SetPqDim sets PqDim field to given value.


### GetPqBits

`func (o *IndexConfig) GetPqBits() int32`

GetPqBits returns the PqBits field if non-nil, zero value otherwise.

### GetPqBitsOk

`func (o *IndexConfig) GetPqBitsOk() (*int32, bool)`

GetPqBitsOk returns a tuple with the PqBits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPqBits

`func (o *IndexConfig) SetPqBits(v int32)`

SetPqBits sets PqBits field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


