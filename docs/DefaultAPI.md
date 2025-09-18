# \DefaultAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateIndexV1IndexesCreatePost**](DefaultAPI.md#CreateIndexV1IndexesCreatePost) | **Post** /v1/indexes/create | Create Encrypted Index
[**DeleteIndexV1IndexesDeletePost**](DefaultAPI.md#DeleteIndexV1IndexesDeletePost) | **Post** /v1/indexes/delete | Delete Encrypted Index
[**DeleteVectorsV1VectorsDeletePost**](DefaultAPI.md#DeleteVectorsV1VectorsDeletePost) | **Post** /v1/vectors/delete | Delete Items from Encrypted Index
[**GetIndexInfoV1IndexesDescribePost**](DefaultAPI.md#GetIndexInfoV1IndexesDescribePost) | **Post** /v1/indexes/describe | Describe Encrypted Index
[**GetIndexSizeV1VectorsNumVectorsPost**](DefaultAPI.md#GetIndexSizeV1VectorsNumVectorsPost) | **Post** /v1/vectors/num_vectors | Get the number of vectors in an index
[**GetTrainingStatusV1IndexesTrainingStatusGet**](DefaultAPI.md#GetTrainingStatusV1IndexesTrainingStatusGet) | **Get** /v1/indexes/training-status | Get Training Status
[**GetVectorsV1VectorsGetPost**](DefaultAPI.md#GetVectorsV1VectorsGetPost) | **Post** /v1/vectors/get | Get Items from Encrypted Index
[**HealthCheckV1HealthGet**](DefaultAPI.md#HealthCheckV1HealthGet) | **Get** /v1/health | Health check endpoint
[**ListIdsV1VectorsListIdsPost**](DefaultAPI.md#ListIdsV1VectorsListIdsPost) | **Post** /v1/vectors/list_ids | List all IDs in an index
[**ListIndexesV1IndexesListGet**](DefaultAPI.md#ListIndexesV1IndexesListGet) | **Get** /v1/indexes/list | List Encrypted Indexes
[**QueryVectorsV1VectorsQueryPost**](DefaultAPI.md#QueryVectorsV1VectorsQueryPost) | **Post** /v1/vectors/query | Query Encrypted Index
[**TrainIndexV1IndexesTrainPost**](DefaultAPI.md#TrainIndexV1IndexesTrainPost) | **Post** /v1/indexes/train | Train Encrypted index
[**UpsertVectorsV1VectorsUpsertPost**](DefaultAPI.md#UpsertVectorsV1VectorsUpsertPost) | **Post** /v1/vectors/upsert | Add Items to Encrypted Index



## CreateIndexV1IndexesCreatePost

> CyborgdbServiceApiSchemasIndexSuccessResponseModel CreateIndexV1IndexesCreatePost(ctx).CreateIndexRequest(createIndexRequest).Execute()

Create Encrypted Index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	createIndexRequest := *openapiclient.NewCreateIndexRequest("IndexKey_example", "IndexName_example") // CreateIndexRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.CreateIndexV1IndexesCreatePost(context.Background()).CreateIndexRequest(createIndexRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.CreateIndexV1IndexesCreatePost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateIndexV1IndexesCreatePost`: CyborgdbServiceApiSchemasIndexSuccessResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.CreateIndexV1IndexesCreatePost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateIndexV1IndexesCreatePostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createIndexRequest** | [**CreateIndexRequest**](CreateIndexRequest.md) |  | 

### Return type

[**CyborgdbServiceApiSchemasIndexSuccessResponseModel**](CyborgdbServiceApiSchemasIndexSuccessResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteIndexV1IndexesDeletePost

> CyborgdbServiceApiSchemasIndexSuccessResponseModel DeleteIndexV1IndexesDeletePost(ctx).IndexOperationRequest(indexOperationRequest).Execute()

Delete Encrypted Index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	indexOperationRequest := *openapiclient.NewIndexOperationRequest("IndexKey_example", "IndexName_example") // IndexOperationRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.DeleteIndexV1IndexesDeletePost(context.Background()).IndexOperationRequest(indexOperationRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.DeleteIndexV1IndexesDeletePost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteIndexV1IndexesDeletePost`: CyborgdbServiceApiSchemasIndexSuccessResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.DeleteIndexV1IndexesDeletePost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiDeleteIndexV1IndexesDeletePostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **indexOperationRequest** | [**IndexOperationRequest**](IndexOperationRequest.md) |  | 

### Return type

[**CyborgdbServiceApiSchemasIndexSuccessResponseModel**](CyborgdbServiceApiSchemasIndexSuccessResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteVectorsV1VectorsDeletePost

> CyborgdbServiceApiSchemasVectorsSuccessResponseModel DeleteVectorsV1VectorsDeletePost(ctx).DeleteRequest(deleteRequest).Execute()

Delete Items from Encrypted Index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	deleteRequest := *openapiclient.NewDeleteRequest("IndexKey_example", "IndexName_example", []string{"Ids_example"}) // DeleteRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.DeleteVectorsV1VectorsDeletePost(context.Background()).DeleteRequest(deleteRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.DeleteVectorsV1VectorsDeletePost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteVectorsV1VectorsDeletePost`: CyborgdbServiceApiSchemasVectorsSuccessResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.DeleteVectorsV1VectorsDeletePost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiDeleteVectorsV1VectorsDeletePostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **deleteRequest** | [**DeleteRequest**](DeleteRequest.md) |  | 

### Return type

[**CyborgdbServiceApiSchemasVectorsSuccessResponseModel**](CyborgdbServiceApiSchemasVectorsSuccessResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetIndexInfoV1IndexesDescribePost

> IndexInfoResponseModel GetIndexInfoV1IndexesDescribePost(ctx).IndexOperationRequest(indexOperationRequest).Execute()

Describe Encrypted Index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	indexOperationRequest := *openapiclient.NewIndexOperationRequest("IndexKey_example", "IndexName_example") // IndexOperationRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetIndexInfoV1IndexesDescribePost(context.Background()).IndexOperationRequest(indexOperationRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetIndexInfoV1IndexesDescribePost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetIndexInfoV1IndexesDescribePost`: IndexInfoResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetIndexInfoV1IndexesDescribePost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGetIndexInfoV1IndexesDescribePostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **indexOperationRequest** | [**IndexOperationRequest**](IndexOperationRequest.md) |  | 

### Return type

[**IndexInfoResponseModel**](IndexInfoResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetIndexSizeV1VectorsNumVectorsPost

> CyborgdbServiceApiSchemasVectorsSuccessResponseModel GetIndexSizeV1VectorsNumVectorsPost(ctx).IndexOperationRequest(indexOperationRequest).Execute()

Get the number of vectors in an index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	indexOperationRequest := *openapiclient.NewIndexOperationRequest("IndexKey_example", "IndexName_example") // IndexOperationRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetIndexSizeV1VectorsNumVectorsPost(context.Background()).IndexOperationRequest(indexOperationRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetIndexSizeV1VectorsNumVectorsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetIndexSizeV1VectorsNumVectorsPost`: CyborgdbServiceApiSchemasVectorsSuccessResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetIndexSizeV1VectorsNumVectorsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGetIndexSizeV1VectorsNumVectorsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **indexOperationRequest** | [**IndexOperationRequest**](IndexOperationRequest.md) |  | 

### Return type

[**CyborgdbServiceApiSchemasVectorsSuccessResponseModel**](CyborgdbServiceApiSchemasVectorsSuccessResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetTrainingStatusV1IndexesTrainingStatusGet

> interface{} GetTrainingStatusV1IndexesTrainingStatusGet(ctx).Execute()

Get Training Status



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetTrainingStatusV1IndexesTrainingStatusGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetTrainingStatusV1IndexesTrainingStatusGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetTrainingStatusV1IndexesTrainingStatusGet`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetTrainingStatusV1IndexesTrainingStatusGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiGetTrainingStatusV1IndexesTrainingStatusGetRequest struct via the builder pattern


### Return type

**interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetVectorsV1VectorsGetPost

> GetResponseModel GetVectorsV1VectorsGetPost(ctx).GetRequest(getRequest).Execute()

Get Items from Encrypted Index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	getRequest := *openapiclient.NewGetRequest("IndexKey_example", "IndexName_example", []string{"Ids_example"}) // GetRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetVectorsV1VectorsGetPost(context.Background()).GetRequest(getRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetVectorsV1VectorsGetPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetVectorsV1VectorsGetPost`: GetResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetVectorsV1VectorsGetPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGetVectorsV1VectorsGetPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **getRequest** | [**GetRequest**](GetRequest.md) |  | 

### Return type

[**GetResponseModel**](GetResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HealthCheckV1HealthGet

> map[string]string HealthCheckV1HealthGet(ctx).Execute()

Health check endpoint



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.HealthCheckV1HealthGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.HealthCheckV1HealthGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `HealthCheckV1HealthGet`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.HealthCheckV1HealthGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiHealthCheckV1HealthGetRequest struct via the builder pattern


### Return type

**map[string]string**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListIdsV1VectorsListIdsPost

> ListIDsResponse ListIdsV1VectorsListIdsPost(ctx).ListIDsRequest(listIDsRequest).Execute()

List all IDs in an index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	listIDsRequest := *openapiclient.NewListIDsRequest("IndexKey_example", "IndexName_example") // ListIDsRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ListIdsV1VectorsListIdsPost(context.Background()).ListIDsRequest(listIDsRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ListIdsV1VectorsListIdsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListIdsV1VectorsListIdsPost`: ListIDsResponse
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ListIdsV1VectorsListIdsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListIdsV1VectorsListIdsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **listIDsRequest** | [**ListIDsRequest**](ListIDsRequest.md) |  | 

### Return type

[**ListIDsResponse**](ListIDsResponse.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListIndexesV1IndexesListGet

> IndexListResponseModel ListIndexesV1IndexesListGet(ctx).Execute()

List Encrypted Indexes



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ListIndexesV1IndexesListGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ListIndexesV1IndexesListGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListIndexesV1IndexesListGet`: IndexListResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ListIndexesV1IndexesListGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListIndexesV1IndexesListGetRequest struct via the builder pattern


### Return type

[**IndexListResponseModel**](IndexListResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## QueryVectorsV1VectorsQueryPost

> QueryResponse QueryVectorsV1VectorsQueryPost(ctx).Request(request).Execute()

Query Encrypted Index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	request := *openapiclient.NewRequest("IndexKey_example", "IndexName_example", [][]float32{[]float32{float32(123)}}) // Request | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.QueryVectorsV1VectorsQueryPost(context.Background()).Request(request).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.QueryVectorsV1VectorsQueryPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `QueryVectorsV1VectorsQueryPost`: QueryResponse
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.QueryVectorsV1VectorsQueryPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiQueryVectorsV1VectorsQueryPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**Request**](Request.md) |  | 

### Return type

[**QueryResponse**](QueryResponse.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## TrainIndexV1IndexesTrainPost

> CyborgdbServiceApiSchemasIndexSuccessResponseModel TrainIndexV1IndexesTrainPost(ctx).TrainRequest(trainRequest).Execute()

Train Encrypted index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	trainRequest := *openapiclient.NewTrainRequest("IndexKey_example", "IndexName_example") // TrainRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.TrainIndexV1IndexesTrainPost(context.Background()).TrainRequest(trainRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.TrainIndexV1IndexesTrainPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `TrainIndexV1IndexesTrainPost`: CyborgdbServiceApiSchemasIndexSuccessResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.TrainIndexV1IndexesTrainPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiTrainIndexV1IndexesTrainPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **trainRequest** | [**TrainRequest**](TrainRequest.md) |  | 

### Return type

[**CyborgdbServiceApiSchemasIndexSuccessResponseModel**](CyborgdbServiceApiSchemasIndexSuccessResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpsertVectorsV1VectorsUpsertPost

> CyborgdbServiceApiSchemasVectorsSuccessResponseModel UpsertVectorsV1VectorsUpsertPost(ctx).UpsertRequest(upsertRequest).Execute()

Add Items to Encrypted Index



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	upsertRequest := *openapiclient.NewUpsertRequest("IndexKey_example", "IndexName_example", []openapiclient.VectorItem{*openapiclient.NewVectorItem("Id_example")}) // UpsertRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.UpsertVectorsV1VectorsUpsertPost(context.Background()).UpsertRequest(upsertRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.UpsertVectorsV1VectorsUpsertPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpsertVectorsV1VectorsUpsertPost`: CyborgdbServiceApiSchemasVectorsSuccessResponseModel
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.UpsertVectorsV1VectorsUpsertPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiUpsertVectorsV1VectorsUpsertPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **upsertRequest** | [**UpsertRequest**](UpsertRequest.md) |  | 

### Return type

[**CyborgdbServiceApiSchemasVectorsSuccessResponseModel**](CyborgdbServiceApiSchemasVectorsSuccessResponseModel.md)

### Authorization

[APIKeyHeader](../README.md#APIKeyHeader)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

