# IndexIVFModel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dimension** | Pointer to **NullableInt32** |  | [optional] 
**Type** | Pointer to **string** |  | [optional] [default to "ivf"]

## Methods

### NewIndexIVFModel

`func NewIndexIVFModel() *IndexIVFModel`

NewIndexIVFModel instantiates a new IndexIVFModel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexIVFModelWithDefaults

`func NewIndexIVFModelWithDefaults() *IndexIVFModel`

NewIndexIVFModelWithDefaults instantiates a new IndexIVFModel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDimension

`func (o *IndexIVFModel) GetDimension() int32`

GetDimension returns the Dimension field if non-nil, zero value otherwise.

### GetDimensionOk

`func (o *IndexIVFModel) GetDimensionOk() (*int32, bool)`

GetDimensionOk returns a tuple with the Dimension field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDimension

`func (o *IndexIVFModel) SetDimension(v int32)`

SetDimension sets Dimension field to given value.

### HasDimension

`func (o *IndexIVFModel) HasDimension() bool`

HasDimension returns a boolean if a field has been set.

### SetDimensionNil

`func (o *IndexIVFModel) SetDimensionNil(b bool)`

 SetDimensionNil sets the value for Dimension to be an explicit nil

### UnsetDimension
`func (o *IndexIVFModel) UnsetDimension()`

UnsetDimension ensures that no value is present for Dimension, not even an explicit nil
### GetType

`func (o *IndexIVFModel) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *IndexIVFModel) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *IndexIVFModel) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *IndexIVFModel) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


