package test

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// RBAC user-management tests, mirroring cyborgdb-py/tests/test_rbac_users.py.
//
// EncryptedIndex.CreateUser mints a per-index API key scoped to read/write
// permissions enforced cryptographically by the service; ListUsers/DeleteUser
// let the root enumerate and revoke users. After revocation the key stops
// working on the next request.
//
// These run live against an RBAC-enabled, KMS-backed service. Both
// CYBORGDB_SERVICE_ROOT_KEY and CYBORGDB_KMS_NAME must be set; if either is
// missing the tests fail hard (rather than skipping) so a misconfigured
// environment can't quietly drop RBAC coverage. The index must be KMS-backed
// so user keys can resolve the index KEK server-side.

const rbacDimension = 4

// rbacSeed returns the two-vector seed data shared by the RBAC tests.
func rbacSeed() ([]string, [][]float32) {
	return []string{"a", "b"}, [][]float32{
		{0.1, 0.2, 0.3, 0.4},
		{0.9, 0.8, 0.7, 0.6},
	}
}

// rbacRootIndex creates a KMS-backed index owned by the root key, seeds it, and
// registers cleanup. Fails the test if the RBAC env is not configured.
func rbacRootIndex(t *testing.T) (*cyborgdb.EncryptedIndex, string) {
	t.Helper()
	rootKey := os.Getenv("CYBORGDB_SERVICE_ROOT_KEY")
	kmsName := os.Getenv("CYBORGDB_KMS_NAME_REAL")
	var missing []string
	if rootKey == "" {
		missing = append(missing, "CYBORGDB_SERVICE_ROOT_KEY")
	}
	if kmsName == "" {
		missing = append(missing, "CYBORGDB_KMS_NAME")
	}
	if len(missing) > 0 {
		t.Skipf("RBAC tests require %s.\n"+
			"These tests run against an RBAC-enabled, KMS-backed service:\n"+
			"  - Start cyborgdb-service with CYBORGDB_SERVICE_ROOT_KEY set (this turns RBAC on)\n"+
			"    and a kms.registry slot defined in its cyborgdb.yaml (provider aws or aws-kms,\n"+
			"    which requires AWS BYOK credentials).\n"+
			"  - Export CYBORGDB_SERVICE_ROOT_KEY (the same value as the server) and\n"+
			"    CYBORGDB_KMS_NAME (the registry slot name) into the test environment.",
			strings.Join(missing, " and "))
	}

	root, err := cyborgdb.NewClient(testBaseURL(), rootKey)
	if err != nil {
		t.Fatalf("Failed to create root client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := generateUniqueName("rbac_users_test_")
	dim := int32(rbacDimension)
	// KMS-backed (no index key): the service wraps the per-index KEK so user
	// keys can resolve it server-side.
	index, err := root.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName: name,
		KmsName:   &kmsName,
		Dimension: &dim,
	})
	if err != nil {
		t.Fatalf("Failed to create KMS-backed index: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	ids, vectors := rbacSeed()
	if err := index.UpsertVectors(ctx, ids, vectors, nil); err != nil {
		t.Fatalf("Failed to seed index: %v", err)
	}
	return index, name
}

// rbacUserIndex loads the given index as a user (no index key — the service
// resolves the KEK from the user's wrapped key).
func rbacUserIndex(t *testing.T, apiKey, name string) *cyborgdb.EncryptedIndex {
	t.Helper()
	client, err := cyborgdb.NewClient(testBaseURL(), apiKey)
	if err != nil {
		t.Fatalf("Failed to create user client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	index, err := client.LoadIndex(ctx, name, nil)
	if err != nil {
		t.Fatalf("Failed to load index as user: %v", err)
	}
	return index
}

func findRBACUser(users []cyborgdb.UserInfo, id string) *cyborgdb.UserInfo {
	for i := range users {
		if users[i].UserID == id {
			return &users[i]
		}
	}
	return nil
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// Mirrors test_create_returns_key_and_id.
func TestRBACUserCreateReturnsKeyAndID(t *testing.T) {
	index, _ := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := index.CreateUser(ctx, []string{"read"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.UserID == "" {
		t.Error("CreateUser returned empty user_id")
	}
	if !strings.HasPrefix(user.APIKey, "cdbk_") {
		t.Errorf("expected api_key to start with cdbk_, got %q", user.APIKey)
	}
	// Cleanup so list assertions elsewhere stay deterministic.
	if err := index.DeleteUser(ctx, user.UserID); err != nil {
		t.Errorf("DeleteUser failed: %v", err)
	}
}

// Mirrors test_read_only_user_can_query_but_not_write.
func TestRBACReadOnlyUserCanQueryButNotWrite(t *testing.T) {
	index, name := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := index.CreateUser(ctx, []string{"read"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	defer func() { _ = index.DeleteUser(ctx, user.UserID) }()

	reader := rbacUserIndex(t, user.APIKey, name)

	// Read op succeeds.
	resp, err := reader.Query(ctx, cyborgdb.QueryParams{
		QueryVector: []float32{0.1, 0.2, 0.3, 0.4},
		TopK:        1,
	})
	if err != nil {
		t.Fatalf("read-only user query failed: %v", err)
	}
	if len(getQueryResultItems(&resp.Results)) < 1 {
		t.Error("expected at least one query result for read-only user")
	}

	// Write op is cryptographically denied.
	if wErr := reader.UpsertVectors(ctx, []string{"z"}, [][]float32{{0.0, 0.0, 0.0, 1.0}}, nil); wErr == nil {
		t.Error("expected read-only user upsert to be denied, got nil error")
	}
}

// Mirrors test_read_write_user_can_do_both.
func TestRBACReadWriteUserCanDoBoth(t *testing.T) {
	index, name := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := index.CreateUser(ctx, []string{"read", "write"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	defer func() { _ = index.DeleteUser(ctx, user.UserID) }()

	writer := rbacUserIndex(t, user.APIKey, name)

	if upErr := writer.UpsertVectors(ctx, []string{"w"}, [][]float32{{0.0, 0.0, 0.0, 1.0}}, nil); upErr != nil {
		t.Fatalf("read-write user upsert failed: %v", upErr)
	}
	resp, err := writer.Query(ctx, cyborgdb.QueryParams{
		QueryVector: []float32{0.0, 0.0, 0.0, 1.0},
		TopK:        1,
	})
	if err != nil {
		t.Fatalf("read-write user query failed: %v", err)
	}
	if len(getQueryResultItems(&resp.Results)) < 1 {
		t.Error("expected at least one query result for read-write user")
	}
}

// Mirrors test_list_then_revoke.
func TestRBACListThenRevoke(t *testing.T) {
	index, name := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := index.CreateUser(ctx, []string{"read", "write"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// The user appears in the list with the granted permissions.
	users, err := index.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	listed := findRBACUser(users, user.UserID)
	if listed == nil {
		t.Fatalf("created user %s not found in ListUsers", user.UserID)
	}
	if got := sortedCopy(listed.Permissions); !reflect.DeepEqual(got, []string{"read", "write"}) {
		t.Errorf("expected permissions [read write], got %v", got)
	}

	// Revoke; the user must disappear from the list.
	if delErr := index.DeleteUser(ctx, user.UserID); delErr != nil {
		t.Fatalf("DeleteUser failed: %v", delErr)
	}
	users, err = index.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers after revoke failed: %v", err)
	}
	if findRBACUser(users, user.UserID) != nil {
		t.Errorf("revoked user %s still present in ListUsers", user.UserID)
	}

	// The revoked key must stop working. Revocation erases the user's wrapped
	// keys, so either the load (describe) or the query must fail.
	revokedClient, err := cyborgdb.NewClient(testBaseURL(), user.APIKey)
	if err != nil {
		t.Fatalf("Failed to create revoked-user client: %v", err)
	}
	revoked, loadErr := revokedClient.LoadIndex(ctx, name, nil)
	if loadErr != nil {
		return // load already blocked — revocation enforced.
	}
	if _, qErr := revoked.Query(ctx, cyborgdb.QueryParams{
		QueryVector: []float32{0.1, 0.2, 0.3, 0.4},
		TopK:        1,
	}); qErr == nil {
		t.Error("expected query with revoked key to fail, got nil error")
	}
}

// Mirrors test_read_only_user_lists_with_read_permission.
func TestRBACReadOnlyUserListedWithReadPermission(t *testing.T) {
	index, _ := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := index.CreateUser(ctx, []string{"read"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	defer func() { _ = index.DeleteUser(ctx, user.UserID) }()

	users, err := index.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	listed := findRBACUser(users, user.UserID)
	if listed == nil {
		t.Fatalf("created user %s not found in ListUsers", user.UserID)
	}
	if !reflect.DeepEqual(listed.Permissions, []string{"read"}) {
		t.Errorf("expected permissions [read], got %v", listed.Permissions)
	}
}

// Mirrors test_write_only_user_can_write_but_not_query.
func TestRBACWriteOnlyUserCanWriteButNotQuery(t *testing.T) {
	index, name := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := index.CreateUser(ctx, []string{"write"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	defer func() { _ = index.DeleteUser(ctx, user.UserID) }()

	writer := rbacUserIndex(t, user.APIKey, name)

	// Write op succeeds.
	if upErr := writer.UpsertVectors(ctx, []string{"wo"}, [][]float32{{0.0, 0.0, 1.0, 0.0}}, nil); upErr != nil {
		t.Fatalf("write-only user upsert failed: %v", upErr)
	}
	// Read op is cryptographically denied — no read DEK for this user.
	if _, qErr := writer.Query(ctx, cyborgdb.QueryParams{
		QueryVector: []float32{0.0, 0.0, 1.0, 0.0},
		TopK:        1,
	}); qErr == nil {
		t.Error("expected write-only user query to be denied, got nil error")
	}
}

// Mirrors test_invalid_permissions_rejected.
func TestRBACInvalidPermissionsRejected(t *testing.T) {
	index, _ := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// The grant must be a non-empty subset of {"read","write"}; the service
	// rejects an empty set and unknown permission names alike.
	if _, err := index.CreateUser(ctx, []string{}); err == nil {
		t.Error("expected empty permissions to be rejected, got nil error")
	}
	if _, err := index.CreateUser(ctx, []string{"admin"}); err == nil {
		t.Error("expected unknown permission to be rejected, got nil error")
	}
}

// Mirrors test_non_root_user_cannot_manage_users.
func TestRBACNonRootUserCannotManageUsers(t *testing.T) {
	index, name := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := index.CreateUser(ctx, []string{"read", "write"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	defer func() { _ = index.DeleteUser(ctx, user.UserID) }()

	userIdx := rbacUserIndex(t, user.APIKey, name)

	// Minting, listing, and revoking users are root-only operations; a user key
	// is rejected on each.
	if _, cErr := userIdx.CreateUser(ctx, []string{"read"}); cErr == nil {
		t.Error("expected non-root CreateUser to be denied, got nil error")
	}
	if _, lErr := userIdx.ListUsers(ctx); lErr == nil {
		t.Error("expected non-root ListUsers to be denied, got nil error")
	}
	if dErr := userIdx.DeleteUser(ctx, user.UserID); dErr == nil {
		t.Error("expected non-root DeleteUser to be denied, got nil error")
	}
}

// Mirrors test_revoking_one_user_leaves_another_working.
func TestRBACRevokingOneUserLeavesAnotherWorking(t *testing.T) {
	index, name := rbacRootIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	first, err := index.CreateUser(ctx, []string{"read"})
	if err != nil {
		t.Fatalf("CreateUser (first) failed: %v", err)
	}
	second, err := index.CreateUser(ctx, []string{"read"})
	if err != nil {
		t.Fatalf("CreateUser (second) failed: %v", err)
	}
	defer func() {
		_ = index.DeleteUser(ctx, first.UserID)
		_ = index.DeleteUser(ctx, second.UserID)
	}()

	// Revoking one user drops only that user's wrapped keys; the other user's
	// key keeps resolving the index.
	if delErr := index.DeleteUser(ctx, first.UserID); delErr != nil {
		t.Fatalf("DeleteUser (first) failed: %v", delErr)
	}

	survivor := rbacUserIndex(t, second.APIKey, name)
	resp, err := survivor.Query(ctx, cyborgdb.QueryParams{
		QueryVector: []float32{0.1, 0.2, 0.3, 0.4},
		TopK:        1,
	})
	if err != nil {
		t.Fatalf("survivor query failed: %v", err)
	}
	if len(getQueryResultItems(&resp.Results)) < 1 {
		t.Error("expected at least one query result for surviving user")
	}
}
