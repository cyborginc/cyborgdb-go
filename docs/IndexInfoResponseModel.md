# IndexInfoResponseModel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexName** | **string** |  | 
**IndexType** | **string** |  | 
**IsTrained** | **bool** |  | 
**IndexConfig** | **map[string]interface{}** |  | 

## Methods

### NewIndexInfoResponseModel

`func NewIndexInfoResponseModel(indexName string, indexType string, isTrained bool, indexConfig map[string]interface{}, ) *IndexInfoResponseModel`

NewIndexInfoResponseModel instantiates a new IndexInfoResponseModel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexInfoResponseModelWithDefaults

`func NewIndexInfoResponseModelWithDefaults() *IndexInfoResponseModel`

NewIndexInfoResponseModelWithDefaults instantiates a new IndexInfoResponseModel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexName

`func (o *IndexInfoResponseModel) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *IndexInfoResponseModel) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *IndexInfoResponseModel) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetIndexType

`func (o *IndexInfoResponseModel) GetIndexType() string`

GetIndexType returns the IndexType field if non-nil, zero value otherwise.

### GetIndexTypeOk

`func (o *IndexInfoResponseModel) GetIndexTypeOk() (*string, bool)`

GetIndexTypeOk returns a tuple with the IndexType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexType

`func (o *IndexInfoResponseModel) SetIndexType(v string)`

SetIndexType sets IndexType field to given value.


### GetIsTrained

`func (o *IndexInfoResponseModel) GetIsTrained() bool`

GetIsTrained returns the IsTrained field if non-nil, zero value otherwise.

### GetIsTrainedOk

`func (o *IndexInfoResponseModel) GetIsTrainedOk() (*bool, bool)`

GetIsTrainedOk returns a tuple with the IsTrained field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsTrained

`func (o *IndexInfoResponseModel) SetIsTrained(v bool)`

SetIsTrained sets IsTrained field to given value.


### GetIndexConfig

`func (o *IndexInfoResponseModel) GetIndexConfig() map[string]interface{}`

GetIndexConfig returns the IndexConfig field if non-nil, zero value otherwise.

### GetIndexConfigOk

`func (o *IndexInfoResponseModel) GetIndexConfigOk() (*map[string]interface{}, bool)`

GetIndexConfigOk returns a tuple with the IndexConfig field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexConfig

`func (o *IndexInfoResponseModel) SetIndexConfig(v map[string]interface{})`

SetIndexConfig sets IndexConfig field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


