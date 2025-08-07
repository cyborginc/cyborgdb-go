# QueryRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexName** | **string** |  | 
**IndexKey** | **string** |  | 
**QueryVector** | Pointer to **[]float32** |  | [optional] 
**QueryVectors** | Pointer to **[][]float32** |  | [optional] 
**QueryContents** | Pointer to **string** |  | [optional] 
**TopK** | **int32** |  | 
**NProbes** | **int32** |  | 
**Greedy** | Pointer to **bool** |  | [optional] 
**Filters** | Pointer to **map[string]interface{}** |  | [optional] 
**Include** | **[]string** |  | 

## Methods

### NewQueryRequest

`func NewQueryRequest(indexName string, indexKey string, topK int32, nProbes int32, include []string, ) *QueryRequest`

NewQueryRequest instantiates a new QueryRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewQueryRequestWithDefaults

`func NewQueryRequestWithDefaults() *QueryRequest`

NewQueryRequestWithDefaults instantiates a new QueryRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

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


### GetQueryVector

`func (o *QueryRequest) GetQueryVector() []float32`

GetQueryVector returns the QueryVector field if non-nil, zero value otherwise.

### GetQueryVectorOk

`func (o *QueryRequest) GetQueryVectorOk() (*[]float32, bool)`

GetQueryVectorOk returns a tuple with the QueryVector field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueryVector

`func (o *QueryRequest) SetQueryVector(v []float32)`

SetQueryVector sets QueryVector field to given value.

### HasQueryVector

`func (o *QueryRequest) HasQueryVector() bool`

HasQueryVector returns a boolean if a field has been set.

### GetQueryVectors

`func (o *QueryRequest) GetQueryVectors() [][]float32`

GetQueryVectors returns the QueryVectors field if non-nil, zero value otherwise.

### GetQueryVectorsOk

`func (o *QueryRequest) GetQueryVectorsOk() (*[][]float32, bool)`

GetQueryVectorsOk returns a tuple with the QueryVectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueryVectors

`func (o *QueryRequest) SetQueryVectors(v [][]float32)`

SetQueryVectors sets QueryVectors field to given value.

### HasQueryVectors

`func (o *QueryRequest) HasQueryVectors() bool`

HasQueryVectors returns a boolean if a field has been set.

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



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


