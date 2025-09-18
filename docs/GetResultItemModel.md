# GetResultItemModel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 
**Contents** | Pointer to **Nullable*os.File** |  | [optional] 
**Vector** | Pointer to **[]float32** |  | [optional] 

## Methods

### NewGetResultItemModel

`func NewGetResultItemModel(id string, ) *GetResultItemModel`

NewGetResultItemModel instantiates a new GetResultItemModel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGetResultItemModelWithDefaults

`func NewGetResultItemModelWithDefaults() *GetResultItemModel`

NewGetResultItemModelWithDefaults instantiates a new GetResultItemModel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *GetResultItemModel) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *GetResultItemModel) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *GetResultItemModel) SetId(v string)`

SetId sets Id field to given value.


### GetMetadata

`func (o *GetResultItemModel) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *GetResultItemModel) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *GetResultItemModel) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *GetResultItemModel) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### SetMetadataNil

`func (o *GetResultItemModel) SetMetadataNil(b bool)`

 SetMetadataNil sets the value for Metadata to be an explicit nil

### UnsetMetadata
`func (o *GetResultItemModel) UnsetMetadata()`

UnsetMetadata ensures that no value is present for Metadata, not even an explicit nil
### GetContents

`func (o *GetResultItemModel) GetContents() *os.File`

GetContents returns the Contents field if non-nil, zero value otherwise.

### GetContentsOk

`func (o *GetResultItemModel) GetContentsOk() (**os.File, bool)`

GetContentsOk returns a tuple with the Contents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContents

`func (o *GetResultItemModel) SetContents(v *os.File)`

SetContents sets Contents field to given value.

### HasContents

`func (o *GetResultItemModel) HasContents() bool`

HasContents returns a boolean if a field has been set.

### SetContentsNil

`func (o *GetResultItemModel) SetContentsNil(b bool)`

 SetContentsNil sets the value for Contents to be an explicit nil

### UnsetContents
`func (o *GetResultItemModel) UnsetContents()`

UnsetContents ensures that no value is present for Contents, not even an explicit nil
### GetVector

`func (o *GetResultItemModel) GetVector() []float32`

GetVector returns the Vector field if non-nil, zero value otherwise.

### GetVectorOk

`func (o *GetResultItemModel) GetVectorOk() (*[]float32, bool)`

GetVectorOk returns a tuple with the Vector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVector

`func (o *GetResultItemModel) SetVector(v []float32)`

SetVector sets Vector field to given value.

### HasVector

`func (o *GetResultItemModel) HasVector() bool`

HasVector returns a boolean if a field has been set.

### SetVectorNil

`func (o *GetResultItemModel) SetVectorNil(b bool)`

 SetVectorNil sets the value for Vector to be an explicit nil

### UnsetVector
`func (o *GetResultItemModel) UnsetVector()`

UnsetVector ensures that no value is present for Vector, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


