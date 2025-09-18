# QueryRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 
**QueryVectors** | Pointer to **[]float32** |  | [optional] 
**QueryContents** | Pointer to **NullableString** |  | [optional] 
**TopK** | Pointer to **NullableInt32** |  | [optional] 
**NProbes** | Pointer to **NullableInt32** |  | [optional] 
**Greedy** | Pointer to **NullableBool** |  | [optional] 
**Filters** | Pointer to **map[string]interface{}** |  | [optional] 
**Include** | Pointer to **[]string** |  | [optional] [default to [distance, metadata]]

## Methods

### NewQueryRequest

`func NewQueryRequest(indexKey string, indexName string, ) *QueryRequest`

NewQueryRequest instantiates a new QueryRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewQueryRequestWithDefaults

`func NewQueryRequestWithDefaults() *QueryRequest`

NewQueryRequestWithDefaults instantiates a new QueryRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexKey

`func (o *QueryRequest) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *QueryRequest) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *QueryRequest) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *QueryRequest) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *QueryRequest) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *QueryRequest) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetQueryVectors

`func (o *QueryRequest) GetQueryVectors() []float32`

GetQueryVectors returns the QueryVectors field if non-nil, zero value otherwise.

### GetQueryVectorsOk

`func (o *QueryRequest) GetQueryVectorsOk() (*[]float32, bool)`

GetQueryVectorsOk returns a tuple with the QueryVectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueryVectors

`func (o *QueryRequest) SetQueryVectors(v []float32)`

SetQueryVectors sets QueryVectors field to given value.

### HasQueryVectors

`func (o *QueryRequest) HasQueryVectors() bool`

HasQueryVectors returns a boolean if a field has been set.

### SetQueryVectorsNil

`func (o *QueryRequest) SetQueryVectorsNil(b bool)`

 SetQueryVectorsNil sets the value for QueryVectors to be an explicit nil

### UnsetQueryVectors
`func (o *QueryRequest) UnsetQueryVectors()`

UnsetQueryVectors ensures that no value is present for QueryVectors, not even an explicit nil
### GetQueryContents

`func (o *QueryRequest) GetQueryContents() string`

GetQueryContents returns the QueryContents field if non-nil, zero value otherwise.

### GetQueryContentsOk

`func (o *QueryRequest) GetQueryContentsOk() (*string, bool)`

GetQueryContentsOk returns a tuple with the QueryContents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueryContents

`func (o *QueryRequest) SetQueryContents(v string)`

SetQueryContents sets QueryContents field to given value.

### HasQueryContents

`func (o *QueryRequest) HasQueryContents() bool`

HasQueryContents returns a boolean if a field has been set.

### SetQueryContentsNil

`func (o *QueryRequest) SetQueryContentsNil(b bool)`

 SetQueryContentsNil sets the value for QueryContents to be an explicit nil

### UnsetQueryContents
`func (o *QueryRequest) UnsetQueryContents()`

UnsetQueryContents ensures that no value is present for QueryContents, not even an explicit nil
### GetTopK

`func (o *QueryRequest) GetTopK() int32`

GetTopK returns the TopK field if non-nil, zero value otherwise.

### GetTopKOk

`func (o *QueryRequest) GetTopKOk() (*int32, bool)`

GetTopKOk returns a tuple with the TopK field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTopK

`func (o *QueryRequest) SetTopK(v int32)`

SetTopK sets TopK field to given value.

### HasTopK

`func (o *QueryRequest) HasTopK() bool`

HasTopK returns a boolean if a field has been set.

### SetTopKNil

`func (o *QueryRequest) SetTopKNil(b bool)`

 SetTopKNil sets the value for TopK to be an explicit nil

### UnsetTopK
`func (o *QueryRequest) UnsetTopK()`

UnsetTopK ensures that no value is present for TopK, not even an explicit nil
### GetNProbes

`func (o *QueryRequest) GetNProbes() int32`

GetNProbes returns the NProbes field if non-nil, zero value otherwise.

### GetNProbesOk

`func (o *QueryRequest) GetNProbesOk() (*int32, bool)`

GetNProbesOk returns a tuple with the NProbes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNProbes

`func (o *QueryRequest) SetNProbes(v int32)`

SetNProbes sets NProbes field to given value.

### HasNProbes

`func (o *QueryRequest) HasNProbes() bool`

HasNProbes returns a boolean if a field has been set.

### SetNProbesNil

`func (o *QueryRequest) SetNProbesNil(b bool)`

 SetNProbesNil sets the value for NProbes to be an explicit nil

### UnsetNProbes
`func (o *QueryRequest) UnsetNProbes()`

UnsetNProbes ensures that no value is present for NProbes, not even an explicit nil
### GetGreedy

`func (o *QueryRequest) GetGreedy() bool`

GetGreedy returns the Greedy field if non-nil, zero value otherwise.

### GetGreedyOk

`func (o *QueryRequest) GetGreedyOk() (*bool, bool)`

GetGreedyOk returns a tuple with the Greedy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGreedy

`func (o *QueryRequest) SetGreedy(v bool)`

SetGreedy sets Greedy field to given value.

### HasGreedy

`func (o *QueryRequest) HasGreedy() bool`

HasGreedy returns a boolean if a field has been set.

### SetGreedyNil

`func (o *QueryRequest) SetGreedyNil(b bool)`

 SetGreedyNil sets the value for Greedy to be an explicit nil

### UnsetGreedy
`func (o *QueryRequest) UnsetGreedy()`

UnsetGreedy ensures that no value is present for Greedy, not even an explicit nil
### GetFilters

`func (o *QueryRequest) GetFilters() map[string]interface{}`

GetFilters returns the Filters field if non-nil, zero value otherwise.

### GetFiltersOk

`func (o *QueryRequest) GetFiltersOk() (*map[string]interface{}, bool)`

GetFiltersOk returns a tuple with the Filters field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFilters

`func (o *QueryRequest) SetFilters(v map[string]interface{})`

SetFilters sets Filters field to given value.

### HasFilters

`func (o *QueryRequest) HasFilters() bool`

HasFilters returns a boolean if a field has been set.

### SetFiltersNil

`func (o *QueryRequest) SetFiltersNil(b bool)`

 SetFiltersNil sets the value for Filters to be an explicit nil

### UnsetFilters
`func (o *QueryRequest) UnsetFilters()`

UnsetFilters ensures that no value is present for Filters, not even an explicit nil
### GetInclude

`func (o *QueryRequest) GetInclude() []string`

GetInclude returns the Include field if non-nil, zero value otherwise.

### GetIncludeOk

`func (o *QueryRequest) GetIncludeOk() (*[]string, bool)`

GetIncludeOk returns a tuple with the Include field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInclude

`func (o *QueryRequest) SetInclude(v []string)`

SetInclude sets Include field to given value.

### HasInclude

`func (o *QueryRequest) HasInclude() bool`

HasInclude returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


