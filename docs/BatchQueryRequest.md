# BatchQueryRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 
**QueryVectors** | **[][]float32** |  | 
**TopK** | Pointer to **NullableInt32** |  | [optional] 
**NProbes** | Pointer to **NullableInt32** |  | [optional] 
**Greedy** | Pointer to **NullableBool** |  | [optional] 
**Filters** | Pointer to **map[string]interface{}** |  | [optional] 
**Include** | Pointer to **[]string** |  | [optional] [default to [distance, metadata]]

## Methods

### NewBatchQueryRequest

`func NewBatchQueryRequest(indexKey string, indexName string, queryVectors [][]float32, ) *BatchQueryRequest`

NewBatchQueryRequest instantiates a new BatchQueryRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewBatchQueryRequestWithDefaults

`func NewBatchQueryRequestWithDefaults() *BatchQueryRequest`

NewBatchQueryRequestWithDefaults instantiates a new BatchQueryRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexKey

`func (o *BatchQueryRequest) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *BatchQueryRequest) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *BatchQueryRequest) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *BatchQueryRequest) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *BatchQueryRequest) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *BatchQueryRequest) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetQueryVectors

`func (o *BatchQueryRequest) GetQueryVectors() [][]float32`

GetQueryVectors returns the QueryVectors field if non-nil, zero value otherwise.

### GetQueryVectorsOk

`func (o *BatchQueryRequest) GetQueryVectorsOk() (*[][]float32, bool)`

GetQueryVectorsOk returns a tuple with the QueryVectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueryVectors

`func (o *BatchQueryRequest) SetQueryVectors(v [][]float32)`

SetQueryVectors sets QueryVectors field to given value.


### GetTopK

`func (o *BatchQueryRequest) GetTopK() int32`

GetTopK returns the TopK field if non-nil, zero value otherwise.

### GetTopKOk

`func (o *BatchQueryRequest) GetTopKOk() (*int32, bool)`

GetTopKOk returns a tuple with the TopK field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTopK

`func (o *BatchQueryRequest) SetTopK(v int32)`

SetTopK sets TopK field to given value.

### HasTopK

`func (o *BatchQueryRequest) HasTopK() bool`

HasTopK returns a boolean if a field has been set.

### SetTopKNil

`func (o *BatchQueryRequest) SetTopKNil(b bool)`

 SetTopKNil sets the value for TopK to be an explicit nil

### UnsetTopK
`func (o *BatchQueryRequest) UnsetTopK()`

UnsetTopK ensures that no value is present for TopK, not even an explicit nil
### GetNProbes

`func (o *BatchQueryRequest) GetNProbes() int32`

GetNProbes returns the NProbes field if non-nil, zero value otherwise.

### GetNProbesOk

`func (o *BatchQueryRequest) GetNProbesOk() (*int32, bool)`

GetNProbesOk returns a tuple with the NProbes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNProbes

`func (o *BatchQueryRequest) SetNProbes(v int32)`

SetNProbes sets NProbes field to given value.

### HasNProbes

`func (o *BatchQueryRequest) HasNProbes() bool`

HasNProbes returns a boolean if a field has been set.

### SetNProbesNil

`func (o *BatchQueryRequest) SetNProbesNil(b bool)`

 SetNProbesNil sets the value for NProbes to be an explicit nil

### UnsetNProbes
`func (o *BatchQueryRequest) UnsetNProbes()`

UnsetNProbes ensures that no value is present for NProbes, not even an explicit nil
### GetGreedy

`func (o *BatchQueryRequest) GetGreedy() bool`

GetGreedy returns the Greedy field if non-nil, zero value otherwise.

### GetGreedyOk

`func (o *BatchQueryRequest) GetGreedyOk() (*bool, bool)`

GetGreedyOk returns a tuple with the Greedy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGreedy

`func (o *BatchQueryRequest) SetGreedy(v bool)`

SetGreedy sets Greedy field to given value.

### HasGreedy

`func (o *BatchQueryRequest) HasGreedy() bool`

HasGreedy returns a boolean if a field has been set.

### SetGreedyNil

`func (o *BatchQueryRequest) SetGreedyNil(b bool)`

 SetGreedyNil sets the value for Greedy to be an explicit nil

### UnsetGreedy
`func (o *BatchQueryRequest) UnsetGreedy()`

UnsetGreedy ensures that no value is present for Greedy, not even an explicit nil
### GetFilters

`func (o *BatchQueryRequest) GetFilters() map[string]interface{}`

GetFilters returns the Filters field if non-nil, zero value otherwise.

### GetFiltersOk

`func (o *BatchQueryRequest) GetFiltersOk() (*map[string]interface{}, bool)`

GetFiltersOk returns a tuple with the Filters field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFilters

`func (o *BatchQueryRequest) SetFilters(v map[string]interface{})`

SetFilters sets Filters field to given value.

### HasFilters

`func (o *BatchQueryRequest) HasFilters() bool`

HasFilters returns a boolean if a field has been set.

### SetFiltersNil

`func (o *BatchQueryRequest) SetFiltersNil(b bool)`

 SetFiltersNil sets the value for Filters to be an explicit nil

### UnsetFilters
`func (o *BatchQueryRequest) UnsetFilters()`

UnsetFilters ensures that no value is present for Filters, not even an explicit nil
### GetInclude

`func (o *BatchQueryRequest) GetInclude() []string`

GetInclude returns the Include field if non-nil, zero value otherwise.

### GetIncludeOk

`func (o *BatchQueryRequest) GetIncludeOk() (*[]string, bool)`

GetIncludeOk returns a tuple with the Include field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInclude

`func (o *BatchQueryRequest) SetInclude(v []string)`

SetInclude sets Include field to given value.

### HasInclude

`func (o *BatchQueryRequest) HasInclude() bool`

HasInclude returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


