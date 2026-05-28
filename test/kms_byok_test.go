// KMS BYOK integration tests for the CyborgDB Go SDK.
//
// The service supports two wire encodings for index encryption keys, and
// kms_name + index_key are strictly mutually exclusive on the create
// request (the server returns 400 regardless of which slot kms_name
// resolves to):
//
//  1. SDK-supplied KEK — IndexKey alone, no KmsName. Service records
//     the envelope as provider="none" and the SDK re-supplies the same
//     key on every subsequent request. No KMS registry slot is
//     referenced on the wire.
//  2. KMS-managed KEK — KmsName alone, no IndexKey. Service generates a
//     random KEK and wraps it via the named registry slot's provider.
//
// KMS-managed variants are gated on the registry-slot envs because they
// require a configured kms.registry entry in the running service:
//
//   - CYBORGDB_KMS_NAME_REAL — entry with provider: aws-kms (HSM-resident
//     wrap key; service asks the HSM to wrap the per-index KEK).
//   - CYBORGDB_KMS_NAME_SM   — entry with provider: aws (Secrets Manager-
//     resident wrap key; service AES-GCM-wraps locally under the SM
//     value).
//
// The SDK-supplied variant needs no registry slot and is exercised live
// whenever CYBORGDB_API_KEY is set; it used to be gated on a
// provider: none slot that has since been removed from the registry —
// strict mutex made that slot unreachable from the SDK anyway.

package test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	// envVar names the env var that gates the variant in/out. Empty for
	// the SDK-supplied variant, which references no registry slot and is
	// gated on CYBORGDB_API_KEY at the suite level.
	envVar      string
	label       string // sub-test label
	needsSDKKey bool   // SDK-supplied variant — IndexKey alone, no KmsName
}

// kmsBYOKConfigs enumerates every supported KMS posture. Add new providers
// here and they will be picked up automatically.
var kmsBYOKConfigs = []kmsBYOKConfig{
	{envVar: "CYBORGDB_KMS_NAME_REAL", label: "aws-kms", needsSDKKey: false},
	{envVar: "CYBORGDB_KMS_NAME_SM", label: "aws", needsSDKKey: false},
	// SDK-supplied KEK: IndexKey alone, no KmsName, no registry slot.
	// Gated on CYBORGDB_API_KEY only (the suite-level gate).
	{envVar: "", label: "sdk-supplied", needsSDKKey: true},
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
	// Send exactly one of IndexKey / KmsName. Supplying both is rejected
	// by the service with a 400 for every provider — see
	// TestKMSRejectsBothFields below for the contract enforcement.
	dim := int32(kmsDimension)
	metric := kmsEuclideanMetric
	params := &cyborgdb.CreateIndexParams{
		IndexName: indexName,
		Dimension: &dim,
		Metric:    &metric,
	}
	if cfg.needsSDKKey {
		params.IndexKey = indexKey
	} else {
		params.KmsName = &kmsName
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

	// --- load (keyless for KMS-managed, keyed for SDK-supplied) ---
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

// TestKMSBYOK iterates every configured KMS posture variant. KMS-managed
// variants skip cleanly when their gating env var isn't set; the
// SDK-supplied variant has no env-var gate and runs whenever
// CYBORGDB_API_KEY is set. Safe to run with zero, one, or all KMS slots
// configured.
func TestKMSBYOK(t *testing.T) {
	if testAPIKey() == "" {
		t.Skip("CYBORGDB_API_KEY not set — skipping live BYOK round-trips.")
	}
	for _, cfg := range kmsBYOKConfigs {
		cfg := cfg
		t.Run(cfg.label, func(t *testing.T) {
			var kmsName string
			if cfg.envVar != "" {
				kmsName = os.Getenv(cfg.envVar)
				if kmsName == "" {
					t.Skipf("%s not set — skipping %s round-trip.", cfg.envVar, cfg.label)
				}
			}
			runKMSRoundTrip(t, cfg, kmsName)
		})
	}
}

// TestKMSRejectsBothFields checks the server-side mutex contract:
// supplying IndexKey alongside KmsName is contradictory and the service
// rejects it with a 400 for *every* provider, including the
// SDK-supplied path. The SDK forwards both fields untouched, so the
// rejection is the server's call, not the client's.
//
// Routes through the SDK helper (vs. raw HTTP) because the SDK already
// surfaces the error reliably as a non-nil err — content inspection is
// deferred to TestKMSStrictMutexFiresBeforeSlotLookup.
func TestKMSRejectsBothFields(t *testing.T) {
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

// TestKMSStrictMutexFiresBeforeSlotLookup pins down the ordering of the
// server's two checks: the IndexKey + KmsName mutex check fires BEFORE
// the registry slot lookup, so an unknown slot combined with an
// IndexKey returns the *mutex* 400 (not an "unknown slot" 400). A
// future server refactor that quietly swaps the ordering would let the
// combination through for an as-yet-unknown slot — this test guards
// against that drift.
//
// Hits the endpoint directly via net/http — bypassing the SDK helper
// so we can inspect the server's `detail` field, which the SDK's
// GenericOpenAPIError.Error() doesn't surface today.
func TestKMSStrictMutexFiresBeforeSlotLookup(t *testing.T) {
	if testAPIKey() == "" {
		t.Skip("CYBORGDB_API_KEY not set — skipping strict-mutex coverage.")
	}

	indexKey, err := cyborgdb.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	payload, err := json.Marshal(map[string]interface{}{
		"index_name": fmt.Sprintf("test_kms_mutex_%d", time.Now().UnixNano()),
		"index_key":  hex.EncodeToString(indexKey),
		"kms_name":   "definitely-not-a-registered-slot",
		"dimension":  kmsDimension,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost,
		testBaseURL()+"/v1/indexes/create",
		bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var parsed struct {
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v (body=%q)", err, body)
	}
	if !strings.Contains(parsed.Detail, "index_key must not be supplied alongside") {
		t.Errorf("detail does not match mutex message: %q", parsed.Detail)
	}
}
