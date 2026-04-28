## Description (Required)
Fix tests that broke after `Query()` in `cyborgdb-core` / `cyborgdb-service` was changed to return only `id` by default (previously returned `id`, `distance`, and `metadata` when no `include` argument was provided). Updated affected tests in `test/api_contract_test.go`, `test/comprehensive_test.go`, and `test/concurrency_test.go` to pass `Include: []string{"distance"}` (or `["distance", "metadata"]`) wherever they assert on those keys. Also added a new contract test that pins the new default behavior — `Query()` with no `Include` returns only `id` — mirroring the test added in `cyborgdb-js`. Includes one drive-by fix for a pre-existing flake in `quick_flow_test.go::test_06_upsert_to_trigger_auto_train` that races against server-side auto-train (replaced a fixed `time.Sleep` with `pollUntil`). Also migrates the test workflow from the now-removed 1Password `PIP_EXTRA_INDEX_URL` field to the CodeArtifact-via-OIDC pattern already in use in `cyborgdb-js` and `cyborgdb-py`.

## Related Issue (Required)
Mirrors the fix already landed in `cyborgdb-js` (commits `f161735` "distance fixed in tests" and `38b917b` "added test to make sure we should only return ids from query") and `cyborgdb-py` (commit `6a37cc9` "distances bug fix"). Brings the Go SDK test suite in line with the new server-side default `include` behavior.

## Scope of This PR (Required)

- [ ] Feature Implementation
- [ ] Refactoring
- [ ] Performance Improvement
- [ ] Security Fix
- [x] Bug Fix
- [ ] Other (describe below)

**If "Other" was selected, describe the scope here:**

## Test Changes (Required)

- **Added/Removed Tests:**

  Updated:
  - `test/api_contract_test.go::TestEncryptedIndexQuery/QueryWithSingleVectorFlatArray`
  - `test/api_contract_test.go::TestEncryptedIndexQuery/QueryWithNestedArraySingleVector`
  - `test/comprehensive_test.go::TestIVFSQQueryCorrectness`
  - `test/comprehensive_test.go::TestIVFPQQueryCorrectness`
  - `test/concurrency_test.go::TestConcurrentReadsAndWrites`
  - `test/concurrency_test.go::TestDeletesDuringQueries`
  - `test/concurrency_test.go::TestStressHighConcurrency`
  - `test/quick_flow_test.go::TestUnitFlow/test_06_upsert_to_trigger_auto_train` (drive-by flake fix)

  Added:
  - `test/api_contract_test.go::TestEncryptedIndexQuery/QueryDefaultIncludeReturnsOnlyID` — verifies `Query()` with no `Include` arg returns only `id` (no `distance`, `metadata`, or `vector`). Go equivalent of the new `cyborgdb-js` "should not return distance by default" test.

  - [ ] No test changes

- **Reason:**

  `Query()` no longer returns `distance` and `metadata` by default. Tests that asserted on those keys without passing them in `Include` were silently passing zero-value checks (e.g. `GetDistance() < 0` is trivially false when distance is unset) or failing on ordering checks. Updated each affected call site to pass the appropriate `Include` argument, then pinned the new default behavior with an explicit contract assertion.

  The `test_06_upsert_to_trigger_auto_train` change is unrelated to the distances bug — it's a pre-existing race where the test slept a fixed 1s after the upsert that crosses `RETRAIN_THRESHOLD=10000`, and `ListIDs` returns 0 transiently while server-side auto-train runs. Replaced the sleep with a `pollUntil` so the test waits for the IDs to converge instead.

  - [ ] No test changes

## Breaking Changes

- [ ] This PR introduces breaking changes

  **If checked, please describe:**

  - **Impact:**

  - **Migration Steps:**

(No SDK code changes in this PR — the breaking change in `Query()`'s default `Include` was made in `cyborgdb-core` / `cyborgdb-service`. This PR only updates the Go SDK's tests to match the new contract.)

## Performance & Security Considerations

- [x] No known performance impact
- [x] No security concerns
- [ ] Requires additional security review

## Checklist

- [x] Code follows project style guidelines
- [x] Tests have been updated if needed
- [ ] Documentation has been updated if applicable

## Additional Context
Verified the affected suites (`api_contract_test.go`, `comprehensive_test.go`, `concurrency_test.go`, `quick_flow_test.go`) against a local `cyborgdb-service` instance. Audited the updated tests against the corresponding `cyborgdb-py` (`tests/test_api_contract.py`, `tests/test_client.py`, `tests/test_concurrency.py`) and `cyborgdb-js` (`src/__tests__/api_contract.test.ts`, `src/__tests__/concurrency_test.test.ts`) files to confirm parity — every Go query assertion that checks `distance` or `metadata` now passes those keys via `Include`, matching the JS and Python suites.

---
_Information regarding test changes will be automatically stored in the test log._
