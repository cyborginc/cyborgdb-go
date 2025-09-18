# ErrorResponseModel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**StatusCode** | **int32** |  | 
**Detail** | **string** |  | 

## Methods

### NewErrorResponseModel

`func NewErrorResponseModel(statusCode int32, detail string, ) *ErrorResponseModel`

NewErrorResponseModel instantiates a new ErrorResponseModel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewErrorResponseModelWithDefaults

`func NewErrorResponseModelWithDefaults() *ErrorResponseModel`

NewErrorResponseModelWithDefaults instantiates a new ErrorResponseModel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetStatusCode

`func (o *ErrorResponseModel) GetStatusCode() int32`

GetStatusCode returns the StatusCode field if non-nil, zero value otherwise.

### GetStatusCodeOk

`func (o *ErrorResponseModel) GetStatusCodeOk() (*int32, bool)`

GetStatusCodeOk returns a tuple with the StatusCode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatusCode

`func (o *ErrorResponseModel) SetStatusCode(v int32)`

SetStatusCode sets StatusCode field to given value.


### GetDetail

`func (o *ErrorResponseModel) GetDetail() string`

GetDetail returns the Detail field if non-nil, zero value otherwise.

### GetDetailOk

`func (o *ErrorResponseModel) GetDetailOk() (*string, bool)`

GetDetailOk returns a tuple with the Detail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetail

`func (o *ErrorResponseModel) SetDetail(v string)`

SetDetail sets Detail field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


