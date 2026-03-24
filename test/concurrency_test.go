/*
Concurrency and Multi-Index Tests for CyborgDB Go SDK

Tests thread safety, data integrity under concurrent load, and index isolation.
All tests hit a real backend — no mocking. Each test is fully isolated: creates
its own client, indexes, and data, then cleans up. Tests can run in parallel.

Test inventory and what each catches:

  - TestConcurrentUpsertsNoDataLoss: Dropped writes / request body corruption
    in shared HTTP client under concurrent goroutines.

  - TestConcurrentUpsertsOverlappingIDs: Byte-level vector corruption from
    interleaved writes to the same keys.

  - TestQueriesDuringUpserts: Malformed responses or crashes from concurrent
    read/write HTTP access through shared connection pool.

  - TestDeletesDuringQueries: Server-side race between delete and read paths
    causing crashes or garbled results.

  - TestConcurrentUpsertsAndDeletesOnSameIDs: Ghost entries or partial state
    from write-delete races on identical keys.

  - TestBadGoroutineDoesntBreakGoodGoroutines: Error responses poisoning shared
    HTTP connection pool state (Go net/http connection reuse after 4xx).

  - TestNoDataLeakageBetweenIndexes: Cross-index contamination from incorrect
    index_name routing in query requests.

  - TestDeleteInOneIndexDoesntAffectOthers: Cross-index contamination from
    incorrect index_name routing in delete requests (write-path isolation).

  - TestConcurrentWritesToDifferentIndexes: index_name mix-up in request
    serialization when concurrent goroutines target different indexes through
    the same shared Client.

  - TestStress20Goroutines200VectorsEach: Connection pool exhaustion, deadlocks,
    and performance cliffs under high goroutine counts (20) and large payloads
    (4,000 total vectors).
*/
package test

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

const (
	concDimension  = 128
	concNumVectors = 50 // Per-goroutine/per-index vector count
	concTimeout    = 120 * time.Second
)

func concBaseURL() string {
	u := os.Getenv("CYBORGDB_BASE_URL")
	if u == "" {
		return "http://localhost:8000"
	}
	return u
}

func concAPIKey() string {
	return os.Getenv("CYBORGDB_API_KEY")
}

// newIsolatedClient creates a fresh CyborgDB client. Safe to call from any goroutine.
func newIsolatedClient(t *testing.T) *cyborgdb.Client {
	t.Helper()
	client, err := cyborgdb.NewClient(concBaseURL(), concAPIKey())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}

// newIsolatedIndex creates a uniquely-named IVFFlat index with its own cleanup.
// Must be called from the test goroutine (uses t.Fatalf). Registers cleanup via t.Cleanup.
func newIsolatedIndex(t *testing.T, client *cyborgdb.Client, prefix string) (*cyborgdb.EncryptedIndex, string, []byte) {
	t.Helper()
	name := generateUniqueName(prefix + "_")
	key := generateRandomKey()
	metric := "euclidean"
	config := cyborgdb.IndexIVFFlat(int32(concDimension))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:   name,
		IndexKey:    key,
		IndexConfig: config,
		Metric:      &metric,
	})
	if err != nil {
		t.Fatalf("Failed to create index %s: %v", name, err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = index.DeleteIndex(ctx)
	})
	return index, name, key
}

// generateRandomVectors generates random float32 vectors.
func generateRandomVectors(count, dimension int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			vectors[i][j] = rand.Float32()
		}
	}
	return vectors
}

// concUpsertBatch upserts a batch of random vectors. Returns error instead of calling
// t.Fatal, making it safe to call from any goroutine.
func concUpsertBatch(index *cyborgdb.EncryptedIndex, idPrefix string, count int) ([]string, [][]float32, error) {
	vectors := generateRandomVectors(count, concDimension)
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = fmt.Sprintf("%s_%d", idPrefix, i)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := index.UpsertVectors(ctx, ids, vectors, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("upsertBatch(%s): %w", idPrefix, err)
	}
	return ids, vectors, nil
}

// seedIndex upserts seed data from the test goroutine. Calls t.Fatalf on error.
func seedIndex(t *testing.T, index *cyborgdb.EncryptedIndex, prefix string, count int) {
	t.Helper()
	_, _, err := concUpsertBatch(index, prefix, count)
	if err != nil {
		t.Fatalf("seedIndex failed: %v", err)
	}
}

// vectorsApproxEqual checks if two float32 vectors are approximately equal.
func vectorsApproxEqual(a, b []float32, rtol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		diff := math.Abs(float64(a[i]) - float64(b[i]))
		limit := rtol * math.Max(math.Abs(float64(a[i])), math.Abs(float64(b[i])))
		if diff > limit+1e-8 {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Concurrent Operations — Single Index
// ---------------------------------------------------------------------------

func TestConcurrentUpsertsNoDataLoss(t *testing.T) {
	t.Parallel()
	// 10 goroutines each upsert 50 vectors (500 total) through one shared
	// EncryptedIndex. After all finish, every single ID must be present.
	// Catches: request body corruption in shared client, dropped writes.
	client := newIsolatedClient(t)
	index, _, _ := newIsolatedIndex(t, client, "conc_upsert")

	numGoroutines := 10
	var mu sync.Mutex
	var allIDs []string
	var errs []error
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			ids, _, err := concUpsertBatch(index, fmt.Sprintf("t%d", goroutineID), concNumVectors)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			allIDs = append(allIDs, ids...)
		}(i)
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("Goroutines raised errors: %v", errs)
	}

	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
	defer cancel()
	resp, err := index.ListIDs(ctx)
	if err != nil {
		t.Fatalf("ListIDs failed: %v", err)
	}

	storedIDs := make(map[string]bool, len(resp.Ids))
	for _, id := range resp.Ids {
		storedIDs[id] = true
	}

	var missing []string
	for _, id := range allIDs {
		if !storedIDs[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d/%d IDs missing after concurrent upsert", len(missing), len(allIDs))
	}
}

func TestConcurrentUpsertsOverlappingIDs(t *testing.T) {
	t.Parallel()
	// 5 goroutines upsert different vectors to the SAME 20 IDs.
	// After completion: each ID must exist, and the stored vector must
	// exactly match one of the 5 written vectors (proving no corruption
	// from interleaved writes).
	client := newIsolatedClient(t)
	index, _, _ := newIsolatedIndex(t, client, "conc_overlap")

	numIDs := 20
	numGoroutines := 5
	sharedIDs := make([]string, numIDs)
	for i := 0; i < numIDs; i++ {
		sharedIDs[i] = fmt.Sprintf("overlap_%d", i)
	}

	var mu sync.Mutex
	writtenVectors := make(map[string][][]float32)
	for _, id := range sharedIDs {
		writtenVectors[id] = make([][]float32, 0, numGoroutines)
	}
	var errs []error
	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			vectors := generateRandomVectors(numIDs, concDimension)
			mu.Lock()
			for i, id := range sharedIDs {
				vecCopy := make([]float32, len(vectors[i]))
				copy(vecCopy, vectors[i])
				writtenVectors[id] = append(writtenVectors[id], vecCopy)
			}
			mu.Unlock()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := index.UpsertVectors(ctx, sharedIDs, vectors, nil); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("goroutine %d: %w", goroutineID, err))
				mu.Unlock()
			}
		}(g)
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("Goroutines raised errors: %v", errs)
	}
	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
	defer cancel()
	resp, err := index.Get(ctx, sharedIDs, []string{"vector"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(resp.Results) != numIDs {
		t.Fatalf("Expected %d vectors, got %d", numIDs, len(resp.Results))
	}
	for _, item := range resp.Results {
		candidates := writtenVectors[item.GetId()]
		storedVec := item.GetVector()
		matched := false
		for _, c := range candidates {
			if vectorsApproxEqual(storedVec, c, 1e-5) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("ID '%s': stored vector doesn't match ANY of the %d written vectors — possible corruption",
				item.GetId(), len(candidates))
		}
	}
}

// ---------------------------------------------------------------------------
// Concurrent Reads and Writes
// ---------------------------------------------------------------------------

func TestQueriesDuringUpserts(t *testing.T) {
	t.Parallel()
	// 3 writer goroutines upsert while 5 reader goroutines query concurrently.
	// Readers must get well-formed results with valid distances.
	// Catches: crashes from concurrent HTTP access, malformed responses.
	client := newIsolatedClient(t)
	index, _, _ := newIsolatedIndex(t, client, "conc_rw")

	// Seed with initial data so queries have something to return
	seedIndex(t, index, "seed", 100)
	time.Sleep(1 * time.Second)

	numWriters := 3
	numReaders := 5
	queryCount := 10
	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup

	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for batch := 0; batch < 3; batch++ {
				ids := make([]string, 20)
				for i := 0; i < 20; i++ {
					ids[i] = fmt.Sprintf("w%d_b%d_%d", writerID, batch, i)
				}
				vectors := generateRandomVectors(20, concDimension)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := index.UpsertVectors(ctx, ids, vectors, nil)
				cancel()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("writer %d batch %d: %w", writerID, batch, err))
					mu.Unlock()
					return
				}
			}
		}(w)
	}

	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for q := 0; q < queryCount; q++ {
				qv := generateRandomVectors(1, concDimension)[0]
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				results, err := index.Query(ctx, cyborgdb.QueryParams{
					QueryVector: qv,
					TopK:        5,
					Include:     []string{"metadata"},
				})
				cancel()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("reader %d: %w", readerID, err))
					mu.Unlock()
					return
				}
				items := getQueryResultItems(&results.Results)
				for _, item := range items {
					if item.GetId() == "" {
						mu.Lock()
						errs = append(errs, fmt.Errorf("reader %d: result missing ID", readerID))
						mu.Unlock()
					}
					if item.GetDistance() < 0 {
						mu.Lock()
						errs = append(errs, fmt.Errorf("reader %d: negative distance %f for ID %s",
							readerID, item.GetDistance(), item.GetId()))
						mu.Unlock()
					}
				}
			}
		}(r)
	}

	wg.Wait()
	if len(errs) > 0 {
		t.Fatalf("Concurrent read/write errors: %v", errs)
	}
}

func TestDeletesDuringQueries(t *testing.T) {
	t.Parallel()
	// One goroutine deletes vectors in batches while 4 goroutines query.
	// Queries must never crash or return malformed results.
	// Catches: server-side race between delete and read paths.
	client := newIsolatedClient(t)
	index, _, _ := newIsolatedIndex(t, client, "conc_delq")

	// Seed data so queries always have something to return
	seedIndex(t, index, "seed", 100)

	deleteCount := 30
	deleteIDs := make([]string, deleteCount)
	for i := 0; i < deleteCount; i++ {
		deleteIDs[i] = fmt.Sprintf("del_%d", i)
	}
	deleteVectors := generateRandomVectors(deleteCount, concDimension)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := index.UpsertVectors(ctx, deleteIDs, deleteVectors, nil)
	cancel()
	if err != nil {
		t.Fatalf("Failed to upsert delete targets: %v", err)
	}
	time.Sleep(1 * time.Second)

	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup

	// Deleter
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < deleteCount; i += 5 {
			end := i + 5
			if end > deleteCount {
				end = deleteCount
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := index.Delete(ctx, deleteIDs[i:end])
			cancel()
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("deleter batch %d: %w", i, err))
				mu.Unlock()
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Queriers
	for q := 0; q < 4; q++ {
		wg.Add(1)
		go func(queryID int) {
			defer wg.Done()
			for i := 0; i < 15; i++ {
				qv := generateRandomVectors(1, concDimension)[0]
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				results, err := index.Query(ctx, cyborgdb.QueryParams{
					QueryVector: qv,
					TopK:        10,
					Include:     []string{"metadata"},
				})
				cancel()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("querier %d: %w", queryID, err))
					mu.Unlock()
					return
				}
				items := getQueryResultItems(&results.Results)
				for _, item := range items {
					if item.GetId() == "" {
						mu.Lock()
						errs = append(errs, fmt.Errorf("querier %d: result missing ID", queryID))
						mu.Unlock()
					}
					if item.GetDistance() < 0 {
						mu.Lock()
						errs = append(errs, fmt.Errorf("querier %d: negative distance", queryID))
						mu.Unlock()
					}
				}
			}
		}(q)
	}

	wg.Wait()
	if len(errs) > 0 {
		t.Fatalf("Delete-during-query errors: %v", errs)
	}
}

func TestConcurrentUpsertsAndDeletesOnSameIDs(t *testing.T) {
	t.Parallel()
	// 2 goroutines upsert a set of IDs while 2 other goroutines delete from
	// the same set. After all finish, every surviving ID must have a valid
	// vector — no ghost entries or truncated state.
	// Catches: write-delete races causing ghost entries or corrupt vectors.
	client := newIsolatedClient(t)
	index, _, _ := newIsolatedIndex(t, client, "conc_race")

	targetCount := 40
	targetIDs := make([]string, targetCount)
	for i := 0; i < targetCount; i++ {
		targetIDs[i] = fmt.Sprintf("race_%d", i)
	}
	vectors := generateRandomVectors(targetCount, concDimension)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := index.UpsertVectors(ctx, targetIDs, vectors, nil)
	cancel()
	if err != nil {
		t.Fatalf("Initial upsert failed: %v", err)
	}
	time.Sleep(1 * time.Second)

	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup

	// Upserters
	for u := 0; u < 2; u++ {
		wg.Add(1)
		go func(upserterID int) {
			defer wg.Done()
			for round := 0; round < 5; round++ {
				newVecs := generateRandomVectors(targetCount, concDimension)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := index.UpsertVectors(ctx, targetIDs, newVecs, nil)
				cancel()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("upserter %d round %d: %w", upserterID, round, err))
					mu.Unlock()
					return
				}
			}
		}(u)
	}

	// Deleters
	for d := 0; d < 2; d++ {
		wg.Add(1)
		go func(deleterID int) {
			defer wg.Done()
			for round := 0; round < 5; round++ {
				batch := targetIDs[deleterID*10 : (deleterID+1)*10]
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := index.Delete(ctx, batch)
				cancel()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("deleter %d round %d: %w", deleterID, round, err))
					mu.Unlock()
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}(d)
	}

	wg.Wait()
	if len(errs) > 0 {
		t.Fatalf("Upsert/delete race errors: %v", errs)
	}

	time.Sleep(1 * time.Second)

	ctx2, cancel2 := context.WithTimeout(context.Background(), concTimeout)
	defer cancel2()
	resp, err := index.ListIDs(ctx2)
	if err != nil {
		t.Fatalf("ListIDs failed: %v", err)
	}

	if len(resp.Ids) == 0 {
		t.Fatal("All IDs gone after upsert/delete race — upserters never committed or deleters swept everything")
	}

	// Every surviving ID must have a valid, retrievable vector
	getResp, err := index.Get(ctx2, resp.Ids, []string{"vector"})
	if err != nil {
		t.Fatalf("Get surviving IDs failed: %v", err)
	}
	for _, item := range getResp.Results {
		vec := item.GetVector()
		if vec == nil {
			t.Errorf("ID '%s' exists but has no vector — ghost entry", item.GetId())
		} else if len(vec) != concDimension {
			t.Errorf("ID '%s' has wrong dimension: got %d, want %d — truncated write", item.GetId(), len(vec), concDimension)
		}
	}
}

// ---------------------------------------------------------------------------
// Error Isolation Under Load
// ---------------------------------------------------------------------------

func TestBadGoroutineDoesntBreakGoodGoroutines(t *testing.T) {
	t.Parallel()
	// One goroutine sends wrong-dimension vectors (expects errors).
	// 4 other goroutines do valid queries through the same shared client.
	// Good goroutines must succeed — proving error handling doesn't poison
	// shared HTTP connection pool state.
	client := newIsolatedClient(t)
	index, _, _ := newIsolatedIndex(t, client, "conc_errisolation")

	seedIndex(t, index, "base", 50)
	time.Sleep(1 * time.Second)

	var mu sync.Mutex
	var goodResults []int
	var badErrors []error
	var goodErrors []error
	var wg sync.WaitGroup

	// Bad worker: sends wrong dimension vectors
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			wrongDimVecs := generateRandomVectors(10, 64) // Wrong dimension
			ids := make([]string, 10)
			for j := 0; j < 10; j++ {
				ids[j] = fmt.Sprintf("bad_%d_%d", i, j)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := index.UpsertVectors(ctx, ids, wrongDimVecs, nil)
			cancel()
			if err != nil {
				mu.Lock()
				badErrors = append(badErrors, err)
				mu.Unlock()
			}
		}
	}()

	// Good workers: valid queries
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for q := 0; q < 10; q++ {
				qv := generateRandomVectors(1, concDimension)[0]
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				results, err := index.Query(ctx, cyborgdb.QueryParams{
					QueryVector: qv,
					TopK:        3,
					Include:     []string{"metadata"},
				})
				cancel()
				if err != nil {
					mu.Lock()
					goodErrors = append(goodErrors, fmt.Errorf("goroutine %d query %d: %w", goroutineID, q, err))
					mu.Unlock()
					return
				}
				items := getQueryResultItems(&results.Results)
				mu.Lock()
				goodResults = append(goodResults, len(items))
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()

	if len(badErrors) == 0 {
		t.Error("Bad worker should have failed at least once — wrong dimension was accepted")
	}
	if len(goodErrors) > 0 {
		t.Fatalf("Good workers failed due to bad worker poisoning shared state: %v", goodErrors)
	}
	if len(goodResults) == 0 {
		t.Error("No good results collected — good workers never completed a query")
	}
}

// ---------------------------------------------------------------------------
// Multi-Index Tests
// ---------------------------------------------------------------------------

func TestNoDataLeakageBetweenIndexes(t *testing.T) {
	t.Parallel()
	// Creates 3 indexes with unique data. Queries each and verifies every
	// returned ID belongs ONLY to that index.
	// Catches: cross-index contamination from incorrect index_name routing.
	client := newIsolatedClient(t)

	type indexInfo struct {
		index *cyborgdb.EncryptedIndex
		name  string
		ids   map[string]bool
	}

	indexes := make([]indexInfo, 3)
	for i := 0; i < 3; i++ {
		idx, name, _ := newIsolatedIndex(t, client, fmt.Sprintf("iso_%d", i))
		idSet := make(map[string]bool)

		ids := make([]string, 30)
		for j := 0; j < 30; j++ {
			ids[j] = fmt.Sprintf("idx%d_vec%d", i, j)
			idSet[ids[j]] = true
		}
		vectors := generateRandomVectors(30, concDimension)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := idx.UpsertVectors(ctx, ids, vectors, nil)
		cancel()
		if err != nil {
			t.Fatalf("Failed to upsert to index %d: %v", i, err)
		}

		indexes[i] = indexInfo{index: idx, name: name, ids: idSet}
	}

	time.Sleep(2 * time.Second)

	for i, info := range indexes {
		otherIDs := make(map[string]bool)
		for j, other := range indexes {
			if j != i {
				for id := range other.ids {
					otherIDs[id] = true
				}
			}
		}

		for q := 0; q < 5; q++ {
			qv := generateRandomVectors(1, concDimension)[0]
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			results, err := info.index.Query(ctx, cyborgdb.QueryParams{
				QueryVector: qv,
				TopK:        10,
				Include:     []string{"metadata"},
			})
			cancel()
			if err != nil {
				t.Fatalf("Query failed on index %d: %v", i, err)
			}

			items := getQueryResultItems(&results.Results)
			if len(items) == 0 {
				t.Errorf("Index '%s' returned empty results — isolation check is vacuous", info.name)
				continue
			}
			for _, item := range items {
				if otherIDs[item.GetId()] {
					t.Errorf("DATA LEAKAGE: Index '%s' returned ID '%s' from another index", info.name, item.GetId())
				}
				if !info.ids[item.GetId()] {
					t.Errorf("Index '%s' returned unknown ID '%s'", info.name, item.GetId())
				}
			}
		}
	}
}

func TestDeleteInOneIndexDoesntAffectOthers(t *testing.T) {
	t.Parallel()
	// Deleting from index 0 must not remove anything from indexes 1 or 2.
	// Catches: cross-index contamination in the delete write-path (distinct
	// from query-path tested by TestNoDataLeakageBetweenIndexes).
	client := newIsolatedClient(t)

	type indexInfo struct {
		index *cyborgdb.EncryptedIndex
		name  string
	}

	numIndexes := 3
	indexes := make([]indexInfo, numIndexes)
	for i := 0; i < numIndexes; i++ {
		idx, name, _ := newIsolatedIndex(t, client, fmt.Sprintf("deliso_%d", i))

		ids := make([]string, 30)
		for j := 0; j < 30; j++ {
			ids[j] = fmt.Sprintf("delidx%d_%d", i, j)
		}
		vectors := generateRandomVectors(30, concDimension)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := idx.UpsertVectors(ctx, ids, vectors, nil)
		cancel()
		if err != nil {
			t.Fatalf("Failed to upsert to index %d: %v", i, err)
		}

		indexes[i] = indexInfo{index: idx, name: name}
	}

	time.Sleep(2 * time.Second)

	// Snapshot other indexes' IDs before deletion
	otherSnapshots := make(map[int]map[string]bool)
	for i := 1; i < numIndexes; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
		resp, err := indexes[i].index.ListIDs(ctx)
		cancel()
		if err != nil {
			t.Fatalf("ListIDs failed for index %d: %v", i, err)
		}
		snapshot := make(map[string]bool, len(resp.Ids))
		for _, id := range resp.Ids {
			snapshot[id] = true
		}
		otherSnapshots[i] = snapshot
	}

	// Delete from index 0
	ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
	resp0, err := indexes[0].index.ListIDs(ctx)
	cancel()
	if err != nil {
		t.Fatalf("ListIDs failed for index 0: %v", err)
	}
	if len(resp0.Ids) == 0 {
		t.Fatal("Index 0 is empty — nothing to delete")
	}

	toDelete := resp0.Ids
	if len(toDelete) > 15 {
		toDelete = toDelete[:15]
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	err = indexes[0].index.Delete(ctx2, toDelete)
	cancel2()
	if err != nil {
		t.Fatalf("Delete from index 0 failed: %v", err)
	}
	time.Sleep(1 * time.Second)

	// Verify other indexes are unaffected
	for i := 1; i < numIndexes; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
		resp, err := indexes[i].index.ListIDs(ctx)
		cancel()
		if err != nil {
			t.Fatalf("ListIDs failed for index %d after delete: %v", i, err)
		}
		currentIDs := make(map[string]bool, len(resp.Ids))
		for _, id := range resp.Ids {
			currentIDs[id] = true
		}
		snapshot := otherSnapshots[i]

		for id := range snapshot {
			if !currentIDs[id] {
				t.Errorf("Index '%s' lost ID '%s' after deleting from '%s'",
					indexes[i].name, id, indexes[0].name)
			}
		}
		for id := range currentIDs {
			if !snapshot[id] {
				t.Errorf("Index '%s' gained unexpected ID '%s' after deleting from '%s'",
					indexes[i].name, id, indexes[0].name)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Concurrent Multi-Index Writes
// ---------------------------------------------------------------------------

func TestConcurrentWritesToDifferentIndexes(t *testing.T) {
	t.Parallel()
	// 5 goroutines, each writing to its own pre-existing index via the same
	// shared Client. Then verify each index has ONLY its own data with correct
	// vectors.
	// Catches: index_name mix-up in request serialization under concurrency,
	// cross-index writes from goroutine racing on shared client state.
	client := newIsolatedClient(t)

	type indexInfo struct {
		index *cyborgdb.EncryptedIndex
		name  string
	}

	numIndexes := 5
	indexes := make([]indexInfo, numIndexes)
	for i := 0; i < numIndexes; i++ {
		idx, name, _ := newIsolatedIndex(t, client, fmt.Sprintf("cw_%d", i))
		indexes[i] = indexInfo{index: idx, name: name}
	}

	var mu sync.Mutex
	var errs []error
	type goroutineData struct {
		ids     []string
		vectors [][]float32
		name    string
	}
	perGoroutine := make(map[int]*goroutineData)
	var wg sync.WaitGroup

	for i := 0; i < numIndexes; i++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			info := indexes[gID]
			vectors := generateRandomVectors(20, concDimension)
			ids := make([]string, 20)
			for j := 0; j < 20; j++ {
				ids[j] = fmt.Sprintf("cw%d_%d", gID, j)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := info.index.UpsertVectors(ctx, ids, vectors, nil)
			cancel()
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("goroutine %d: %w", gID, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			perGoroutine[gID] = &goroutineData{ids: ids, vectors: vectors, name: info.name}
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("Concurrent write errors: %v", errs)
	}

	time.Sleep(2 * time.Second)

	// Verify each index has ONLY its own data and vectors are intact
	for gID, data := range perGoroutine {
		info := indexes[gID]

		ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
		resp, err := info.index.ListIDs(ctx)
		cancel()
		if err != nil {
			t.Fatalf("ListIDs failed for index %d: %v", gID, err)
		}

		expectedPrefix := fmt.Sprintf("cw%d_", gID)
		storedIDs := make(map[string]bool, len(resp.Ids))
		for _, id := range resp.Ids {
			storedIDs[id] = true
			if len(id) < len(expectedPrefix) || id[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("Index '%s' contains foreign ID '%s' (expected prefix '%s')",
					data.name, id, expectedPrefix)
			}
		}

		for _, id := range data.ids {
			if !storedIDs[id] {
				t.Errorf("Index '%s' missing expected ID '%s'", data.name, id)
			}
		}

		// Spot-check vector integrity: first and last vector
		for _, checkIdx := range []int{0, len(data.ids) - 1} {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
			retrieved, err := info.index.Get(ctx2, []string{data.ids[checkIdx]}, []string{"vector"})
			cancel2()
			if err != nil {
				t.Fatalf("Get failed for index %d, ID %s: %v", gID, data.ids[checkIdx], err)
			}
			if len(retrieved.Results) != 1 {
				t.Errorf("Expected 1 vector, got %d for ID %s", len(retrieved.Results), data.ids[checkIdx])
				continue
			}
			retrievedVec := retrieved.Results[0].GetVector()
			if !vectorsApproxEqual(retrievedVec, data.vectors[checkIdx], 1e-5) {
				t.Errorf("Index '%s', ID '%s': vector mismatch — data routed to wrong index",
					data.name, data.ids[checkIdx])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Scale & Stress
// ---------------------------------------------------------------------------

func TestStress20Goroutines200VectorsEach(t *testing.T) {
	t.Parallel()
	// 20 goroutines each upsert 200 vectors (4,000 total) then query.
	// All queries must return well-formed results, all IDs must be stored.
	// Catches: connection pool exhaustion, deadlocks under high goroutine
	// counts, performance cliffs at scale.
	client := newIsolatedClient(t)
	index, _, _ := newIsolatedIndex(t, client, "stress")

	numGoroutines := 20
	vectorsPerGoroutine := 200
	var mu sync.Mutex
	var allIDs []string
	var errs []error
	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			ids, _, err := concUpsertBatch(index, fmt.Sprintf("stress_%d", goroutineID), vectorsPerGoroutine)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			mu.Lock()
			allIDs = append(allIDs, ids...)
			mu.Unlock()

			// Each goroutine also queries to validate responses under load
			for q := 0; q < 5; q++ {
				qv := generateRandomVectors(1, concDimension)[0]
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				results, err := index.Query(ctx, cyborgdb.QueryParams{
					QueryVector: qv,
					TopK:        10,
					Include:     []string{"metadata"},
				})
				cancel()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("goroutine %d query: %w", goroutineID, err))
					mu.Unlock()
					return
				}
				items := getQueryResultItems(&results.Results)
				for _, item := range items {
					if item.GetId() == "" {
						mu.Lock()
						errs = append(errs, fmt.Errorf("goroutine %d: missing ID in result", goroutineID))
						mu.Unlock()
					}
					if item.GetDistance() < 0 {
						mu.Lock()
						errs = append(errs, fmt.Errorf("goroutine %d: negative distance", goroutineID))
						mu.Unlock()
					}
				}
			}
		}(g)
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("Stress test errors: %v", errs)
	}

	time.Sleep(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
	defer cancel()
	resp, err := index.ListIDs(ctx)
	if err != nil {
		t.Fatalf("ListIDs failed: %v", err)
	}

	storedIDs := make(map[string]bool, len(resp.Ids))
	for _, id := range resp.Ids {
		storedIDs[id] = true
	}
	var missing []string
	for _, id := range allIDs {
		if !storedIDs[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d/%d IDs missing after 20-goroutine stress test", len(missing), len(allIDs))
	}
}

// ---------------------------------------------------------------------------
// Mixed Index Types — One Client, Three Index Types
// Targets a reported bug where one client managing multiple index types
// (IVFFlat, IVFPQ, IVFSQ) simultaneously causes cross-type contamination
// or dispatch errors.
// ---------------------------------------------------------------------------

// mixedIndexSet holds the three index types created from one client.
type mixedIndexSet struct {
	client *cyborgdb.Client
	flat   *cyborgdb.EncryptedIndex
	pq     *cyborgdb.EncryptedIndex
	sq     *cyborgdb.EncryptedIndex
}

// newMixedIndexSet creates one client with three index types and registers cleanup.
func newMixedIndexSet(t *testing.T) *mixedIndexSet {
	t.Helper()
	client := newIsolatedClient(t)
	metric := "euclidean"
	dim := int32(concDimension)

	create := func(prefix string, config cyborgdb.IndexModel) *cyborgdb.EncryptedIndex {
		t.Helper()
		name := generateUniqueName(prefix + "_")
		key := generateRandomKey()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		idx, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
			IndexName:   name,
			IndexKey:    key,
			IndexConfig: config,
			Metric:      &metric,
		})
		if err != nil {
			t.Fatalf("Failed to create %s index: %v", prefix, err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = idx.DeleteIndex(ctx)
		})
		return idx
	}

	return &mixedIndexSet{
		client: client,
		flat:   create("mixed_flat", cyborgdb.IndexIVFFlat(dim)),
		pq:     create("mixed_pq", cyborgdb.IndexIVFPQ(dim, 32, 8)),
		sq:     create("mixed_sq", cyborgdb.IndexIVFSQ(dim, 8)),
	}
}

func TestMixedIndexTypesUpsertAndQuery(t *testing.T) {
	t.Parallel()
	// One client creates IVFFlat, IVFPQ, IVFSQ indexes. Upserts 20 vectors
	// to each, then queries each and verifies:
	// - Correct index_type is reported
	// - No returned IDs belong to another index type
	// Catches: client dispatching requests to wrong index type, index_type
	// metadata being overwritten when multiple types share a client.
	m := newMixedIndexSet(t)
	ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
	defer cancel()

	type indexEntry struct {
		index       *cyborgdb.EncryptedIndex
		typeName    string
		idPrefix    string
		vectors     [][]float32
		expectedIDs map[string]bool
	}

	entries := []indexEntry{
		{m.flat, "ivfflat", "flat", nil, make(map[string]bool)},
		{m.pq, "ivfpq", "pq", nil, make(map[string]bool)},
		{m.sq, "ivfsq", "sq", nil, make(map[string]bool)},
	}

	// Upsert 20 vectors to each
	for i := range entries {
		e := &entries[i]
		e.vectors = generateRandomVectors(20, concDimension)
		items := make(cyborgdb.VectorItems, 20)
		for j := 0; j < 20; j++ {
			id := fmt.Sprintf("%s_%d", e.idPrefix, j)
			e.expectedIDs[id] = true
			items[j] = cyborgdb.VectorItem{Id: id, Vector: e.vectors[j]}
		}
		if err := e.index.Upsert(ctx, items); err != nil {
			t.Fatalf("Upsert to %s index failed: %v", e.typeName, err)
		}
	}

	// Collect all IDs across all types for cross-contamination check
	allOtherIDs := make([]map[string]bool, len(entries))
	for i := range entries {
		allOtherIDs[i] = make(map[string]bool)
		for j, other := range entries {
			if j != i {
				for id := range other.expectedIDs {
					allOtherIDs[i][id] = true
				}
			}
		}
	}

	// Wait for propagation then query each
	time.Sleep(2 * time.Second)

	for i, e := range entries {
		// Verify index type
		if e.index.GetIndexType() != e.typeName {
			t.Errorf("Expected index type '%s', got '%s'", e.typeName, e.index.GetIndexType())
		}

		// Query and check isolation
		qv := e.vectors[0]
		var resultItems []internal.QueryResultItem
		ok := pollUntil(30*time.Second, 2*time.Second, func() bool {
			results, err := e.index.Query(ctx, cyborgdb.QueryParams{
				QueryVector: qv, TopK: 10,
			})
			if err != nil || results == nil {
				return false
			}
			resultItems = getQueryResultItems(&results.Results)
			return len(resultItems) > 0
		})
		if !ok {
			t.Fatalf("%s index query returned no results within 30s", e.typeName)
		}

		for _, item := range resultItems {
			if allOtherIDs[i][item.GetId()] {
				t.Errorf("CROSS-TYPE LEAKAGE: %s index returned ID '%s' from another index type",
					e.typeName, item.GetId())
			}
			if !e.expectedIDs[item.GetId()] {
				t.Errorf("%s index returned unknown ID '%s'", e.typeName, item.GetId())
			}
		}
	}
}

func TestMixedIndexTypesConcurrentWrites(t *testing.T) {
	t.Parallel()
	// 3 goroutines each write to a different index type (flat, pq, sq)
	// through the same shared client concurrently. Then verify:
	// - Each index has only its own data
	// - IVFFlat: exact vector match (lossless)
	// - IVFPQ/IVFSQ: correct dimensions and finite values (lossy compression)
	// Catches: client-level race condition when concurrent goroutines target
	// different index types, request body mix-up across types.
	m := newMixedIndexSet(t)
	ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
	defer cancel()

	type goroutineResult struct {
		typeName string
		idPrefix string
		ids      []string
		vectors  [][]float32
		index    *cyborgdb.EncryptedIndex
	}

	targets := []struct {
		index    *cyborgdb.EncryptedIndex
		typeName string
		prefix   string
	}{
		{m.flat, "ivfflat", "cwflat"},
		{m.pq, "ivfpq", "cwpq"},
		{m.sq, "ivfsq", "cwsq"},
	}

	var mu sync.Mutex
	var errs []error
	results := make(map[string]*goroutineResult)
	var wg sync.WaitGroup

	for _, tgt := range targets {
		wg.Add(1)
		go func(idx *cyborgdb.EncryptedIndex, typeName, prefix string) {
			defer wg.Done()
			vectors := generateRandomVectors(20, concDimension)
			ids := make([]string, 20)
			for j := 0; j < 20; j++ {
				ids[j] = fmt.Sprintf("%s_%d", prefix, j)
			}

			writeCtx, writeCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer writeCancel()
			if err := idx.UpsertVectors(writeCtx, ids, vectors, nil); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s upsert: %w", typeName, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			results[typeName] = &goroutineResult{
				typeName: typeName, idPrefix: prefix,
				ids: ids, vectors: vectors, index: idx,
			}
			mu.Unlock()
		}(tgt.index, tgt.typeName, tgt.prefix)
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("Concurrent writes failed: %v", errs)
	}

	time.Sleep(2 * time.Second)

	for typeName, res := range results {
		// Verify each index has only its own IDs
		listResp, err := res.index.ListIDs(ctx)
		if err != nil {
			t.Fatalf("ListIDs failed for %s: %v", typeName, err)
		}

		storedIDs := make(map[string]bool, len(listResp.Ids))
		for _, id := range listResp.Ids {
			storedIDs[id] = true
		}
		for _, id := range res.ids {
			if !storedIDs[id] {
				t.Errorf("%s index missing ID '%s'", typeName, id)
			}
		}

		// Verify vector integrity: spot-check first vector
		getResp, err := res.index.Get(ctx, []string{res.ids[0]}, []string{"vector"})
		if err != nil {
			t.Fatalf("Get failed for %s: %v", typeName, err)
		}
		if len(getResp.Results) != 1 {
			t.Fatalf("%s: expected 1 result, got %d", typeName, len(getResp.Results))
		}

		retrieved := getResp.Results[0].GetVector()
		if len(retrieved) != concDimension {
			t.Errorf("%s: vector dimension mismatch: got %d, want %d", typeName, len(retrieved), concDimension)
			continue
		}

		if typeName == "ivfflat" {
			// Lossless — exact match expected
			if !vectorsApproxEqual(retrieved, res.vectors[0], 1e-5) {
				t.Errorf("ivfflat: vector mismatch on first vector — data routed to wrong index")
			}
		} else {
			// IVFPQ/IVFSQ are lossy — just verify finite values (no NaN/Inf)
			for j, v := range retrieved {
				if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
					t.Errorf("%s: vector element %d is %f — corrupted by compression", typeName, j, v)
					break
				}
			}
		}
	}
}

func TestMixedIndexTypesInterleavedOperations(t *testing.T) {
	t.Parallel()
	// Single goroutine interleaves operations across all three index types:
	// upsert to flat, query PQ, delete from SQ, query flat, verify SQ deletes.
	// Catches: client maintaining hidden "current index type" state that
	// causes wrong-type dispatch when rapidly switching between types.
	m := newMixedIndexSet(t)
	ctx, cancel := context.WithTimeout(context.Background(), concTimeout)
	defer cancel()

	// Step 1: Seed all three indexes with data
	flatIDs := make([]string, 20)
	flatVecs := generateRandomVectors(20, concDimension)
	for i := 0; i < 20; i++ {
		flatIDs[i] = fmt.Sprintf("il_flat_%d", i)
	}
	if err := m.flat.UpsertVectors(ctx, flatIDs, flatVecs, nil); err != nil {
		t.Fatalf("Flat upsert failed: %v", err)
	}

	pqIDs := make([]string, 20)
	pqVecs := generateRandomVectors(20, concDimension)
	for i := 0; i < 20; i++ {
		pqIDs[i] = fmt.Sprintf("il_pq_%d", i)
	}
	if err := m.pq.UpsertVectors(ctx, pqIDs, pqVecs, nil); err != nil {
		t.Fatalf("PQ upsert failed: %v", err)
	}

	sqIDs := make([]string, 20)
	sqVecs := generateRandomVectors(20, concDimension)
	for i := 0; i < 20; i++ {
		sqIDs[i] = fmt.Sprintf("il_sq_%d", i)
	}
	if err := m.sq.UpsertVectors(ctx, sqIDs, sqVecs, nil); err != nil {
		t.Fatalf("SQ upsert failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Step 2: Interleaved operations — rapidly switch between index types

	// Query PQ
	pqResults, err := m.pq.Query(ctx, cyborgdb.QueryParams{
		QueryVector: pqVecs[0], TopK: 5,
	})
	if err != nil {
		t.Fatalf("PQ query failed after interleave: %v", err)
	}
	pqItems := getQueryResultItems(&pqResults.Results)
	for _, item := range pqItems {
		if strings.HasPrefix(item.GetId(), "il_flat_") || strings.HasPrefix(item.GetId(), "il_sq_") {
			t.Errorf("PQ query returned non-PQ ID '%s' — cross-type dispatch bug", item.GetId())
		}
	}

	// Upsert more to flat (while PQ was just queried)
	extraFlatIDs := make([]string, 5)
	extraFlatVecs := generateRandomVectors(5, concDimension)
	for i := 0; i < 5; i++ {
		extraFlatIDs[i] = fmt.Sprintf("il_flat_extra_%d", i)
	}
	if err := m.flat.UpsertVectors(ctx, extraFlatIDs, extraFlatVecs, nil); err != nil {
		t.Fatalf("Flat extra upsert failed: %v", err)
	}

	// Delete first 10 from SQ (while flat was just upserted to)
	if err := m.sq.Delete(ctx, sqIDs[:10]); err != nil {
		t.Fatalf("SQ delete failed: %v", err)
	}

	// Query flat (while SQ was just deleted from)
	flatResults, err := m.flat.Query(ctx, cyborgdb.QueryParams{
		QueryVector: flatVecs[0], TopK: 5,
	})
	if err != nil {
		t.Fatalf("Flat query failed after interleave: %v", err)
	}
	flatItems := getQueryResultItems(&flatResults.Results)
	for _, item := range flatItems {
		if strings.HasPrefix(item.GetId(), "il_pq_") || strings.HasPrefix(item.GetId(), "il_sq_") {
			t.Errorf("Flat query returned non-flat ID '%s' — cross-type dispatch bug", item.GetId())
		}
	}

	time.Sleep(2 * time.Second)

	// Step 3: Verify SQ deletes actually took effect (not applied to flat or PQ)
	sqListResp, err := m.sq.ListIDs(ctx)
	if err != nil {
		t.Fatalf("SQ ListIDs failed: %v", err)
	}
	sqSurviving := make(map[string]bool, len(sqListResp.Ids))
	for _, id := range sqListResp.Ids {
		sqSurviving[id] = true
	}

	// First 10 should be gone
	for _, deleted := range sqIDs[:10] {
		if sqSurviving[deleted] {
			t.Errorf("SQ delete didn't take effect: '%s' still present", deleted)
		}
	}
	// Last 10 should survive
	for _, kept := range sqIDs[10:] {
		if !sqSurviving[kept] {
			t.Errorf("SQ lost ID '%s' that wasn't deleted — collateral damage from interleaved ops", kept)
		}
	}

	// Flat should still have all original + extra IDs (delete on SQ must not affect flat)
	flatListResp, err := m.flat.ListIDs(ctx)
	if err != nil {
		t.Fatalf("Flat ListIDs failed: %v", err)
	}
	if len(flatListResp.Ids) != 25 { // 20 original + 5 extra
		t.Errorf("Flat index has %d IDs, expected 25 — SQ delete leaked to flat", len(flatListResp.Ids))
	}

	// PQ should still have all 20 IDs (untouched by flat upserts and SQ deletes)
	pqListResp, err := m.pq.ListIDs(ctx)
	if err != nil {
		t.Fatalf("PQ ListIDs failed: %v", err)
	}
	if len(pqListResp.Ids) != 20 {
		t.Errorf("PQ index has %d IDs, expected 20 — operations on other types leaked to PQ", len(pqListResp.Ids))
	}
}
