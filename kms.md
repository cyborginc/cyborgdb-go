# KMS + Multi-Tenancy Re-Implementation Guide (PR #76)

A step-by-step guide for re-applying the KMS / multi-tenancy work on a fresh
branch off an updated `origin/main`. Companion to `go-pr.md` (which is the PR
description prose) — this file is the *how to redo it* checklist.

The reference implementation lives on the local `multi-tenancy` branch at
commit `f59bacb` (pre-rebase HEAD). Merge-base with `origin/main` at the time
of the original work was `b905a99`. All diffs below are
`b905a99..f59bacb`.

---

## 1. What this PR does, in one paragraph

Track `cyborgdb-service`'s `multi-tenancy` branch — specifically the Phase 1
per-index KMS routing slice. Regenerate the OpenAPI client against the new
`CreateIndexRequest` shape (flat fields, no more `index_config` union), make
`IndexKey` optional, add `KmsName` / `Dimension` / `StoragePrecision` to
`CreateIndexParams`, and add offline + live BYOK tests. Mirrors the Python SDK
changes one-for-one (see `cyborgdb-py`'s analogous PR for the Python form).

Underlying service behavior: `CreateIndexRequest` was restructured so
`index_config` / `IndexIVF*Model` are gone in favor of flat
`dimension` / `metric` / `storage_precision` fields, `index_key` is now
optional, and a new `kms_name` references entries in `kms.registry`.
`IndexOperationRequest` similarly made `index_key` optional so KMS-backed
indexes can be loaded without an SDK-held key.

---

## 2. Source-of-truth spec

- Copy `openapi.json` verbatim from `cyborgdb-py` (multi-tenancy HEAD).
  `info.version` should read **`0.16.1`**.
- Both repos share identical spec bytes — no need to re-run the FastAPI
  export. If `cyborgdb-py`'s spec has moved on, regenerate from there first
  rather than re-exporting from the service.

Sanity-check the regenerated spec:

```bash
python3 -c "
import json
d = json.load(open('openapi.json'))
print('version:', d['info']['version'])  # expect 0.16.1
print('CreateIndexRequest keys:', list(d['components']['schemas']['CreateIndexRequest']['properties'].keys()))
# expect: ['index_name', 'kms_name', 'index_key', 'dimension', 'embedding_model', 'metric', 'storage_precision']
print('CreateIndexRequest required:', d['components']['schemas']['CreateIndexRequest'].get('required'))
# expect: ['index_name']
print('IndexOperationRequest required:', d['components']['schemas']['IndexOperationRequest'].get('required'))
# expect: ['index_name']
"
```

---

## 3. Regenerate `internal/`

Update `update-openapi-client.sh` to use the npm-hosted CLI (the brew package
lags behind):

```sh
# replace the "openapi-generator" check + invocation with:
if ! command -v openapi-generator-cli &> /dev/null; then
    echo "Installing openapi-generator-cli..."
    npm install -g @openapitools/openapi-generator-cli
fi

# ... later in the script ...
openapi-generator-cli generate \
    -i openapi.json \
    -g go \
    -o ./internal \
    ...
```

Add `openapitools.json` at the repo root (pins the generator version):

```json
{
  "$schema": "./node_modules/@openapitools/openapi-generator-cli/config.schema.json",
  "spaces": 2,
  "generator-cli": {
    "version": "7.22.0"
  }
}
```

Then `./update-openapi-client.sh`. Confirm these are **deleted** afterwards
(the spec no longer references them):

- `internal/model_index_config.go`
- `internal/model_index_ivf_model.go`
- `internal/model_index_ivf_flat_model.go`
- `internal/model_index_ivfpq_model.go`
- `internal/model_index_ivfsq_model.go`

Confirm these are **renamed**:

- `internal/model_validation_error_loc_inner.go` → `internal/model_location_inner.go`

Existing fix-ups in `update-openapi-client.sh` are still needed and unchanged
(Contents anyOf patch, `GetResultItemModel` `*os.File`→`string` patch,
`DisallowUnknownFields` removal). Don't touch those.

---

## 4. Hand-written code changes

The rest of these are outside `internal/`. All changes are mechanical given
the new spec shape.

### 4.1 `client.go`

Add a new sentinel error:

```go
// ErrMissingKeyOrKMS is returned when CreateIndex is called with neither
// IndexKey nor KmsName set.
ErrMissingKeyOrKMS = fmt.Errorf("create_index requires IndexKey, KmsName, or both")
```

**`Client.CreateIndex`** — three behavior changes:

1. Validate `IndexKey` only when non-empty; require at least one of
   `IndexKey` / `KmsName`:

   ```go
   if len(params.IndexKey) == 0 && params.KmsName == nil {
       return nil, ErrMissingKeyOrKMS
   }

   var keyHex *string
   if len(params.IndexKey) > 0 {
       if len(params.IndexKey) != KeySize {
           return nil, fmt.Errorf("%w, got %d", ErrInvalidKeyLength, len(params.IndexKey))
       }
       hex := fmt.Sprintf("%x", params.IndexKey)
       keyHex = &hex
   }
   ```

2. Build the request with `NullableString` wrappers, omitting fields whose
   source is nil:

   ```go
   req := internal.CreateIndexRequest{IndexName: params.IndexName}
   if keyHex != nil {
       req.IndexKey = *internal.NewNullableString(keyHex)
   }
   if params.KmsName != nil {
       req.KmsName = *internal.NewNullableString(params.KmsName)
   }
   if params.Dimension != nil {
       req.Dimension = *internal.NewNullableInt32(params.Dimension)
   }
   if params.Metric != nil {
       req.Metric = *internal.NewNullableString(params.Metric)
   }
   if params.EmbeddingModel != nil {
       req.EmbeddingModel = *internal.NewNullableString(params.EmbeddingModel)
   }
   if params.StoragePrecision != nil {
       req.StoragePrecision = *internal.NewNullableString(params.StoragePrecision)
   }
   ```

3. Drop all the `IndexConfig` / `IndexIVF*Model` handling in the
   `EncryptedIndex` builder — there's no `config` field anymore (see 4.2),
   and there's no `indexType` switch on three IVF variants.

**`Client.LoadIndex`** — make `indexKey []byte` optional:

```go
var keyHex *string
if len(indexKey) > 0 {
    if len(indexKey) != KeySize {
        return nil, fmt.Errorf("%w, got %d", ErrInvalidKeyLength, len(indexKey))
    }
    hex := fmt.Sprintf("%x", indexKey)
    keyHex = &hex
}

describeReq := internal.IndexOperationRequest{IndexName: indexName}
if keyHex != nil {
    describeReq.IndexKey = *internal.NewNullableString(keyHex)
}
```

Return the `EncryptedIndex` without a `config` field; `indexType` comes from
`indexInfo.IndexType`.

### 4.2 `encrypted_index.go`

**Struct shape** — `indexKey` becomes `*string`, drop `config`:

```go
type EncryptedIndex struct {
    indexName string
    // indexKey is nil for KMS-backed indexes; the service resolves the DEK
    // from the stored KMSBlob.
    indexKey  *string
    indexType string
    trained   bool
    client    *internal.Client
    // ...
}
```

**Drop `GetIndexConfig()`** entirely.

**Add private `indexKeyField()` helper** — every data-plane request builder
calls this instead of poking `e.indexKey` directly:

```go
// indexKeyField builds the NullableString IndexKey field expected by the
// generated request models. Returns the unset zero value when this index has
// no SDK-held key (KMS-backed indexes), which serializes as "field omitted"
// and lets the service resolve the DEK from the stored KMSBlob.
func (e *EncryptedIndex) indexKeyField() internal.NullableString {
    if e.indexKey == nil {
        return internal.NullableString{}
    }
    return *internal.NewNullableString(e.indexKey)
}
```

Then in every request builder (`IsTrained`, `CheckTrainingStatus`, `Upsert`,
`Query` single + batch, `Get`, `Delete`, `Train`, `DeleteIndex`, `ListIDs`,
`BinaryUpsert`, `BinaryQuery`) replace:

```go
IndexKey: e.indexKey,                  // old: string
```

with:

```go
IndexKey: e.indexKeyField(),           // new: internal.NullableString
```

This is **14 callsites**. Worth a grep after editing.

### 4.3 `types.go`

**Rewrite `CreateIndexParams`** — `IndexKey` becomes `[]byte` with
`omitempty`; new fields `KmsName`, `Dimension`, `StoragePrecision`; drop
`IndexConfig`:

```go
type CreateIndexParams struct {
    IndexName        string  `json:"index_name"`
    IndexKey         []byte  `json:"index_key,omitempty"`
    KmsName          *string `json:"kms_name,omitempty"`
    Dimension        *int32  `json:"dimension,omitempty"`
    Metric           *string `json:"metric,omitempty"`
    EmbeddingModel   *string `json:"embedding_model,omitempty"`
    StoragePrecision *string `json:"storage_precision,omitempty"`
}
```

**Delete the entire `IndexModel` surface from the bottom of the file**:

- `IndexModel` interface
- `indexIVFFlat` / `indexIVFPQ` / `indexIVFSQ` wrapper structs
- `IndexIVFFlat()` / `IndexIVFPQ()` / `IndexIVFSQ()` constructor funcs
- All three `ToIndexConfig()` methods

That's ~115 lines removed. Nothing else in the public surface
references them once `CreateIndexParams.IndexConfig` is gone.

---

## 5. Test changes

### 5.1 `test/api_contract_test.go`

- Drop `TestIndexConfigTypes` entirely.
- Drop `ExposeIndexConfigViaGetter` (test was already broken — relied on
  `GetIndexConfig()` which is gone).
- Drop the `KNOWN SDK BUG: Go SDK requires IndexConfig` skip block — no
  longer a bug; embedding-model auto-detects dimension.
- Bulk-convert every `IndexConfig: cyborgdb.IndexIVFFlat(dim)` to
  `Dimension: &dim`. (Variables may need promoting to `int32` via a small
  helper — see 5.4.)
- Replace per-test hardcoded `"ivfflat"` / `"ivfpq"` / `"ivfsq"` expectations
  for `GetIndexType()` with a shared constant:

  ```go
  const expectedIndexType = "disk_ivf"
  ```

- **Add `TestSDKConstructionOffline`** (4 sub-tests, mirrors the Python
  `TestSDKConstructionOffline`):

  1. `CreateIndex` with neither `IndexKey` nor `KmsName` returns
     `ErrMissingKeyOrKMS`.
  2. `CreateIndexRequest` JSON marshals to a payload that includes
     `kms_name: "vendor-slot"` and **omits** `index_key` when only
     `KmsName` is set.
  3. `IndexOperationRequest` JSON marshals **without** `index_key` when
     only `IndexName` is set.
  4. All six data-plane request models
     (`QueryRequest`, `UpsertRequest`, `GetRequest`, `DeleteRequest`,
     `TrainRequest`, `ListIDsRequest`, plus
     `BinaryQueryRequest` + `BinaryUpsertRequest`) accept keyless
     construction with `index_key` omitted or null on the wire.

  The structural assertion in #4 is: marshal the request,
  `json.Unmarshal` into `map[string]any`, then either
  `_, present := payload["index_key"]; present` is false, or the value is
  literal `nil`. Both shapes are equivalent on the service side.

### 5.2 `test/kms_byok_test.go` (new file)

Env-gated live integration tests covering all three KMS posture variants.
Gating envs:

- `CYBORGDB_KMS_NAME_REAL` — `provider: aws-kms` slot (HSM-resident KEK)
- `CYBORGDB_KMS_NAME_SM` — `provider: aws` slot (Secrets Manager KEK)
- `CYBORGDB_KMS_NAME_NONE` — `provider: none` slot (SDK supplies KEK)

Test structure (one `kmsBYOKConfig` per slot, looped):

```go
type kmsBYOKConfig struct {
    envVar       string  // env var that gates the test
    kmsName      string  // resolved registry entry name
    label        string  // sub-test label
    needsSDKKey  bool    // provider: none variant — SDK supplies the KEK
    expectedType string  // index type the describe path should report
}
```

For each configured slot, exercise:

- `CreateIndex` (`KmsName` only for real-KMS, `IndexKey` only for `none`)
- `LoadIndex` (nil-key for real-KMS, key for `none`)
- Upsert, Query, Get, ListIDs, Delete
- `IsTrained`, `CheckTrainingStatus`

The point of running the full data-plane on every variant is to catch any
regression where a model lost its keyless path. Each test skips cleanly when
its env var isn't set.

### 5.3 `test/concurrency_test.go` — delete entirely

The whole `TestMixedIndexTypes*` block (`newMixedIndexSet`,
`TestMixedIndexTypesUpsertAndQuery`,
`TestMixedIndexTypesConcurrentWrites`,
`TestMixedIndexTypesInterleavedOperations`, ~428 lines) goes away. The
service now exposes a single DiskIVF index type — cross-type contamination
is structurally impossible. Mirrors the Python
`TestMixedIndexTypesOneClient` deletion.

### 5.4 `test/comprehensive_test.go`

- Bulk-convert `IndexConfig: cyborgdb.IndexIVFFlat(128)` to
  `Dimension: int32Ptr(128)`.
- Fold `TestIVFSQQueryCorrectness` and `TestIVFPQQueryCorrectness` into
  `TestDiskIVFSmallSetCorrectness`.
- Add a new `TestDiskIVFAutoDimensionCorrectness` that exercises the
  dimension-auto-detect path — create the index *without* setting
  `Dimension`, upsert vectors, verify the dimension was inferred.
- Add a tiny helper:

  ```go
  func int32Ptr(v int32) *int32 { return &v }
  ```

### 5.5 `test/helpers_test.go` and `test/quick_flow_test.go`

Same `IndexConfig → Dimension` conversion as the contract tests. Drop the
two `GetIndexConfig()` callsites. Update expected index type to `"disk_ivf"`.

---

## 6. README

Two edits:

1. Feature bullet: `Flexible Indexing: Support for multiple index types
   (IVF, IVFPQ, IVFFlat) ...` → `Encrypted DiskIVF Indexing: Disk-backed
   inverted-file index with customizable training parameters`.

2. Fix pre-existing bug in the basic-usage example: `IndexKey` was being
   passed as `fmt.Sprintf("%x", indexKey)` (a hex string) into a `[]byte`
   field. Pass the raw bytes directly, and add a `Dimension` field:

   ```go
   dimension := int32(128)
   createParams := &cyborgdb.CreateIndexParams{
       IndexName: "my-index",
       IndexKey:  indexKey,
       Dimension: &dimension,
   }
   ```

3. Add a **Bring Your Own Key (BYOK) via KMS** section under "Usage" with
   three code blocks:

   - KMS-backed create (`KmsName` only, no `IndexKey`).
   - KMS-backed load (`client.LoadIndex(ctx, name, nil)`).
   - No-KMS create (`IndexKey` only, `KmsName` omitted).

   Then a paragraph clarifying that `kms.registry` slots are configured by
   the cyborgdb-service operator (not the SDK), and pointing to `BYOK.md` in
   the cyborgdb-service repo for the operator + customer AWS IAM walkthrough.

The full BYOK section as written on `f59bacb` is the prose to copy verbatim
into the new branch (see `git show f59bacb:README.md` for the canonical
version).

---

## 7. Files touched, at a glance

```
Hand-written code:
  client.go                  — CreateIndex/LoadIndex rewritten
  encrypted_index.go         — *string indexKey + indexKeyField() helper
  types.go                   — CreateIndexParams rewritten, IndexModel surface deleted
  README.md                  — BYOK section + 2 small fixes
  update-openapi-client.sh   — switch to openapi-generator-cli
  openapitools.json          — new (pins generator version)
  openapi.json               — replaced from cyborgdb-py (v0.16.1)

Generated (do not hand-edit; `./update-openapi-client.sh` produces these):
  internal/*.go              — full regeneration

Tests:
  test/api_contract_test.go      — dim-only ctors, expectedIndexType, +TestSDKConstructionOffline
  test/comprehensive_test.go     — dim-only ctors, IVF tests folded into DiskIVF
  test/helpers_test.go           — dim-only ctors, drop GetIndexConfig calls
  test/quick_flow_test.go        — dim-only ctors
  test/concurrency_test.go       — mixed-types block deleted
  test/kms_byok_test.go          — new

Deletions (after regeneration):
  internal/model_index_config.go
  internal/model_index_ivf_model.go
  internal/model_index_ivf_flat_model.go
  internal/model_index_ivfpq_model.go
  internal/model_index_ivfsq_model.go
  (concurrency_test.go's mixed-types block)
```

---

## 8. Verification checklist (post-port)

```bash
# Build cleanly
go build ./...

# Offline tests compile and pass without a running service
go test -count=1 -run TestSDKConstructionOffline ./test/...

# Full compile check (run a no-op match so no live service is needed)
go test -count=1 -run NoSuchTest ./...

# Live BYOK (only after kms.registry is wired in cyborgdb.yaml)
export CYBORGDB_KMS_NAME_REAL=...
export CYBORGDB_KMS_NAME_SM=...
export CYBORGDB_KMS_NAME_NONE=...
go test -count=1 -run TestKMSBYOK ./test/...
```

Expected wire-shape behavior for the offline test:

- `kms_name` only → `kms_name` present, `index_key` absent (omitempty).
- `index_key` only → `index_key` present as 64-char hex, `kms_name` absent.
- Both set → both present on the wire (the SDK forwards them unchanged); the
  service then rejects the pair with a 400 — no provider accepts both.
- Neither set → `CreateIndex` errors with `ErrMissingKeyOrKMS` before the
  wire is touched.

---

## 9. Known follow-ups (out of scope for the port)

- **Live BYOK CI**: `test/kms_byok_test.go` needs a CI slot wiring the
  three env vars against a configured `cyborgdb.yaml`. Same gap exists on
  the Python side — coordinate when one lands.
- **Phase 2+ scope**: Vendor / admin / user routes and RBAC are still
  pending in cyborgdb-service; no SDK surface for them in this port.

---

## 10. Source artifacts on the local branch

These exist on `multi-tenancy@f59bacb` and can be copied verbatim into the
fresh branch — they're not affected by `origin/main` churn:

- `go-pr.md` — full PR description prose (motivation, risk, reviewer notes).
- `test/kms_byok_test.go` — full BYOK test file (~254 lines).
- `openapitools.json` — generator pin.

Pull any of them with `git show f59bacb:<path> > <path>`.
