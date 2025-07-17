# IndexConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dimension** | **int32** |  | 
**Metric** | **string** |  | 
**IndexType** | **string** |  | 
**NLists** | **int32** |  | 
**PqDim** | Pointer to **int32** |  | [optional] 
**PqBits** | Pointer to **int32** |  | [optional] 

## Methods

### NewIndexConfig

`func NewIndexConfig(dimension int32, metric string, indexType string, nLists int32, ) *IndexConfig`

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


### GetMetric

`func (o *IndexConfig) GetMetric() string`

GetMetric returns the Metric field if non-nil, zero value otherwise.

### GetMetricOk

`func (o *IndexConfig) GetMetricOk() (*string, bool)`

GetMetricOk returns a tuple with the Metric field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetric

`func (o *IndexConfig) SetMetric(v string)`

SetMetric sets Metric field to given value.


### GetIndexType

`func (o *IndexConfig) GetIndexType() string`

GetIndexType returns the IndexType field if non-nil, zero value otherwise.

### GetIndexTypeOk

`func (o *IndexConfig) GetIndexTypeOk() (*string, bool)`

GetIndexTypeOk returns a tuple with the IndexType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexType

`func (o *IndexConfig) SetIndexType(v string)`

SetIndexType sets IndexType field to given value.


### GetNLists

`func (o *IndexConfig) GetNLists() int32`

GetNLists returns the NLists field if non-nil, zero value otherwise.

### GetNListsOk

`func (o *IndexConfig) GetNListsOk() (*int32, bool)`

GetNListsOk returns a tuple with the NLists field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNLists

`func (o *IndexConfig) SetNLists(v int32)`

SetNLists sets NLists field to given value.


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

### HasPqDim

`func (o *IndexConfig) HasPqDim() bool`

HasPqDim returns a boolean if a field has been set.

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

### HasPqBits

`func (o *IndexConfig) HasPqBits() bool`

HasPqBits returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


