# TrainRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IndexKey** | **string** | 32-byte encryption key as hex string | 
**IndexName** | **string** | ID name | 
**NLists** | Pointer to **NullableInt32** |  | [optional] 
**BatchSize** | Pointer to **NullableInt32** |  | [optional] 
**MaxIters** | Pointer to **NullableInt32** |  | [optional] 
**Tolerance** | Pointer to **NullableFloat32** |  | [optional] 
**MaxMemory** | Pointer to **NullableInt32** |  | [optional] 

## Methods

### NewTrainRequest

`func NewTrainRequest(indexKey string, indexName string, ) *TrainRequest`

NewTrainRequest instantiates a new TrainRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTrainRequestWithDefaults

`func NewTrainRequestWithDefaults() *TrainRequest`

NewTrainRequestWithDefaults instantiates a new TrainRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetIndexKey

`func (o *TrainRequest) GetIndexKey() string`

GetIndexKey returns the IndexKey field if non-nil, zero value otherwise.

### GetIndexKeyOk

`func (o *TrainRequest) GetIndexKeyOk() (*string, bool)`

GetIndexKeyOk returns a tuple with the IndexKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexKey

`func (o *TrainRequest) SetIndexKey(v string)`

SetIndexKey sets IndexKey field to given value.


### GetIndexName

`func (o *TrainRequest) GetIndexName() string`

GetIndexName returns the IndexName field if non-nil, zero value otherwise.

### GetIndexNameOk

`func (o *TrainRequest) GetIndexNameOk() (*string, bool)`

GetIndexNameOk returns a tuple with the IndexName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndexName

`func (o *TrainRequest) SetIndexName(v string)`

SetIndexName sets IndexName field to given value.


### GetNLists

`func (o *TrainRequest) GetNLists() int32`

GetNLists returns the NLists field if non-nil, zero value otherwise.

### GetNListsOk

`func (o *TrainRequest) GetNListsOk() (*int32, bool)`

GetNListsOk returns a tuple with the NLists field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNLists

`func (o *TrainRequest) SetNLists(v int32)`

SetNLists sets NLists field to given value.

### HasNLists

`func (o *TrainRequest) HasNLists() bool`

HasNLists returns a boolean if a field has been set.

### SetNListsNil

`func (o *TrainRequest) SetNListsNil(b bool)`

 SetNListsNil sets the value for NLists to be an explicit nil

### UnsetNLists
`func (o *TrainRequest) UnsetNLists()`

UnsetNLists ensures that no value is present for NLists, not even an explicit nil
### GetBatchSize

`func (o *TrainRequest) GetBatchSize() int32`

GetBatchSize returns the BatchSize field if non-nil, zero value otherwise.

### GetBatchSizeOk

`func (o *TrainRequest) GetBatchSizeOk() (*int32, bool)`

GetBatchSizeOk returns a tuple with the BatchSize field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBatchSize

`func (o *TrainRequest) SetBatchSize(v int32)`

SetBatchSize sets BatchSize field to given value.

### HasBatchSize

`func (o *TrainRequest) HasBatchSize() bool`

HasBatchSize returns a boolean if a field has been set.

### SetBatchSizeNil

`func (o *TrainRequest) SetBatchSizeNil(b bool)`

 SetBatchSizeNil sets the value for BatchSize to be an explicit nil

### UnsetBatchSize
`func (o *TrainRequest) UnsetBatchSize()`

UnsetBatchSize ensures that no value is present for BatchSize, not even an explicit nil
### GetMaxIters

`func (o *TrainRequest) GetMaxIters() int32`

GetMaxIters returns the MaxIters field if non-nil, zero value otherwise.

### GetMaxItersOk

`func (o *TrainRequest) GetMaxItersOk() (*int32, bool)`

GetMaxItersOk returns a tuple with the MaxIters field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxIters

`func (o *TrainRequest) SetMaxIters(v int32)`

SetMaxIters sets MaxIters field to given value.

### HasMaxIters

`func (o *TrainRequest) HasMaxIters() bool`

HasMaxIters returns a boolean if a field has been set.

### SetMaxItersNil

`func (o *TrainRequest) SetMaxItersNil(b bool)`

 SetMaxItersNil sets the value for MaxIters to be an explicit nil

### UnsetMaxIters
`func (o *TrainRequest) UnsetMaxIters()`

UnsetMaxIters ensures that no value is present for MaxIters, not even an explicit nil
### GetTolerance

`func (o *TrainRequest) GetTolerance() float32`

GetTolerance returns the Tolerance field if non-nil, zero value otherwise.

### GetToleranceOk

`func (o *TrainRequest) GetToleranceOk() (*float32, bool)`

GetToleranceOk returns a tuple with the Tolerance field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTolerance

`func (o *TrainRequest) SetTolerance(v float32)`

SetTolerance sets Tolerance field to given value.

### HasTolerance

`func (o *TrainRequest) HasTolerance() bool`

HasTolerance returns a boolean if a field has been set.

### SetToleranceNil

`func (o *TrainRequest) SetToleranceNil(b bool)`

 SetToleranceNil sets the value for Tolerance to be an explicit nil

### UnsetTolerance
`func (o *TrainRequest) UnsetTolerance()`

UnsetTolerance ensures that no value is present for Tolerance, not even an explicit nil
### GetMaxMemory

`func (o *TrainRequest) GetMaxMemory() int32`

GetMaxMemory returns the MaxMemory field if non-nil, zero value otherwise.

### GetMaxMemoryOk

`func (o *TrainRequest) GetMaxMemoryOk() (*int32, bool)`

GetMaxMemoryOk returns a tuple with the MaxMemory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxMemory

`func (o *TrainRequest) SetMaxMemory(v int32)`

SetMaxMemory sets MaxMemory field to given value.

### HasMaxMemory

`func (o *TrainRequest) HasMaxMemory() bool`

HasMaxMemory returns a boolean if a field has been set.

### SetMaxMemoryNil

`func (o *TrainRequest) SetMaxMemoryNil(b bool)`

 SetMaxMemoryNil sets the value for MaxMemory to be an explicit nil

### UnsetMaxMemory
`func (o *TrainRequest) UnsetMaxMemory()`

UnsetMaxMemory ensures that no value is present for MaxMemory, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


