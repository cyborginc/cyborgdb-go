# IndexIVFSQModel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dimension** | Pointer to **NullableInt32** |  | [optional]
**Type** | Pointer to **string** |  | [optional] [default to "ivfsq"]
**SqBits** | Pointer to **int32** |  | [optional] [default to 16]

## Methods

### NewIndexIVFSQModel

`func NewIndexIVFSQModel() *IndexIVFSQModel`

NewIndexIVFSQModel instantiates a new IndexIVFSQModel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexIVFSQModelWithDefaults

`func NewIndexIVFSQModelWithDefaults() *IndexIVFSQModel`

NewIndexIVFSQModelWithDefaults instantiates a new IndexIVFSQModel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDimension

`func (o *IndexIVFSQModel) GetDimension() int32`

GetDimension returns the Dimension field if non-nil, zero value otherwise.

### GetDimensionOk

`func (o *IndexIVFSQModel) GetDimensionOk() (*int32, bool)`

GetDimensionOk returns a tuple with the Dimension field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDimension

`func (o *IndexIVFSQModel) SetDimension(v int32)`

SetDimension sets Dimension field to given value.

### HasDimension

`func (o *IndexIVFSQModel) HasDimension() bool`

HasDimension returns a boolean if a field has been set.

### SetDimensionNil

`func (o *IndexIVFSQModel) SetDimensionNil(b bool)`

 SetDimensionNil sets the value for Dimension to be an explicit nil

### UnsetDimension
`func (o *IndexIVFSQModel) UnsetDimension()`

UnsetDimension ensures that no value is present for Dimension, not even an explicit nil
### GetType

`func (o *IndexIVFSQModel) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *IndexIVFSQModel) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *IndexIVFSQModel) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *IndexIVFSQModel) HasType() bool`

HasType returns a boolean if a field has been set.

### GetSqBits

`func (o *IndexIVFSQModel) GetSqBits() int32`

GetSqBits returns the SqBits field if non-nil, zero value otherwise.

### GetSqBitsOk

`func (o *IndexIVFSQModel) GetSqBitsOk() (*int32, bool)`

GetSqBitsOk returns a tuple with the SqBits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSqBits

`func (o *IndexIVFSQModel) SetSqBits(v int32)`

SetSqBits sets SqBits field to given value.

### HasSqBits

`func (o *IndexIVFSQModel) HasSqBits() bool`

HasSqBits returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


