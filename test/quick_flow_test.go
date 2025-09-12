package test

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

func generateUniqueName(prefix string) string {
	if prefix == "" {
		prefix = "test_"
	}
	return fmt.Sprintf("%s%s", prefix, uuid.New().String())
}

func checkQueryResults(results *cyborgdb.QueryResponse, neighbors [][]int32, numQueries int) float64 {
	// Parse results to extract IDs from the returned dictionaries
	resultsData := results.GetResults()

	// Handle both single query and batch query results
	var queryResults [][]cyborgdb.QueryResultItem
	if resultsData.ArrayOfQueryResultItem != nil {
		// Single query result - wrap in array
		queryResults = [][]cyborgdb.QueryResultItem{*resultsData.ArrayOfQueryResultItem}
	} else if resultsData.ArrayOfArrayOfQueryResultItem != nil {
		// Batch query result
		queryResults = *resultsData.ArrayOfArrayOfQueryResultItem
	} else {
		panic("Unexpected results type")
	}

	resultIds := make([][]int, len(queryResults))
	for i, qr := range queryResults {
		resultIds[i] = make([]int, len(qr))
		for j, res := range qr {
			id, _ := strconv.Atoi(res.GetId())
			resultIds[i][j] = id
		}
	}

	if len(neighbors) != len(resultIds) || len(neighbors[0]) != len(resultIds[0]) {
		panic(fmt.Sprintf("The shapes of the neighbors and results do not match: [%d,%d] != [%d,%d]",
			len(neighbors), len(neighbors[0]), len(resultIds), len(resultIds[0])))
	}

	// Compute the recall using the neighbors
	recall := make([]float64, numQueries)
	for i := 0; i < numQueries; i++ {
		intersectionCount := 0
		for _, n := range neighbors[i] {
			for _, r := range resultIds[i] {
				if int(n) == r {
					intersectionCount++
					break
				}
			}
		}
		recall[i] = float64(intersectionCount) / float64(len(neighbors[i]))
	}

	// Return mean recall
	sum := 0.0
	for _, r := range recall {
		sum += r
	}
	return sum / float64(len(recall))
}

func safeInt(val interface{}) int {
	switch v := val.(type) {
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return -1
		}
		return i
	case float64:
		return int(v)
	case int:
		return v
	default:
		return -1
	}
}

func checkMetadataResults(results []*cyborgdb.QueryResponse, metadataNeighbors [][][]int32, metadataCandidates [][]int32, numQueries int) []float64 {
	allResults := make([][][]cyborgdb.QueryResultItem, len(results))

	for idx, result := range results {
		resultsData := result.GetResults()

		// Handle both single query and batch query results
		if resultsData.ArrayOfQueryResultItem != nil {
			// Single query result - wrap in array
			allResults[idx] = [][]cyborgdb.QueryResultItem{*resultsData.ArrayOfQueryResultItem}
		} else if resultsData.ArrayOfArrayOfQueryResultItem != nil {
			// Batch query result
			allResults[idx] = *resultsData.ArrayOfArrayOfQueryResultItem
		} else {
			panic("Unexpected results type")
		}
	}

	resultIds := make([][][]int, len(allResults))
	for idx, result := range allResults {
		resultIds[idx] = make([][]int, len(result))
		for i, queryResults := range result {
			resultIds[idx][i] = make([]int, len(queryResults))
			for j, res := range queryResults {
				resultIds[idx][i][j] = safeInt(res.GetId())
			}
		}
	}

	recalls := make([]float64, 0)

	for idx, result := range resultIds {
		// Get candidates for this query
		candidates := metadataCandidates[idx]

		// Get groundtruth neighbors for this metadata query
		metadataNeighborsIndices := metadataNeighbors[idx]

		recall := make([]float64, numQueries)
		numReturned := 0
		numExpected := 0

		// Iterate over the queries
		for i := 0; i < numQueries; i++ {
			// Get the groundtruth neighbors for this query
			groundtruthIndices := metadataNeighborsIndices[i]

			groundtruthIds := make([]int, 0)
			for _, idx := range groundtruthIndices {
				if idx != -1 && idx >= 0 && int(idx) < len(candidates) {
					groundtruthIds = append(groundtruthIds, int(candidates[idx]))
				}
			}

			// Get the returned neighbors for this query
			returned := result[i]

			// Update the number of returned neighbors
			numReturned += len(returned)
			localExpected := 0
			for _, id := range groundtruthIds {
				if id != -1 {
					localExpected++
				}
			}
			numExpected += localExpected

			// If we expect no results and got no results, recall is 100%
			if len(returned) == 0 && localExpected == 0 {
				recall[i] = 1
				continue
			}

			// Check if the number of returned neighbors is correct
			if len(returned) > 100 {
				panic(fmt.Sprintf("More than 100 results returned: got %d instead of 100", len(returned)))
			}

			// Compute the recall for this query
			intersectionCount := 0
			for _, gid := range groundtruthIds {
				for _, rid := range returned {
					if gid == rid {
						intersectionCount++
						break
					}
				}
			}
			minExpected := localExpected
			if minExpected > 100 {
				minExpected = 100
			}
			if minExpected > 0 {
				recall[i] = float64(intersectionCount) / float64(minExpected)
			}
		}

		// Calculate mean recall
		sum := 0.0
		for _, r := range recall {
			sum += r
		}
		recalls = append(recalls, sum/float64(len(recall)))
	}

	return recalls
}

type TestData struct {
	Vectors                    [][]float32   `json:"vectors"`
	Queries                    [][]float32   `json:"queries"`
	UntrainedNeighbors         [][]int32     `json:"untrained_neighbors"`
	TrainedNeighbors           [][]int32     `json:"trained_neighbors"`
	Metadata                   []interface{} `json:"metadata"`
	MetadataQueries            []interface{} `json:"metadata_queries"`
	MetadataQueryNames         []string      `json:"metadata_query_names"`
	UntrainedMetadataMatches   [][]int32     `json:"untrained_metadata_matches"`
	TrainedMetadataMatches     [][]int32     `json:"trained_metadata_matches"`
	UntrainedMetadataNeighbors [][][]int32   `json:"untrained_metadata_neighbors"`
	TrainedMetadataNeighbors   [][][]int32   `json:"trained_metadata_neighbors"`
	UntrainedRecall            float64       `json:"untrained_recall"`
	TrainedRecall              float64       `json:"trained_recall"`
	NumUntrainedVectors        int           `json:"num_untrained_vectors"`
	NumTrainedVectors          int           `json:"num_trained_vectors"`
}

func TestUnitFlow(t *testing.T) {
	// Load environment variables from .env.local
	godotenv.Load("../.env.local")

	// Create context for all operations
	ctx := context.Background()

	// Load test data
	testDir := filepath.Dir(".")
	jsonPath := filepath.Join(testDir, "unit_test_flow_data.json")
	jsonData, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	// Compute & validate checksum
	expectedChecksum := "a2989692cb12e8667b22bee4177acb295b72a23be82458ce7dd06e4a901cb04d"
	checksum := fmt.Sprintf("%x", sha256.Sum256(jsonData))
	if checksum != expectedChecksum {
		t.Fatalf("Checksum mismatch: expected %s, got %s", expectedChecksum, checksum)
	}

	var data TestData
	if err = json.Unmarshal(jsonData, &data); err != nil {
		t.Fatalf("Failed to parse test data: %v", err)
	}

	// Set up test variables
	vectors := data.Vectors
	queries := data.Queries
	untrainedNeighbors := data.UntrainedNeighbors
	trainedNeighbors := data.TrainedNeighbors
	metadata := data.Metadata
	metadataQueries := data.MetadataQueries
	untrainedMetadataMatches := data.UntrainedMetadataMatches
	trainedMetadataMatches := data.TrainedMetadataMatches
	untrainedMetadataNeighbors := data.UntrainedMetadataNeighbors
	trainedMetadataNeighbors := data.TrainedMetadataNeighbors
	untrainedRecall := data.UntrainedRecall
	trainedRecall := data.TrainedRecall
	numUntrainedVectors := data.NumUntrainedVectors
	numTrainedVectors := data.NumTrainedVectors
	totalNumVectors := numUntrainedVectors + numTrainedVectors
	numQueries := len(queries)
	dimension := len(vectors[0])
	nLists := 100

	// CYBORGDB SETUP: Create the index once
	indexConfig := cyborgdb.IndexIVFFlat(int32(dimension))

	client, err := cyborgdb.NewClient(
		"http://localhost:8000",
		os.Getenv("CYBORGDB_API_KEY"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	indexName := generateUniqueName("")
	indexKeyBytes := make([]byte, 32)
	cryptoRand.Read(indexKeyBytes)
	indexKey := hex.EncodeToString(indexKeyBytes)

	metric := "euclidean"
	createParams := &cyborgdb.CreateIndexParams{
		IndexName:   indexName,
		IndexKey:    indexKey,
		IndexConfig: indexConfig,
		Metric:      &metric,
	}

	index, err := client.CreateIndex(ctx, createParams)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Clean up at the end
	defer func() {
		if index != nil {
			index.DeleteIndex(ctx)
		}
	}()

	// Test 00: Get Health
	t.Run("test_00_get_health", func(t *testing.T) {
		health, err := client.GetHealth(ctx)
		if err != nil {
			t.Errorf("Failed to get health: %v", err)
		}
		if health["status"] != "healthy" {
			t.Errorf("API is not healthy: %v", health)
		}
	})

	// Test 01: Untrained Upsert
	t.Run("test_01_untrained_upsert", func(t *testing.T) {
		items := make([]cyborgdb.VectorItem, numUntrainedVectors)
		for i := 0; i < numUntrainedVectors; i++ {
			items[i] = cyborgdb.VectorItem{
				Id:       strconv.Itoa(i),
				Vector:   vectors[i],
				Metadata: metadata[i].(map[string]interface{}),
			}
		}
		err := index.Upsert(ctx, items)
		if err != nil {
			t.Errorf("Failed to upsert: %v", err)
		}

		// Wait for 1 second to ensure upsert is processed
		time.Sleep(1 * time.Second)

		// Check if the index has all IDs
		results, err := index.ListIDs(ctx)
		if err != nil {
			t.Errorf("Failed to list IDs: %v", err)
		}

		expectedIds := make([]string, numUntrainedVectors)
		for i := 0; i < numUntrainedVectors; i++ {
			expectedIds[i] = strconv.Itoa(i)
		}

		sort.Strings(results.Ids)
		sort.Strings(expectedIds)

		if len(results.Ids) != len(expectedIds) {
			t.Errorf("ID count mismatch: expected %d, got %d", len(expectedIds), len(results.Ids))
		}
		for i := range expectedIds {
			if results.Ids[i] != expectedIds[i] {
				t.Errorf("ID mismatch at index %d: expected %s, got %s", i, expectedIds[i], results.Ids[i])
			}
		}
	})

	// Test 02: Untrained Query No Metadata
	t.Run("test_02_untrained_query_no_metadata", func(t *testing.T) {
		nProbesVal := int32(1)
		queryParams := cyborgdb.QueryParams{
			BatchQueryVectors: queries,
			TopK:              100,
			NProbes:           &nProbesVal,
		}
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Errorf("Failed to query: %v", err)
		}

		recall := checkQueryResults(results, untrainedNeighbors, numQueries)
		fmt.Printf("Untrained Query (No Metadata). Expected recall: %f, got %f\n", untrainedRecall, recall)

		if math.Abs(recall-untrainedRecall) > 0.02 {
			t.Errorf("Recall mismatch: expected %f±0.02, got %f", untrainedRecall, recall)
		}

		// Check if index is still untrained
		trained := index.IsTrained()
		if trained {
			t.Errorf("Index should still be untrained")
		}
	})

	// Test 03: Untrained Query Metadata
	t.Run("test_03_untrained_query_metadata", func(t *testing.T) {
		results := make([]*cyborgdb.QueryResponse, 0)

		for _, metadataQuery := range metadataQueries {
			nProbesVal := int32(1)
			queryParams := cyborgdb.QueryParams{
				BatchQueryVectors: queries,
				TopK:              100,
				NProbes:           &nProbesVal,
				Filters:           metadataQuery.(map[string]interface{}),
			}
			queryResult, err := index.Query(ctx, queryParams)
			if err != nil {
				t.Errorf("Failed to query with metadata: %v", err)
			}
			results = append(results, queryResult)
		}

		recalls := checkMetadataResults(results, untrainedMetadataNeighbors, untrainedMetadataMatches, numQueries)

		for idx, recall := range recalls {
			fmt.Printf("\nMetadata Query #%d\n", idx+1)
			fmt.Printf("Metadata filters: %v\n", metadataQueries[idx])
			fmt.Printf("Number of candidates: %d / %d\n", len(untrainedMetadataNeighbors[idx]), numUntrainedVectors)
			fmt.Printf("Mean recall: %.2f%%\n", recall*100)

			if math.Abs(recall-untrainedRecall) > 0.02 {
				t.Errorf("Recall mismatch for query %d: expected %f±0.02, got %f", idx+1, untrainedRecall, recall)
			}
		}

		// Check if index is still untrained
		trained := index.IsTrained()
		if trained {
			t.Errorf("Index should still be untrained")
		}
	})

	// Test 04: Untrained Get
	t.Run("test_04_untrained_get", func(t *testing.T) {
		numGet := 1000
		getIndices := make([]int, 0, numGet)
		usedIndices := make(map[int]bool)

		for len(getIndices) < numGet {
			idx := rand.Intn(numUntrainedVectors)
			if !usedIndices[idx] {
				usedIndices[idx] = true
				getIndices = append(getIndices, idx)
			}
		}

		getIndicesStr := make([]string, numGet)
		for i, idx := range getIndices {
			getIndicesStr[i] = strconv.Itoa(idx)
		}

		include := []string{"vector", "contents", "metadata"}
		getResults, err := index.Get(ctx, getIndicesStr, include)
		if err != nil {
			t.Errorf("Failed to get vectors: %v", err)
		}

		for i, getResult := range getResults.Results {
			if getResult.GetId() != getIndicesStr[i] {
				t.Errorf("ID mismatch: %s != %s", getResult.GetId(), getIndicesStr[i])
			}

			// Check vector equality
			expectedVector := vectors[getIndices[i]]
			resultVector := getResult.GetVector()
			if len(resultVector) != len(expectedVector) {
				t.Errorf("Vector length mismatch for index %d", i)
			}
			for j := range expectedVector {
				if resultVector[j] != expectedVector[j] {
					t.Errorf("Vector mismatch for index %d, element %d", i, j)
					break
				}
			}

			// Check metadata equality
			metadataStr1, _ := json.Marshal(getResult.GetMetadata())
			metadataStr2, _ := json.Marshal(metadata[getIndices[i]])
			if string(metadataStr1) != string(metadataStr2) {
				t.Errorf("Metadata mismatch for index %d", i)
			}
		}

		// Check if index is still untrained
		trained := index.IsTrained()
		if trained {
			t.Errorf("Index should still be untrained")
		}
	})

	// Test 05: Untrained List IDs
	t.Run("test_05_untrained_list_ids", func(t *testing.T) {
		results, err := index.ListIDs(ctx)
		if err != nil {
			t.Errorf("Failed to list IDs: %v", err)
		}

		expectedIds := make([]string, numUntrainedVectors)
		for i := 0; i < numUntrainedVectors; i++ {
			expectedIds[i] = strconv.Itoa(i)
		}

		sort.Strings(results.Ids)
		sort.Strings(expectedIds)

		if len(results.Ids) != len(expectedIds) {
			t.Errorf("ID count mismatch: expected %d, got %d", len(expectedIds), len(results.Ids))
		}

		// Check if index is still untrained
		trained := index.IsTrained()
		if trained {
			t.Errorf("Index should still be untrained")
		}
	})

	// Test 06: Upsert for Train
	t.Run("test_06_upsert_for_train", func(t *testing.T) {
		items := make([]cyborgdb.VectorItem, numTrainedVectors)
		for i := 0; i < numTrainedVectors; i++ {
			idx := numUntrainedVectors + i
			items[i] = cyborgdb.VectorItem{
				Id:       strconv.Itoa(idx),
				Vector:   vectors[idx],
				Metadata: metadata[idx].(map[string]interface{}),
			}
		}
		err := index.Upsert(ctx, items)
		if err != nil {
			t.Errorf("Failed to upsert training vectors: %v", err)
		}

		// Wait for 1 second to ensure upsert is processed
		time.Sleep(1 * time.Second)

		// Check if the index has all IDs
		results, err := index.ListIDs(ctx)
		if err != nil {
			t.Errorf("Failed to list IDs: %v", err)
		}

		expectedIds := make([]string, totalNumVectors)
		for i := 0; i < totalNumVectors; i++ {
			expectedIds[i] = strconv.Itoa(i)
		}

		sort.Strings(results.Ids)
		sort.Strings(expectedIds)

		if len(results.Ids) != len(expectedIds) {
			t.Errorf("ID count mismatch: expected %d, got %d", len(expectedIds), len(results.Ids))
		}
	})

	// Test 07: Wait for Initial Training
	t.Run("test_07_wait_for_initial_training", func(t *testing.T) {
		numRetries := 60
		trained := false

		for attempt := 0; attempt < numRetries; attempt++ {
			time.Sleep(2 * time.Second)

			// Check training status with the server
			isTraining, err := index.CheckTrainingStatus(ctx)
			if err != nil {
				fmt.Printf("Error checking training status: %v, retrying... (%d/%d)\n", err, attempt+1, numRetries)
				continue
			}

			// If not training and index is marked as trained, we're done
			if !isTraining && index.IsTrained() {
				trained = true
				fmt.Println("Index is now trained.")
				break
			} else if isTraining {
				fmt.Printf("Index is being trained, waiting... (%d/%d)\n", attempt+1, numRetries)
			} else {
				fmt.Printf("Index not trained yet, retrying... (%d/%d)\n", attempt+1, numRetries)
			}
		}

		if !trained {
			t.Errorf("Index did not become trained in time")
		}
	})

	// Test 08: Trained Query Should Get Perfect Recall
	t.Run("test_08_trained_query_should_get_perfect_recall", func(t *testing.T) {
		nProbesVal := int32(nLists)
		queryParams := cyborgdb.QueryParams{
			BatchQueryVectors: queries,
			TopK:              100,
			NProbes:           &nProbesVal,
		}
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Errorf("Failed to query: %v", err)
		}

		recall := checkQueryResults(results, trainedNeighbors, numQueries)
		expectedRecall := 1.0
		fmt.Printf("Trained Query (N_PROBES == N_LISTS). Expected recall: %f, got %f\n", expectedRecall, recall)

		if recall != expectedRecall {
			t.Errorf("Recall should be perfect: expected %f, got %f", expectedRecall, recall)
		}
	})

	// Test 09: Trained Query No Metadata
	t.Run("test_09_trained_query_no_metadata", func(t *testing.T) {
		nProbesVal := int32(24)
		queryParams := cyborgdb.QueryParams{
			BatchQueryVectors: queries,
			TopK:              100,
			NProbes:           &nProbesVal,
		}
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Errorf("Failed to query: %v", err)
		}

		recall := checkQueryResults(results, trainedNeighbors, numQueries)
		fmt.Printf("Trained Query (No Metadata). Expected recall: %f, got %f\n", trainedRecall, recall)

		if math.Abs(recall-trainedRecall) > 0.08 {
			t.Errorf("Recall mismatch: expected %f±0.08, got %f", trainedRecall, recall)
		}
	})

	// Test 10: Trained Query No Metadata Auto N_Probes
	t.Run("test_10_trained_query_no_metadata_auto_n_probes", func(t *testing.T) {
		queryParams := cyborgdb.QueryParams{
			BatchQueryVectors: queries,
			TopK:              100,
			// NProbes not set - will use auto
		}
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Errorf("Failed to query: %v", err)
		}

		recall := checkQueryResults(results, trainedNeighbors, numQueries)
		fmt.Printf("Trained Query (No Metadata, Auto n_probes). Expected recall: %f, got %f\n", trainedRecall, recall)

		// recall should be ~90% give or take 2%
		if recall < 0.9-0.02 {
			t.Errorf("Recall should be at least 88%%: got %f", recall)
		}
	})

	// Test 11: Trained Query Metadata
	t.Run("test_11_trained_query_metadata", func(t *testing.T) {
		results := make([]*cyborgdb.QueryResponse, 0)

		for _, metadataQuery := range metadataQueries {
			nProbesVal := int32(24)
			queryParams := cyborgdb.QueryParams{
				BatchQueryVectors: queries,
				TopK:              100,
				NProbes:           &nProbesVal,
				Filters:           metadataQuery.(map[string]interface{}),
			}
			queryResult, err := index.Query(ctx, queryParams)
			if err != nil {
				t.Errorf("Failed to query with metadata: %v", err)
			}
			results = append(results, queryResult)
		}
		metadataQueries[6] = map[string]interface{}{"number": 0}

		recalls := checkMetadataResults(results, trainedMetadataNeighbors, trainedMetadataMatches, numQueries)

		fmt.Printf("Number of recall values: %d\n", len(recalls))

		baseThresholds := []float64{
			94.04,  // Query #1
			100.00, // Query #2
			91.05,  // Query #3
			88.24,  // Query #4
			100.00, // Query #5
			78.88,  // Query #6
			100.00, // Query #7
			92.35,  // Query #8
			91.66,  // Query #9
			88.38,  // Query #10
			88.26,  // Query #11
			94.04,  // Query #12
			90.05,  // Query #13
			74.09,  // Query #14
			9.00,   // Query #15
		}

		// For additional recalls, use a default threshold of 70%
		for i := len(baseThresholds); i < len(recalls); i++ {
			baseThresholds = append(baseThresholds, 70.00)
		}

		expectedThresholds := make([]float64, len(baseThresholds))
		for i, threshold := range baseThresholds {
			expectedThresholds[i] = threshold * 0.95
		}

		if len(recalls) != len(expectedThresholds) {
			t.Errorf("Mismatch in number of recalls (%d) and thresholds (%d)", len(recalls), len(expectedThresholds))
		}

		// Check each recall against its threshold
		failingRecalls := make([]string, 0)

		for idx, recall := range recalls {
			recallPercentage := recall * 100
			threshold := expectedThresholds[idx]

			if idx < 15 {
				fmt.Printf("\nMetadata Query #%d\n", idx+1)
				fmt.Printf("Metadata filters: %v\n", metadataQueries[idx])
				fmt.Printf("Number of candidates: %d / %d\n", len(trainedMetadataNeighbors[idx]), totalNumVectors)
				fmt.Printf("Mean recall: %.2f%%\n", recallPercentage)
				fmt.Printf("Expected threshold: %.2f%%\n", threshold)
			} else {
				fmt.Printf("\nAdditional Query #%d\n", idx+1)
				fmt.Printf("Mean recall: %.2f%%\n", recallPercentage)
				fmt.Printf("Expected threshold: %.2f%%\n", threshold)
			}

			if recallPercentage < threshold {
				failingRecalls = append(failingRecalls, fmt.Sprintf("Query #%d: recall %.2f%% < threshold %.2f%%",
					idx+1, recallPercentage, threshold))
			}
		}

		if len(failingRecalls) > 0 {
			t.Errorf("Some recalls are below their thresholds:\n%s", failingRecalls)
		}
	})

	// Test 12: Trained Query Metadata Auto N_Probes
	t.Run("test_12_trained_query_metadata_auto_n_probes", func(t *testing.T) {
		results := make([]*cyborgdb.QueryResponse, 0)

		for _, metadataQuery := range metadataQueries {
			queryParams := cyborgdb.QueryParams{
				BatchQueryVectors: queries,
				TopK:              100,
				// NProbes not set - will use auto
				Filters: metadataQuery.(map[string]interface{}),
			}
			queryResult, err := index.Query(ctx, queryParams)
			if err != nil {
				t.Errorf("Failed to query with metadata: %v", err)
			}
			results = append(results, queryResult)
		}
		metadataQueries[6] = map[string]interface{}{"number": 0}

		recalls := checkMetadataResults(results, trainedMetadataNeighbors, trainedMetadataMatches, numQueries)

		fmt.Printf("Number of recall values: %d\n", len(recalls))

		baseThresholds := []float64{
			94.04,  // Query #1
			100.00, // Query #2
			91.05,  // Query #3
			88.24,  // Query #4
			100.00, // Query #5
			78.88,  // Query #6
			100.00, // Query #7
			92.35,  // Query #8
			91.66,  // Query #9
			88.38,  // Query #10
			88.26,  // Query #11
			94.04,  // Query #12
			90.05,  // Query #13
			74.09,  // Query #14
			9.00,   // Query #15
		}

		// For additional recalls, use a default threshold of 70%
		for i := len(baseThresholds); i < len(recalls); i++ {
			baseThresholds = append(baseThresholds, 70.00)
		}

		// Apply a 10% reduction to the base thresholds
		expectedThresholds := make([]float64, len(baseThresholds))
		for i, threshold := range baseThresholds {
			expectedThresholds[i] = threshold * 0.90
		}

		if len(recalls) != len(expectedThresholds) {
			t.Errorf("Mismatch in number of recalls (%d) and thresholds (%d)", len(recalls), len(expectedThresholds))
		}

		// Check each recall against its threshold
		failingRecalls := make([]string, 0)

		for idx, recall := range recalls {
			recallPercentage := recall * 100
			threshold := expectedThresholds[idx]

			if idx < 15 {
				fmt.Printf("\nMetadata Query #%d\n", idx+1)
				fmt.Printf("Metadata filters: %v\n", metadataQueries[idx])
				fmt.Printf("Number of candidates: %d / %d\n", len(trainedMetadataNeighbors[idx]), totalNumVectors)
				fmt.Printf("Mean recall: %.2f%%\n", recallPercentage)
				fmt.Printf("Expected threshold: %.2f%%\n", threshold)
			} else {
				fmt.Printf("\nAdditional Query #%d\n", idx+1)
				fmt.Printf("Mean recall: %.2f%%\n", recallPercentage)
				fmt.Printf("Expected threshold: %.2f%%\n", threshold)
			}

			if recallPercentage < threshold {
				failingRecalls = append(failingRecalls, fmt.Sprintf("Query #%d: recall %.2f%% < threshold %.2f%%",
					idx+1, recallPercentage, threshold))
			}
		}

		if len(failingRecalls) > 0 {
			t.Errorf("Some recalls are below their thresholds:\n%s", failingRecalls)
		}
	})

	// Test 13: Trained Get
	t.Run("test_13_trained_get", func(t *testing.T) {
		numGet := 1000
		getIndices := make([]int, 0, numGet)
		usedIndices := make(map[int]bool)

		for len(getIndices) < numGet {
			idx := rand.Intn(numUntrainedVectors)
			if !usedIndices[idx] {
				usedIndices[idx] = true
				getIndices = append(getIndices, idx)
			}
		}

		getIndicesStr := make([]string, numGet)
		for i, idx := range getIndices {
			getIndicesStr[i] = strconv.Itoa(idx)
		}

		include := []string{"vector", "contents", "metadata"}
		getResults, err := index.Get(ctx, getIndicesStr, include)
		if err != nil {
			t.Errorf("Failed to get vectors: %v", err)
		}

		for i, getResult := range getResults.Results {
			if getResult.GetId() != getIndicesStr[i] {
				t.Errorf("ID mismatch: %s != %s", getResult.GetId(), getIndicesStr[i])
			}

			// Check vector equality
			expectedVector := vectors[getIndices[i]]
			resultVector := getResult.GetVector()
			if len(resultVector) != len(expectedVector) {
				t.Errorf("Vector length mismatch for index %d", i)
			}
			for j := range expectedVector {
				if resultVector[j] != expectedVector[j] {
					t.Errorf("Vector mismatch for index %d, element %d", i, j)
					break
				}
			}

			// Check metadata equality
			metadataStr1, _ := json.Marshal(getResult.GetMetadata())
			metadataStr2, _ := json.Marshal(metadata[getIndices[i]])
			if string(metadataStr1) != string(metadataStr2) {
				t.Errorf("Metadata mismatch for index %d", i)
			}
		}
	})

	// Test 14: Delete
	t.Run("test_14_delete", func(t *testing.T) {
		idsToDelete := make([]string, numUntrainedVectors)
		for i := 0; i < numUntrainedVectors; i++ {
			idsToDelete[i] = strconv.Itoa(i)
		}

		err := index.Delete(ctx, idsToDelete)
		if err != nil {
			t.Errorf("Failed to delete vectors: %v", err)
		}

		// Wait for 1 second to ensure delete is processed
		time.Sleep(1 * time.Second)

		// Check if the index has deleted the IDs
		results, err := index.ListIDs(ctx)
		if err != nil {
			t.Errorf("Failed to list IDs: %v", err)
		}

		for _, deletedID := range idsToDelete {
			for _, id := range results.Ids {
				if id == deletedID {
					t.Errorf("ID %s was not deleted", deletedID)
				}
			}
		}
	})

	// Test 15: Get Deleted
	t.Run("test_15_get_deleted", func(t *testing.T) {
		numGet := 1000
		getIndices := make([]int, 0, numGet)
		usedIndices := make(map[int]bool)

		for len(getIndices) < numGet {
			idx := rand.Intn(numUntrainedVectors)
			if !usedIndices[idx] {
				usedIndices[idx] = true
				getIndices = append(getIndices, idx)
			}
		}

		getIndicesStr := make([]string, numGet)
		for i, idx := range getIndices {
			getIndicesStr[i] = strconv.Itoa(idx)
		}

		include := []string{"vector", "contents", "metadata"}
		getResults, err := index.Get(ctx, getIndicesStr, include)
		if err != nil {
			t.Errorf("Failed to get vectors: %v", err)
		}

		if len(getResults.Results) != 0 {
			t.Errorf("Expected 0 results for deleted items, got %d", len(getResults.Results))
		}
	})

	// Test 16: Query Deleted
	t.Run("test_16_query_deleted", func(t *testing.T) {
		nProbesVal := int32(24)
		queryParams := cyborgdb.QueryParams{
			BatchQueryVectors: queries,
			TopK:              100,
			NProbes:           &nProbesVal,
		}
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Errorf("Failed to query: %v", err)
		}

		resultsData := results.GetResults()
		var queryResults [][]cyborgdb.QueryResultItem
		if resultsData.ArrayOfQueryResultItem != nil {
			queryResults = [][]cyborgdb.QueryResultItem{*resultsData.ArrayOfQueryResultItem}
		} else if resultsData.ArrayOfArrayOfQueryResultItem != nil {
			queryResults = *resultsData.ArrayOfArrayOfQueryResultItem
		}

		for _, result := range queryResults {
			for _, queryResult := range result {
				id, _ := strconv.Atoi(queryResult.GetId())
				if id < numUntrainedVectors {
					t.Errorf("Deleted ID %d found in query results", id)
				}
			}
		}
	})

	// Test 17: List Indexes
	t.Run("test_17_list_indexes", func(t *testing.T) {
		indexes, err := client.ListIndexes(ctx)
		if err != nil {
			t.Errorf("Failed to list indexes: %v", err)
		}

		if len(indexes) == 0 {
			t.Errorf("No indexes found")
		}

		// Check if the created index is in the list
		found := false
		for _, idx := range indexes {
			if idx == indexName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Index %s not found in the list of indexes", indexName)
		}
	})

	// Test 18: Index Properties
	t.Run("test_18_index_properties", func(t *testing.T) {
		name := index.GetIndexName()
		if name != indexName {
			t.Errorf("Index name does not match: expected %s, got %s", indexName, name)
		}

		indexType := index.GetIndexType()
		if indexType != "ivfflat" {
			t.Errorf("Index type is not IVFFlat: got %s", indexType)
		}

		config := index.GetIndexConfig()
		// Check if config is empty (all fields are nil)
		if config.IndexIVFFlatModel == nil && config.IndexIVFModel == nil && config.IndexIVFPQModel == nil {
			t.Errorf("Index config is empty")
		}
	})

	// Test 19: Load Index
	t.Run("test_19_load_index", func(t *testing.T) {
		loadedKeyBytes, err := hex.DecodeString(indexKey)
		if err != nil {
			t.Errorf("Failed to decode index key: %v", err)
		}

		loadedIndex, err := client.LoadIndex(ctx, indexName, loadedKeyBytes)
		if err != nil {
			t.Errorf("Failed to load index: %v", err)
		}

		loadedName := loadedIndex.GetIndexName()
		if loadedName != indexName {
			t.Errorf("Loaded index name does not match: expected %s, got %s", indexName, loadedName)
		}
	})
}
