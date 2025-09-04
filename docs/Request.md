# Request

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 
**QueryVectors** | **[][]float32** |  | 
**QueryContents** | Pointer to **string** |  | [optional] 
**TopK** | Pointer to **int32** |  | [optional] 
**NProbes** | Pointer to **int32** |  | [optional] 
**Greedy** | Pointer to **bool** |  | [optional] 
**Filters** | Pointer to **map[string]interface{}** |  | [optional] 
**Include** | Pointer to **[]string** |  | [optional] [default to [distance, metadata]]

## Methods

### NewRequest

`func NewRequest(indexKey string, indexName string, queryVectors [][]float32, ) *Request`

NewRequest instantiates a new Request object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRequestWithDefaults

`func NewRequestWithDefaults() *Request`

NewRequestWithDefaults instantiates a new Request object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexKey

`func (o *Request) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *Request) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *Request) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *Request) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *Request) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *Request) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetQueryVectors

`func (o *Request) GetQueryVectors() [][]float32`

GetQueryVectors returns the QueryVectors field if non-nil, zero value otherwise.

### GetQueryVectorsOk

`func (o *Request) GetQueryVectorsOk() (*[][]float32, bool)`

GetQueryVectorsOk returns a tuple with the QueryVectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueryVectors

`func (o *Request) SetQueryVectors(v [][]float32)`

SetQueryVectors sets QueryVectors field to given value.


### GetQueryContents

`func (o *Request) GetQueryContents() string`

GetQueryContents returns the QueryContents field if non-nil, zero value otherwise.

### GetQueryContentsOk

`func (o *Request) GetQueryContentsOk() (*string, bool)`

GetQueryContentsOk returns a tuple with the QueryContents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueryContents

`func (o *Request) SetQueryContents(v string)`

SetQueryContents sets QueryContents field to given value.

### HasQueryContents

`func (o *Request) HasQueryContents() bool`

HasQueryContents returns a boolean if a field has been set.

### GetTopK

`func (o *Request) GetTopK() int32`

GetTopK returns the TopK field if non-nil, zero value otherwise.

### GetTopKOk

`func (o *Request) GetTopKOk() (*int32, bool)`

GetTopKOk returns a tuple with the TopK field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTopK

`func (o *Request) SetTopK(v int32)`

SetTopK sets TopK field to given value.

### HasTopK

`func (o *Request) HasTopK() bool`

HasTopK returns a boolean if a field has been set.

### GetNProbes

`func (o *Request) GetNProbes() int32`

GetNProbes returns the NProbes field if non-nil, zero value otherwise.

### GetNProbesOk

`func (o *Request) GetNProbesOk() (*int32, bool)`

GetNProbesOk returns a tuple with the NProbes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNProbes

`func (o *Request) SetNProbes(v int32)`

SetNProbes sets NProbes field to given value.

### HasNProbes

`func (o *Request) HasNProbes() bool`

HasNProbes returns a boolean if a field has been set.

### GetGreedy

`func (o *Request) GetGreedy() bool`

GetGreedy returns the Greedy field if non-nil, zero value otherwise.

### GetGreedyOk

`func (o *Request) GetGreedyOk() (*bool, bool)`

GetGreedyOk returns a tuple with the Greedy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGreedy

`func (o *Request) SetGreedy(v bool)`

SetGreedy sets Greedy field to given value.

### HasGreedy

`func (o *Request) HasGreedy() bool`

HasGreedy returns a boolean if a field has been set.

### GetFilters

`func (o *Request) GetFilters() map[string]interface{}`

GetFilters returns the Filters field if non-nil, zero value otherwise.

### GetFiltersOk

`func (o *Request) GetFiltersOk() (*map[string]interface{}, bool)`

GetFiltersOk returns a tuple with the Filters field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFilters

`func (o *Request) SetFilters(v map[string]interface{})`

SetFilters sets Filters field to given value.

### HasFilters

`func (o *Request) HasFilters() bool`

HasFilters returns a boolean if a field has been set.

### GetInclude

`func (o *Request) GetInclude() []string`

GetInclude returns the Include field if non-nil, zero value otherwise.

### GetIncludeOk

`func (o *Request) GetIncludeOk() (*[]string, bool)`

GetIncludeOk returns a tuple with the Include field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInclude

`func (o *Request) SetInclude(v []string)`

SetInclude sets Include field to given value.

### HasInclude

`func (o *Request) HasInclude() bool`

HasInclude returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


