// TurboQuant storage precision: the `storage_precision` create-time knob and
// its four quantized tiers `tq12` / `tq8` / `tq6` / `tq4`.
//
// `storage_precision` picks the on-disk rerank-vector format, chosen at create
// and immutable. Alongside the existing `float32` / `float16`, the TurboQuant
// tiers pack 12 / 8 / 6 / 4 bits per dimension, trading a little recall and
// latency for a large storage saving. All tiers work with every metric.
//
// Two layers of coverage:
//
//   - Model-level (no service) — the exported StoragePrecision* constants have
//     the expected wire values and the generated CreateIndexRequest carries
//     each tier through its accessors and JSON serialization. These are the
//     direct, deterministic checks that the tiers were wired in.
//   - End-to-end (live service) — each tier survives the full create -> upsert
//     -> train -> query round-trip and returns sane, high self-recall results.
//     Skipped automatically when no service is reachable.
//
// The index-info response does not echo `storage_precision` back, so the
// end-to-end layer verifies the tiers by behavior, not by reading the value
// back off the index.

package test

import (
	"context"
	"encoding/json"
	"math"
	mrand "math/rand"
	"strconv"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// The four quantized tiers this feature adds.
var turboQuantTiers = []string{
	cyborgdb.StoragePrecisionTQ12,
	cyborgdb.StoragePrecisionTQ8,
	cyborgdb.StoragePrecisionTQ6,
	cyborgdb.StoragePrecisionTQ4,
}

// Every valid storage precision, including the pre-existing float tiers.
var validPrecisions = []string{
	cyborgdb.StoragePrecisionFloat32,
	cyborgdb.StoragePrecisionFloat16,
	cyborgdb.StoragePrecisionTQ12,
	cyborgdb.StoragePrecisionTQ8,
	cyborgdb.StoragePrecisionTQ6,
	cyborgdb.StoragePrecisionTQ4,
}

// TurboQuant corpus sizing. Enough vectors to clear the core training floor
// (train() silently no-ops below 10k vectors) while staying quick.
const (
	tqNumVectors = 10000
	tqDim        = 64
	tqNLists     = 8
)

// TestTurboQuantModel is the model-level contract for `storage_precision` — no
// service required.
func TestTurboQuantModel(t *testing.T) {
	t.Run("ConstantsHaveExpectedWireValues", func(t *testing.T) {
		cases := map[string]string{
			cyborgdb.StoragePrecisionFloat32: "float32",
			cyborgdb.StoragePrecisionFloat16: "float16",
			cyborgdb.StoragePrecisionTQ12:    "tq12",
			cyborgdb.StoragePrecisionTQ8:     "tq8",
			cyborgdb.StoragePrecisionTQ6:     "tq6",
			cyborgdb.StoragePrecisionTQ4:     "tq4",
		}
		for got, want := range cases {
			if got != want {
				t.Errorf("storage precision constant = %q, want %q", got, want)
			}
		}
	})

	t.Run("TurboQuantTiersAccepted", func(t *testing.T) {
		// The three tiers this change adds, carried through the request accessors.
		for _, tier := range turboQuantTiers {
			tier := tier
			t.Run(tier, func(t *testing.T) {
				req := internal.NewCreateIndexRequest("idx")
				req.SetStoragePrecision(tier)
				if !req.HasStoragePrecision() {
					t.Fatalf("HasStoragePrecision = false after setting %q", tier)
				}
				if got := req.GetStoragePrecision(); got != tier {
					t.Errorf("GetStoragePrecision = %q, want %q", got, tier)
				}
			})
		}
	})

	t.Run("StoragePrecisionOptional", func(t *testing.T) {
		req := internal.NewCreateIndexRequest("idx")
		if req.HasStoragePrecision() {
			t.Error("HasStoragePrecision = true on a fresh request, want false")
		}
		// An unset precision must not appear in the serialized payload.
		payload, err := req.ToMap()
		if err != nil {
			t.Fatalf("ToMap failed: %v", err)
		}
		if _, present := payload["storage_precision"]; present {
			t.Error("storage_precision serialized despite never being set")
		}
	})

	t.Run("PrecisionSerializedToWireDict", func(t *testing.T) {
		for _, tier := range validPrecisions {
			tier := tier
			t.Run(tier, func(t *testing.T) {
				req := internal.NewCreateIndexRequest("idx")
				req.SetStoragePrecision(tier)
				payload, err := req.ToMap()
				if err != nil {
					t.Fatalf("ToMap failed: %v", err)
				}
				got, ok := payload["storage_precision"].(*string)
				if !ok || got == nil {
					t.Fatalf("storage_precision missing or wrong type in wire dict: %#v", payload["storage_precision"])
				}
				if *got != tier {
					t.Errorf("wire storage_precision = %q, want %q", *got, tier)
				}
			})
		}
	})

	t.Run("PrecisionRoundTripsThroughJSON", func(t *testing.T) {
		for _, tier := range turboQuantTiers {
			tier := tier
			t.Run(tier, func(t *testing.T) {
				req := internal.NewCreateIndexRequest("idx")
				req.SetStoragePrecision(tier)

				data, err := json.Marshal(req)
				if err != nil {
					t.Fatalf("Marshal failed: %v", err)
				}

				var restored internal.CreateIndexRequest
				if err := json.Unmarshal(data, &restored); err != nil {
					t.Fatalf("Unmarshal failed: %v", err)
				}
				if got := restored.GetStoragePrecision(); got != tier {
					t.Errorf("round-tripped storage_precision = %q, want %q", got, tier)
				}
			})
		}
	})
}

// TestTurboQuantIntegration exercises each TurboQuant tier end-to-end. Skipped
// when no CyborgDB service is reachable.
//
// One shared, cosine-metric corpus is built once (cosine is required by `tq4`
// and valid for every other tier). Each tier gets its own index so a failure
// names the tier that broke.
func TestTurboQuantIntegration(t *testing.T) {
	client := newIsolatedClient(t)

	healthCtx, healthCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer healthCancel()
	if _, err := client.GetHealth(healthCtx); err != nil {
		t.Skipf("no CyborgDB service reachable at %s: %v", testBaseURL(), err)
	}

	// Build the shared normalized corpus once. Normalizing for the cosine
	// metric makes self-queries unambiguous.
	vectors := normalizedRandomVectors(tqNumVectors, tqDim, 42)
	ids := make([]string, tqNumVectors)
	for i := range ids {
		ids[i] = strconv.Itoa(i)
	}

	buildTrainedIndex := func(t *testing.T, precision string) *cyborgdb.EncryptedIndex {
		t.Helper()
		key := generateRandomKey()
		metric := "cosine"
		dim := int32(tqDim)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
			IndexName:        generateUniqueName("tq_" + precision + "_"),
			IndexKey:         key,
			Dimension:        &dim,
			Metric:           &metric,
			StoragePrecision: &precision,
		})
		if err != nil {
			t.Fatalf("create %s index: %v", precision, err)
		}
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cleanupCancel()
			_ = index.DeleteIndex(cleanupCtx)
		})

		if err := index.UpsertVectors(ctx, ids, vectors, nil); err != nil {
			t.Fatalf("upsert into %s index: %v", precision, err)
		}
		time.Sleep(1 * time.Second)

		listed, err := index.ListIDs(ctx)
		if err != nil {
			t.Fatalf("list IDs on %s index: %v", precision, err)
		}
		if len(listed.Ids) != tqNumVectors {
			t.Fatalf("%s index: listed %d IDs, want %d", precision, len(listed.Ids), tqNumVectors)
		}

		nLists := int32(tqNLists)
		if err := index.Train(ctx, cyborgdb.TrainParams{NLists: &nLists}); err != nil {
			t.Fatalf("train %s index: %v", precision, err)
		}

		trained := pollUntil(120*time.Second, func() bool {
			training, err := index.IsTraining(ctx)
			if err != nil || training {
				return false
			}
			done, err := index.IsTrained(ctx)
			return err == nil && done
		})
		if !trained {
			t.Fatalf("%s index failed to train", precision)
		}
		return index
	}

	// assertSelfRecall queries with vectors that are in the index; each should
	// find itself. Exhaustive search (n_probes == n_lists) removes IVF
	// partitioning as a variable, so the only recall loss left is TurboQuant's
	// quantization — which the threshold tolerates.
	assertSelfRecall := func(t *testing.T, index *cyborgdb.EncryptedIndex, precision string, minRecall float64) {
		t.Helper()
		const numProbe = 50
		nProbes := int32(tqNLists)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		results, err := index.Query(ctx, cyborgdb.QueryParams{
			BatchQueryVectors: vectors[:numProbe],
			TopK:              10,
			NProbes:           &nProbes,
		})
		if err != nil {
			t.Fatalf("query %s index: %v", precision, err)
		}
		if results == nil {
			t.Fatalf("query on %s index returned nil results", precision)
		}

		batch := getBatchQueryResults(&results.Results)
		if len(batch) != numProbe {
			t.Fatalf("%s index: got %d result sets, want %d", precision, len(batch), numProbe)
		}

		hits := 0
		for localID, hitList := range batch {
			want := strconv.Itoa(localID)
			for _, res := range hitList {
				if res.Id == want {
					hits++
					break
				}
			}
		}
		recall := float64(hits) / float64(numProbe)
		if recall < minRecall {
			t.Errorf("%s: self-recall %.2f below %.2f", precision, recall, minRecall)
		}
	}

	t.Run("TQ12Lifecycle", func(t *testing.T) {
		// tq12 is the least aggressive tier, so it should have the highest recall.
		index := buildTrainedIndex(t, cyborgdb.StoragePrecisionTQ12)
		assertSelfRecall(t, index, cyborgdb.StoragePrecisionTQ12, 0.9)
	})

	t.Run("TQ8Lifecycle", func(t *testing.T) {
		index := buildTrainedIndex(t, cyborgdb.StoragePrecisionTQ8)
		assertSelfRecall(t, index, cyborgdb.StoragePrecisionTQ8, 0.9)
	})

	t.Run("TQ6Lifecycle", func(t *testing.T) {
		index := buildTrainedIndex(t, cyborgdb.StoragePrecisionTQ6)
		assertSelfRecall(t, index, cyborgdb.StoragePrecisionTQ6, 0.85)
	})

	t.Run("TQ4Lifecycle", func(t *testing.T) {
		// tq4 is the most aggressive tier and is only valid with cosine.
		index := buildTrainedIndex(t, cyborgdb.StoragePrecisionTQ4)
		assertSelfRecall(t, index, cyborgdb.StoragePrecisionTQ4, 0.7)
	})

	t.Run("TQ4RequiresCosineMetric", func(t *testing.T) {
		// tq4 with a non-cosine metric must be rejected by the service.
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		key := generateRandomKey()
		metric := "euclidean"
		dim := int32(tqDim)
		precision := cyborgdb.StoragePrecisionTQ4

		index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
			IndexName:        generateUniqueName("tq4_bad_"),
			IndexKey:         key,
			Dimension:        &dim,
			Metric:           &metric,
			StoragePrecision: &precision,
		})
		if err == nil {
			_ = index.DeleteIndex(ctx)
			t.Fatal("expected error creating tq4 index with euclidean metric, got nil")
		}
	})
}

// normalizedRandomVectors returns count unit-length float32 vectors of the
// given dimension, seeded for determinism.
func normalizedRandomVectors(count, dim, seed int) [][]float32 {
	rng := mrand.New(mrand.NewSource(int64(seed)))
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vec := make([]float32, dim)
		var sumSq float64
		for j := 0; j < dim; j++ {
			v := rng.Float32()
			vec[j] = v
			sumSq += float64(v) * float64(v)
		}
		norm := math.Sqrt(sumSq)
		if norm < 1e-12 {
			norm = 1e-12
		}
		for j := 0; j < dim; j++ {
			vec[j] = float32(float64(vec[j]) / norm)
		}
		vectors[i] = vec
	}
	return vectors
}
