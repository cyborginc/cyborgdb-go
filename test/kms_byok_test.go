// KMS BYOK integration tests for the CyborgDB Go SDK.
//
// These tests are gated on environment variables that name entries in the
// running cyborgdb-service's kms.registry. Set the variable to opt the
// corresponding registry slot in; leave it unset to skip.
//
//   - CYBORGDB_KMS_NAME_REAL — real-provider entry with provider: aws-kms
//     (HSM-resident KEK; service generates the DEK and asks the HSM to wrap it).
//   - CYBORGDB_KMS_NAME_SM   — real-provider entry with provider: aws
//     (Secrets Manager-resident KEK; service generates the DEK and AES-GCM-
//     wraps it locally under the SM-fetched key).
//   - CYBORGDB_KMS_NAME_NONE — entry with provider: none. The SDK supplies
//     the KEK on every request; service does no KMS round-trips.
//
// All three exercise the SDK round-trip introduced when CreateIndex and
// LoadIndex moved to optional IndexKey + KmsName routing.

package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

const (
	kmsDimension       = 128
	kmsNumVectors      = 10
	kmsTimeout         = 60 * time.Second
	kmsCleanupTimeout  = 30 * time.Second
	kmsExpectedType    = "disk_ivf"
	kmsEuclideanMetric = "euclidean"
)

// kmsBYOKConfig captures the env-driven configuration for one BYOK round-trip.
type kmsBYOKConfig struct {
	envVar      string // env var that gates the test
	label       string // sub-test label
	needsSDKKey bool   // provider: none variant — SDK supplies the KEK
}

// kmsBYOKConfigs enumerates every supported KMS posture. Add new providers
// here and they will be picked up automatically.
var kmsBYOKConfigs = []kmsBYOKConfig{
	{envVar: "CYBORGDB_KMS_NAME_REAL", label: "aws-kms", needsSDKKey: false},
	{envVar: "CYBORGDB_KMS_NAME_SM", label: "aws", needsSDKKey: false},
	{envVar: "CYBORGDB_KMS_NAME_NONE", label: "none", needsSDKKey: true},
}

// makeKMSVectors deterministically generates dimension-sized vectors. Stable
// values let self-match assertions in the data-plane checks rely on id=0
// being its own nearest neighbor.
func makeKMSVectors(n, dim int) (cyborgdb.VectorItems, [][]float32) {
	items := make(cyborgdb.VectorItems, n)
	vectors := make([][]float32, n)
	for i := 0; i < n; i++ {
		vec := make([]float32, dim)
		for j := 0; j < dim; j++ {
			vec[j] = float32((i*dim+j)%1009) / 1009.0
		}
		vectors[i] = vec
		items[i] = cyborgdb.VectorItem{
			Id:       fmt.Sprintf("%d", i),
			Vector:   vec,
			Metadata: map[string]interface{}{"idx": i},
		}
	}
	return items, vectors
}

// runKMSRoundTrip exercises create + load + full data plane against one
// kms.registry slot. Real-KMS variants exercise the data-plane *without* an
// SDK-held key — the unique regression risk of the new keyless path.
func runKMSRoundTrip(t *testing.T, cfg kmsBYOKConfig, kmsName string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), kmsTimeout)
	defer cancel()

	client, err := cyborgdb.NewClient(testBaseURL(), testAPIKey())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	indexName := fmt.Sprintf("test_kms_%s_%d", cfg.label, time.Now().UnixNano())

	var indexKey []byte
	if cfg.needsSDKKey {
		indexKey, err = cyborgdb.GenerateKey()
		if err != nil {
			t.Fatalf("GenerateKey: %v", err)
		}
	}

	// --- create ---
	dim := int32(kmsDimension)
	metric := kmsEuclideanMetric
	params := &cyborgdb.CreateIndexParams{
		IndexName: indexName,
		KmsName:   &kmsName,
		Dimension: &dim,
		Metric:    &metric,
	}
	if cfg.needsSDKKey {
		params.IndexKey = indexKey
	}

	index, err := client.CreateIndex(ctx, params)
	if err != nil {
		t.Fatalf("CreateIndex (kms=%s): %v", kmsName, err)
	}
	t.Cleanup(func() {
		// Use a fresh context: the test-scoped ctx may already be cancelled
		// (e.g., if the test consumed its full timeout), which would
		// short-circuit cleanup before the request hits the server.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), kmsCleanupTimeout)
		defer cancel()
		_ = index.DeleteIndex(cleanupCtx)
	})
	if index.GetIndexName() != indexName {
		t.Errorf("GetIndexName: got %q, want %q", index.GetIndexName(), indexName)
	}

	// --- load (keyless for real-KMS, keyed for provider:none) ---
	var loaded *cyborgdb.EncryptedIndex
	if cfg.needsSDKKey {
		loaded, err = client.LoadIndex(ctx, indexName, indexKey)
	} else {
		loaded, err = client.LoadIndex(ctx, indexName, nil)
	}
	if err != nil {
		t.Fatalf("LoadIndex (kms=%s, needsSDKKey=%v): %v", kmsName, cfg.needsSDKKey, err)
	}
	if loaded.GetIndexType() != kmsExpectedType {
		t.Errorf("GetIndexType: got %q, want %q", loaded.GetIndexType(), kmsExpectedType)
	}

	// --- data plane round-trip via the LoadIndex-returned handle ---
	// We deliberately exercise `loaded` (not the `index` returned by
	// CreateIndex) so a regression where LoadIndex's keyless path returns
	// a handle with a broken DEK lookup surfaces here. For real-KMS
	// variants, `loaded` has no SDK-held key, which is the unique
	// regression risk of the new keyless path.
	items, vectors := makeKMSVectors(kmsNumVectors, kmsDimension)
	if err := loaded.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	queryResp, err := loaded.Query(ctx, cyborgdb.QueryParams{
		QueryVector: vectors[0],
		TopK:        3,
		Include:     []string{"distance", "metadata"},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	results := queryResp.GetResults()
	if results.ArrayOfQueryResultItem == nil {
		t.Fatalf("Query: expected single-query results, got %+v", results)
	}
	hits := *results.ArrayOfQueryResultItem
	if len(hits) < 3 {
		t.Fatalf("Query: expected at least 3 results, got %d", len(hits))
	}
	if hits[0].Id != "0" {
		t.Errorf("Query: closest match should be self (id=0), got %q", hits[0].Id)
	}

	getResp, err := loaded.Get(ctx, []string{"0"}, []string{"metadata"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(getResp.Results) != 1 || getResp.Results[0].Id != "0" {
		t.Errorf("Get: unexpected result %+v", getResp.Results)
	}

	listResp, err := loaded.ListIDs(ctx)
	if err != nil {
		t.Fatalf("ListIDs: %v", err)
	}
	if len(listResp.Ids) < kmsNumVectors {
		t.Errorf("ListIDs: got %d ids, want at least %d", len(listResp.Ids), kmsNumVectors)
	}

	if _, err := loaded.IsTrained(ctx); err != nil {
		t.Errorf("IsTrained: %v", err)
	}
	if _, err := loaded.CheckTrainingStatus(ctx); err != nil {
		t.Errorf("CheckTrainingStatus: %v", err)
	}

	if err := loaded.Delete(ctx, []string{"0"}); err != nil {
		t.Errorf("Delete: %v", err)
	}
}

// TestKMSBYOK iterates every configured KMS posture variant. Each variant
// skips cleanly when its gating env var isn't set, so the suite is safe to
// run with zero, one, or all slots configured.
func TestKMSBYOK(t *testing.T) {
	if testAPIKey() == "" {
		t.Skip("CYBORGDB_API_KEY not set — skipping live BYOK round-trips.")
	}
	for _, cfg := range kmsBYOKConfigs {
		cfg := cfg
		t.Run(cfg.label, func(t *testing.T) {
			kmsName := os.Getenv(cfg.envVar)
			if kmsName == "" {
				t.Skipf("%s not set — skipping %s round-trip.", cfg.envVar, cfg.label)
			}
			runKMSRoundTrip(t, cfg, kmsName)
		})
	}
}

// TestKMSRealRejectsSDKKey checks the server-side contract for a real-provider
// slot: the service generates the KEK itself, so supplying IndexKey alongside
// KmsName is contradictory and rejected with a 400. The SDK forwards both
// fields untouched — the rejection is the server's call, not the client's.
// (provider:none, where both fields ARE valid, is covered by the "none"
// round-trip in TestKMSBYOK.)
func TestKMSRealRejectsSDKKey(t *testing.T) {
	if testAPIKey() == "" {
		t.Skip("CYBORGDB_API_KEY not set — skipping live BYOK negative test.")
	}
	kmsName := os.Getenv("CYBORGDB_KMS_NAME_REAL")
	if kmsName == "" {
		t.Skip("CYBORGDB_KMS_NAME_REAL not set — skipping real-provider negative test.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), kmsTimeout)
	defer cancel()

	client, err := cyborgdb.NewClient(testBaseURL(), testAPIKey())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	indexKey, err := cyborgdb.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	indexName := fmt.Sprintf("test_kms_neg_%d", time.Now().UnixNano())
	dim := int32(kmsDimension)
	metric := kmsEuclideanMetric

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName: indexName,
		IndexKey:  indexKey,
		KmsName:   &kmsName,
		Dimension: &dim,
		Metric:    &metric,
	})
	if err == nil {
		// Unexpected success — clean up so we don't leak the index.
		cleanupCtx, ccancel := context.WithTimeout(context.Background(), kmsCleanupTimeout)
		defer ccancel()
		_ = index.DeleteIndex(cleanupCtx)
		t.Fatalf("CreateIndex with real-provider kms=%s and IndexKey: expected a rejection, got none", kmsName)
	}
}
