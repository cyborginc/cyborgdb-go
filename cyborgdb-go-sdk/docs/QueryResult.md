# QueryResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Distance** | Pointer to **float32** |  | [optional] 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 
**Vector** | Pointer to **[]float32** |  | [optional] 
**Contents** | Pointer to **string** |  | [optional] 

## Methods

### NewQueryResult

`func NewQueryResult(id string, ) *QueryResult`

NewQueryResult instantiates a new QueryResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewQueryResultWithDefaults

`func NewQueryResultWithDefaults() *QueryResult`

NewQueryResultWithDefaults instantiates a new QueryResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *QueryResult) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *QueryResult) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *QueryResult) SetId(v string)`

SetId sets Id field to given value.


### GetDistance

`func (o *QueryResult) GetDistance() float32`

GetDistance returns the Distance field if non-nil, zero value otherwise.

### GetDistanceOk

`func (o *QueryResult) GetDistanceOk() (*float32, bool)`

GetDistanceOk returns a tuple with the Distance field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDistance

`func (o *QueryResult) SetDistance(v float32)`

SetDistance sets Distance field to given value.

### HasDistance

`func (o *QueryResult) HasDistance() bool`

HasDistance returns a boolean if a field has been set.

### GetMetadata

`func (o *QueryResult) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *QueryResult) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *QueryResult) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *QueryResult) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetVector

`func (o *QueryResult) GetVector() []float32`

GetVector returns the Vector field if non-nil, zero value otherwise.

### GetVectorOk

`func (o *QueryResult) GetVectorOk() (*[]float32, bool)`

GetVectorOk returns a tuple with the Vector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVector

`func (o *QueryResult) SetVector(v []float32)`

SetVector sets Vector field to given value.

### HasVector

`func (o *QueryResult) HasVector() bool`

HasVector returns a boolean if a field has been set.

### GetContents

`func (o *QueryResult) GetContents() string`

GetContents returns the Contents field if non-nil, zero value otherwise.

### GetContentsOk

`func (o *QueryResult) GetContentsOk() (*string, bool)`

GetContentsOk returns a tuple with the Contents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContents

`func (o *QueryResult) SetContents(v string)`

SetContents sets Contents field to given value.

### HasContents

`func (o *QueryResult) HasContents() bool`

HasContents returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


