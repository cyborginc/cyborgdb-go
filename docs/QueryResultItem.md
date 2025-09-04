# QueryResultItem

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Distance** | Pointer to **NullableFloat32** |  | [optional] 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 
**Vector** | Pointer to **[]float32** |  | [optional] 

## Methods

### NewQueryResultItem

`func NewQueryResultItem(id string, ) *QueryResultItem`

NewQueryResultItem instantiates a new QueryResultItem object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewQueryResultItemWithDefaults

`func NewQueryResultItemWithDefaults() *QueryResultItem`

NewQueryResultItemWithDefaults instantiates a new QueryResultItem object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *QueryResultItem) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *QueryResultItem) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *QueryResultItem) SetId(v string)`

SetId sets Id field to given value.


### GetDistance

`func (o *QueryResultItem) GetDistance() float32`

GetDistance returns the Distance field if non-nil, zero value otherwise.

### GetDistanceOk

`func (o *QueryResultItem) GetDistanceOk() (*float32, bool)`

GetDistanceOk returns a tuple with the Distance field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDistance

`func (o *QueryResultItem) SetDistance(v float32)`

SetDistance sets Distance field to given value.

### HasDistance

`func (o *QueryResultItem) HasDistance() bool`

HasDistance returns a boolean if a field has been set.

### SetDistanceNil

`func (o *QueryResultItem) SetDistanceNil(b bool)`

 SetDistanceNil sets the value for Distance to be an explicit nil

### UnsetDistance
`func (o *QueryResultItem) UnsetDistance()`

UnsetDistance ensures that no value is present for Distance, not even an explicit nil
### GetMetadata

`func (o *QueryResultItem) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *QueryResultItem) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *QueryResultItem) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *QueryResultItem) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### SetMetadataNil

`func (o *QueryResultItem) SetMetadataNil(b bool)`

 SetMetadataNil sets the value for Metadata to be an explicit nil

### UnsetMetadata
`func (o *QueryResultItem) UnsetMetadata()`

UnsetMetadata ensures that no value is present for Metadata, not even an explicit nil
### GetVector

`func (o *QueryResultItem) GetVector() []float32`

GetVector returns the Vector field if non-nil, zero value otherwise.

### GetVectorOk

`func (o *QueryResultItem) GetVectorOk() (*[]float32, bool)`

GetVectorOk returns a tuple with the Vector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVector

`func (o *QueryResultItem) SetVector(v []float32)`

SetVector sets Vector field to given value.

### HasVector

`func (o *QueryResultItem) HasVector() bool`

HasVector returns a boolean if a field has been set.

### SetVectorNil

`func (o *QueryResultItem) SetVectorNil(b bool)`

 SetVectorNil sets the value for Vector to be an explicit nil

### UnsetVector
`func (o *QueryResultItem) UnsetVector()`

UnsetVector ensures that no value is present for Vector, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


