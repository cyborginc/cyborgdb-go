# VectorItem

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Vector** | Pointer to **[]float32** |  | [optional] 
**Contents** | Pointer to [**NullableContents**](Contents.md) |  | [optional] 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewVectorItem

`func NewVectorItem(id string, ) *VectorItem`

NewVectorItem instantiates a new VectorItem object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewVectorItemWithDefaults

`func NewVectorItemWithDefaults() *VectorItem`

NewVectorItemWithDefaults instantiates a new VectorItem object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *VectorItem) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *VectorItem) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *VectorItem) SetId(v string)`

SetId sets Id field to given value.


### GetVector

`func (o *VectorItem) GetVector() []float32`

GetVector returns the Vector field if non-nil, zero value otherwise.

### GetVectorOk

`func (o *VectorItem) GetVectorOk() (*[]float32, bool)`

GetVectorOk returns a tuple with the Vector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVector

`func (o *VectorItem) SetVector(v []float32)`

SetVector sets Vector field to given value.

### HasVector

`func (o *VectorItem) HasVector() bool`

HasVector returns a boolean if a field has been set.

### SetVectorNil

`func (o *VectorItem) SetVectorNil(b bool)`

 SetVectorNil sets the value for Vector to be an explicit nil

### UnsetVector
`func (o *VectorItem) UnsetVector()`

UnsetVector ensures that no value is present for Vector, not even an explicit nil
### GetContents

`func (o *VectorItem) GetContents() Contents`

GetContents returns the Contents field if non-nil, zero value otherwise.

### GetContentsOk

`func (o *VectorItem) GetContentsOk() (*Contents, bool)`

GetContentsOk returns a tuple with the Contents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContents

`func (o *VectorItem) SetContents(v Contents)`

SetContents sets Contents field to given value.

### HasContents

`func (o *VectorItem) HasContents() bool`

HasContents returns a boolean if a field has been set.

### SetContentsNil

`func (o *VectorItem) SetContentsNil(b bool)`

 SetContentsNil sets the value for Contents to be an explicit nil

### UnsetContents
`func (o *VectorItem) UnsetContents()`

UnsetContents ensures that no value is present for Contents, not even an explicit nil
### GetMetadata

`func (o *VectorItem) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *VectorItem) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *VectorItem) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *VectorItem) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### SetMetadataNil

`func (o *VectorItem) SetMetadataNil(b bool)`

 SetMetadataNil sets the value for Metadata to be an explicit nil

### UnsetMetadata
`func (o *VectorItem) UnsetMetadata()`

UnsetMetadata ensures that no value is present for Metadata, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


