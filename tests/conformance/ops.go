package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// handler holds the handles ops bind with `as` and look up by name.
type handler struct {
	api     api
	variant string // json or binary: which SDK path upsert and query take
	getenv  func(string) string
	clients map[string]client
	indexes map[string]index
}

func newHandler(a api, variant string, getenv func(string) string) *handler {
	return &handler{api: a, variant: variant, getenv: getenv, clients: map[string]client{}, indexes: map[string]index{}}
}

var ctx = context.Background()

// native turns an SDK call's error into the op's error. Every SDK error should be a typed
// cyborgdb.Error carrying its kind; an untyped one is still the product's answer, so it is
// reported as internal rather than as an adapter failure.
func native(err error) {
	if err == nil {
		return
	}
	var e cyborgdb.Error
	if !errors.As(err, &e) {
		panic(&rpcError{Code: codeOpFailed, Message: err.Error(), Data: &errData{Kind: "internal", Native: fmt.Sprintf("%T", err)}})
	}
	kind := e.Kind()
	if kind == "" {
		kind = "internal"
	}
	name := fmt.Sprintf("%T", e)
	if e.StatusCode() != 0 {
		name = fmt.Sprintf("%s %d", name, e.StatusCode())
	}
	panic(&rpcError{Code: codeOpFailed, Message: err.Error(), Data: &errData{Kind: kind, Native: name}})
}

func (h *handler) clientOf(p params) client {
	name := str(p.need("client"), "client")
	c, ok := h.clients[name]
	if !ok {
		failf(codeBadParams, "unknown client handle %s", name)
	}
	return c
}

// indexOf is the op's index. The SDK binds the key at create/load, so a per-call override in
// any of the named params isn't expressible.
func (h *handler) indexOf(p params, overrides ...string) index {
	reject(p, overrides...)
	name := str(p.need("index"), "index")
	ix, ok := h.indexes[name]
	if !ok {
		failf(codeBadParams, "unknown index handle %s", name)
	}
	return ix
}

// reject fails with -32601 if any of the named params is present.
func reject(p params, names ...string) {
	for _, n := range names {
		if _, ok := p.opt(n); ok {
			failf(codeUnsupported, "the Go SDK can't express %s", n)
		}
	}
}

var ops = map[string]func(*handler, params) any{
	"client.new":              (*handler).clientNew,
	"client.list_indexes":     (*handler).listIndexes,
	"client.health":           (*handler).health,
	"client.embedding_models": (*handler).embeddingModels,
	"client.training_status":  (*handler).trainingStatus,
	"client.create_index":     (*handler).createIndex,
	"client.load_index":       (*handler).loadIndex,
	"index.describe":          (*handler).describe,
	"index.num_vectors":       (*handler).numVectors,
	"index.delete_index":      (*handler).deleteIndex,
	"index.upsert":            (*handler).upsert,
	"index.get":               (*handler).get,
	"index.delete":            (*handler).delete,
	"index.list_ids":          (*handler).listIDs,
	"index.train":             (*handler).train,
	"index.query":             (*handler).query,
	"index.query_metadata":    (*handler).queryMetadata,
	"index.create_user":       (*handler).createUser,
	"index.list_users":        (*handler).listUsers,
	"index.delete_user":       (*handler).deleteUser,
}

// The service's KMS slot names, as the driver reads them (cyborgdb-tests internal/run/secrets.go).
var kmsSlotVariables = []string{
	"CYBORGDB_TESTS_KMS_NAME_AWS_KMS", "CYBORGDB_KMS_NAME_REAL", "CYBORGDB_TESTS_KMS_NAME_AWS", "CYBORGDB_KMS_NAME_SM",
}

func (h *handler) hello() any {
	caps := []string{"vector", "auto_dimension", "metadata_schema", "pattern", "bm25", "text_fields_sugar",
		"training", "storage_precision", "rbac", "embedding", "health", "training_status", "service_auth",
		// The data lives in the service, so it outlives an adapter restart.
		"durable_storage"}
	if h.variant == variantJSON {
		caps = append(caps, "ragged_batches") // JSON items carry each vector on its own
	} else {
		caps = append(caps, "parallel_metadata") // BinaryUpsertParams takes the metadata list as sent
	}
	for _, v := range kmsSlotVariables {
		if h.getenv(v) != "" {
			caps = append(caps, "kms_provider")
			break
		}
	}
	return map[string]any{"protocol": protocolVersion, "impl": "cyborgdb-go", "version": sdkVersion, "capabilities": caps}
}

// Client ops

func (h *handler) clientNew(p params) any {
	as := str(p.need("as"), "as")
	reject(p, "storage", "cpu_threads", "gpu")
	apiKey := ""
	if v, ok := p.opt("api_key"); ok {
		apiKey = str(v, "api_key")
	}
	c, err := h.api.NewClient(str(p.need("base_url"), "base_url"), apiKey)
	native(err)
	h.clients[as] = c
	return nil
}

func (h *handler) listIndexes(p params) any {
	names, err := h.clientOf(p).ListIndexes(ctx)
	native(err)
	return names
}

func (h *handler) health(p params) any {
	status, err := h.clientOf(p).GetHealth(ctx)
	native(err)
	return status
}

func (h *handler) embeddingModels(p params) any {
	models, err := h.clientOf(p).ListEmbeddingModels(ctx)
	native(err)
	return models
}

func (h *handler) trainingStatus(p params) any {
	status, err := h.clientOf(p).GetTrainingStatus(ctx)
	native(err)
	return status
}

// schemaOf decodes metadata_schema strictly: a shape MetadataFieldPolicy can't hold isn't expressible.
func schemaOf(v json.RawMessage) map[string]cyborgdb.MetadataFieldPolicy {
	var fields map[string]params
	if json.Unmarshal(v, &fields) != nil || fields == nil {
		failf(codeUnsupported, "metadata_schema must be an object of objects")
	}
	schema := make(map[string]cyborgdb.MetadataFieldPolicy, len(fields))
	for name, policy := range fields {
		var fp cyborgdb.MetadataFieldPolicy
		for k, x := range policy {
			b := flag(x, "metadata_schema."+k)
			switch k {
			case "filterable":
				fp.Filterable = &b
			case "pattern":
				fp.Pattern = &b
			case "full_text":
				fp.FullText = &b
			default:
				failf(codeUnsupported, "MetadataFieldPolicy has no %s", k)
			}
		}
		schema[name] = fp
	}
	return schema
}

func (h *handler) createIndex(p params) any {
	c := h.clientOf(p)
	as := str(p.need("as"), "as")
	reject(p, "kms")
	o := &cyborgdb.CreateIndexParams{IndexName: str(p.need("name"), "name")}
	if v, ok := p.opt("key"); ok {
		o.IndexKey = bytesOf(v, "key")
	}
	if v, ok := p.opt("kms_name"); ok {
		o.KmsName = ptr(str(v, "kms_name"))
	}
	if v, ok := p.opt("dimension"); ok {
		o.Dimension = ptr(int32Of(v, "dimension"))
	}
	if v, ok := p.opt("metric"); ok {
		o.Metric = ptr(str(v, "metric"))
	}
	if v, ok := p.opt("storage_precision"); ok {
		o.StoragePrecision = ptr(str(v, "storage_precision"))
	}
	if v, ok := p.opt("embedding_model"); ok {
		o.EmbeddingModel = ptr(str(v, "embedding_model"))
	}
	if v, ok := p.opt("metadata_schema"); ok {
		o.MetadataSchema = schemaOf(v)
	}
	if v, ok := p.opt("text_fields"); ok {
		o.TextFields = stringsOf(v, "text_fields")
	}
	if v, ok := p.opt("bm25_k1"); ok {
		o.Bm25K1 = ptr(num(v, "bm25_k1"))
	}
	if v, ok := p.opt("bm25_b"); ok {
		o.Bm25B = ptr(num(v, "bm25_b"))
	}
	ix, err := c.CreateIndex(ctx, o)
	native(err)
	h.indexes[as] = ix
	return nil
}

func (h *handler) loadIndex(p params) any {
	c := h.clientOf(p)
	as := str(p.need("as"), "as")
	reject(p, "user_id")
	var key []byte
	if v, ok := p.opt("key"); ok {
		key = bytesOf(v, "key")
	}
	ix, err := c.LoadIndex(ctx, str(p.need("name"), "name"), key)
	native(err)
	h.indexes[as] = ix
	return nil
}

// Index ops

func (h *handler) describe(p params) any {
	d, err := h.indexOf(p).Describe(ctx)
	native(err)
	b, err := json.Marshal(d)
	if err != nil {
		failf(codeAdapter, "encode IndexDescription: %v", err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		failf(codeAdapter, "decode IndexDescription: %v", err)
	}
	out["name"] = out["index_name"] // the contract calls it `name`
	delete(out, "index_name")
	return out
}

func (h *handler) numVectors(p params) any {
	n, err := h.indexOf(p, "key", "user_id").GetNumVectors(ctx)
	native(err)
	return n
}

func (h *handler) deleteIndex(p params) any {
	native(h.indexOf(p, "key").DeleteIndex(ctx))
	return nil
}

// metadataOf is one item's metadata; the SDK takes an object.
func metadataOf(v json.RawMessage) map[string]any {
	m, ok := document(v).(map[string]any)
	if !ok {
		failf(codeUnsupported, "the SDK takes metadata as an object")
	}
	return m
}

func (h *handler) upsert(p params) any {
	ix := h.indexOf(p, "key", "user_id")
	var input cyborgdb.UpsertInput
	switch _, items := p.opt("items"); {
	case h.variant == variantBinary && items:
		input = binaryColumns(itemsOf(p.need("items")))
	case h.variant == variantBinary:
		input = binaryParallel(p)
	case items:
		input = jsonItems(itemsOf(p.need("items")))
	default:
		input = jsonFromParallel(p)
	}
	native(ix.Upsert(ctx, input))
	return nil
}

func itemsOf(v json.RawMessage) []params {
	var items []params
	if json.Unmarshal(v, &items) != nil {
		failf(codeUnsupported, "items must be a list of objects")
	}
	return items
}

// jsonItems maps items onto VectorItems field by field: an absent field stays unset.
func jsonItems(items []params) cyborgdb.VectorItems {
	out := make(cyborgdb.VectorItems, len(items))
	for i, it := range items {
		for k, x := range it {
			switch k {
			case "id":
				out[i].Id = str(x, "id")
			case "vector":
				if !isNull(x) {
					out[i].Vector = vectorOf(x)
				}
			case "contents":
				if isNull(x) {
					out[i].SetContentsNil()
					break
				}
				var s string
				if json.Unmarshal(x, &s) == nil {
					out[i].SetContents(cyborgdb.Contents{String: &s})
				} else {
					b := bytesOf(x, "contents")
					out[i].SetContents(cyborgdb.Contents{Bytes: &b})
				}
			case "metadata":
				if !isNull(x) {
					out[i].Metadata = metadataOf(x)
				}
			default:
				failf(codeUnsupported, "VectorItem has no %s", k)
			}
		}
		// VectorItem always sends its id, so an item without one can't be expressed.
		if _, ok := it.opt("id"); !ok {
			failf(codeUnsupported, "a VectorItem always carries an id")
		}
	}
	return out
}

// jsonFromParallel builds VectorItems from the ids + vectors (+ metadata) form; the JSON path has
// no parallel form (ADAPTERS.md: choose the call form the variant says). Lists of different
// lengths can't be paired into items.
func jsonFromParallel(p params) cyborgdb.VectorItems {
	ids := stringsOf(p.need("ids"), "ids")
	rows, _ := vectorsOf(p.need("vectors"))
	if len(rows) != len(ids) {
		failf(codeUnsupported, "%d ids and %d vectors can't be paired into items", len(ids), len(rows))
	}
	var metadata []json.RawMessage
	if v, ok := p.opt("metadata"); ok {
		if json.Unmarshal(v, &metadata) != nil {
			failf(codeUnsupported, "metadata must be a list")
		}
		if len(metadata) != len(ids) {
			failf(codeUnsupported, "%d ids and %d metadata entries can't be paired into items", len(ids), len(metadata))
		}
	}
	out := make(cyborgdb.VectorItems, len(ids))
	for i := range ids {
		out[i].Id, out[i].Vector = ids[i], rows[i]
		if metadata != nil && !isNull(metadata[i]) {
			out[i].Metadata = metadataOf(metadata[i])
		}
	}
	return out
}

// binaryParallel passes the ids + vectors (+ metadata) form straight to BinaryUpsertParams.
func binaryParallel(p params) cyborgdb.BinaryUpsertParams {
	rows, _ := vectorsOf(p.need("vectors"))
	b := cyborgdb.BinaryUpsertParams{IDs: stringsOf(p.need("ids"), "ids"), Vectors: rows}
	if v, ok := p.opt("metadata"); ok {
		var entries []json.RawMessage
		if json.Unmarshal(v, &entries) != nil {
			failf(codeUnsupported, "metadata must be a list")
		}
		b.Metadata = make([]map[string]any, len(entries))
		for i, e := range entries {
			if !isNull(e) {
				b.Metadata[i] = metadataOf(e)
			}
		}
	}
	return b
}

// binaryColumns turns items into BinaryUpsertParams' columns. Every item needs an id and a
// vector; contents are strings, with "" meaning none, so bytes or an explicit "" can't be carried.
func binaryColumns(items []params) cyborgdb.BinaryUpsertParams {
	var b cyborgdb.BinaryUpsertParams
	var hasMetadata, hasContents bool
	for _, it := range items {
		_, m := it.opt("metadata")
		_, c := it.opt("contents")
		hasMetadata, hasContents = hasMetadata || m, hasContents || c
	}
	for _, it := range items {
		for k := range it {
			if k != "id" && k != "vector" && k != "contents" && k != "metadata" {
				failf(codeUnsupported, "the binary upsert can't carry %s", k)
			}
		}
		id, okID := it.opt("id")
		vec, okVec := it.opt("vector")
		if !okID || !okVec || isNull(vec) {
			failf(codeUnsupported, "the binary upsert needs an id and a vector for every item")
		}
		b.IDs = append(b.IDs, str(id, "id"))
		b.Vectors = append(b.Vectors, vectorOf(vec))
		if hasMetadata {
			var m map[string]any
			if x, ok := it.opt("metadata"); ok && !isNull(x) {
				m = metadataOf(x)
			}
			b.Metadata = append(b.Metadata, m)
		}
		if hasContents {
			s := ""
			if x, ok := it.opt("contents"); ok && !isNull(x) {
				if json.Unmarshal(x, &s) != nil {
					failf(codeUnsupported, "the binary upsert takes contents as strings, not bytes")
				}
				if s == "" {
					failf(codeUnsupported, "the binary upsert can't tell empty contents from none")
				}
			}
			b.Contents = append(b.Contents, s)
		}
	}
	return b
}

func (h *handler) get(p params) any {
	ix := h.indexOf(p, "key", "user_id")
	var include []string // nil when omitted: the SDK's own default
	if v, ok := p.opt("include"); ok {
		include = stringsOf(v, "include")
	}
	resp, err := ix.Get(ctx, stringsOf(p.need("ids"), "ids"), include)
	native(err)
	rows := make([]map[string]any, len(resp.Results))
	for i, r := range resp.Results {
		// Exactly the fields the SDK returned: absent stays absent.
		row := map[string]any{"id": r.Id}
		if r.Vector != nil {
			row["vector"] = f32Out(r.Vector)
		}
		if r.Contents.IsSet() {
			if s := r.Contents.Get(); s != nil {
				row["contents"] = bytesOut([]byte(*s))
			} else {
				row["contents"] = nil
			}
		}
		if r.Metadata != nil {
			row["metadata"] = r.Metadata
		}
		rows[i] = row
	}
	return rows
}

func (h *handler) delete(p params) any {
	ix := h.indexOf(p, "key", "user_id")
	native(ix.Delete(ctx, stringsOf(p.need("ids"), "ids")))
	return nil
}

func (h *handler) listIDs(p params) any {
	resp, err := h.indexOf(p, "key", "user_id").ListIDs(ctx)
	native(err)
	return resp.Ids // unwrap the documented {ids, count} envelope
}

func (h *handler) train(p params) any {
	ix := h.indexOf(p, "key", "user_id")
	var o cyborgdb.TrainParams
	for k, dst := range map[string]**int32{"n_lists": &o.NLists, "batch_size": &o.BatchSize, "max_iters": &o.MaxIters, "max_memory": &o.MaxMemory} {
		if v, ok := p.opt(k); ok {
			*dst = ptr(int32Of(v, k))
		}
	}
	if v, ok := p.opt("tolerance"); ok {
		o.Tolerance = ptr(num(v, "tolerance"))
	}
	native(ix.Train(ctx, o))
	return nil
}

// filterOf is a filter object; the SDK has no form for filter JSON text.
func filterOf(v json.RawMessage) map[string]any {
	m, ok := document(v).(map[string]any)
	if !ok {
		failf(codeUnsupported, "the SDK takes filters as an object")
	}
	return m
}

// queryKnobs are the query params shared by QueryParams and BinaryQueryParams.
type queryKnobs struct {
	topK                int32
	nProbes, rerankMult *int32
	windowMult          *int32
	greedy, requireAll  *bool
	filters             map[string]any
	include, textFields []string
	textFieldWeights    []float32
	text                *string
	alpha, rrfK         *float64
}

func knobsOf(p params) queryKnobs {
	var q queryKnobs
	if v, ok := p.opt("top_k"); ok {
		q.topK = int32Of(v, "top_k")
	}
	for k, dst := range map[string]**int32{"n_probes": &q.nProbes, "rerank_mult": &q.rerankMult, "window_mult": &q.windowMult} {
		if v, ok := p.opt(k); ok {
			*dst = ptr(int32Of(v, k))
		}
	}
	for k, dst := range map[string]**bool{"greedy": &q.greedy, "require_all_terms": &q.requireAll} {
		if v, ok := p.opt(k); ok {
			*dst = ptr(flag(v, k))
		}
	}
	for k, dst := range map[string]**float64{"alpha": &q.alpha, "rrf_k": &q.rrfK} {
		if v, ok := p.opt(k); ok {
			*dst = ptr(num(v, k))
		}
	}
	if v, ok := p.opt("filters"); ok {
		q.filters = filterOf(v)
	}
	if v, ok := p.opt("include"); ok {
		q.include = stringsOf(v, "include")
	}
	if v, ok := p.opt("text"); ok {
		q.text = ptr(str(v, "text"))
	}
	if v, ok := p.opt("text_fields"); ok {
		q.textFields = stringsOf(v, "text_fields")
	}
	if v, ok := p.opt("text_field_weights"); ok {
		q.textFieldWeights = floats(v, "text_field_weights")
	}
	return q
}

func (h *handler) query(p params) any {
	ix := h.indexOf(p, "key", "user_id")
	q := knobsOf(p)
	var input cyborgdb.QueryInput
	if h.variant == variantBinary {
		reject(p, "contents")
		v, ok := p.opt("vectors")
		if !ok {
			failf(codeUnsupported, "the binary query needs vectors")
		}
		rows, _ := vectorsOf(v)
		input = cyborgdb.BinaryQueryParams{
			QueryVectors: rows, TopK: q.topK, NProbes: q.nProbes, Greedy: q.greedy, RerankMult: q.rerankMult,
			Filters: q.filters, Include: q.include, Text: q.text, TextFields: q.textFields,
			TextFieldWeights: q.textFieldWeights, RequireAllTerms: q.requireAll, Alpha: q.alpha, RrfK: q.rrfK,
			WindowMult: q.windowMult,
		}
	} else {
		qp := cyborgdb.QueryParams{
			TopK: q.topK, NProbes: q.nProbes, Greedy: q.greedy, RerankMult: q.rerankMult,
			Filters: q.filters, Include: q.include, Text: q.text, TextFields: q.textFields,
			TextFieldWeights: q.textFieldWeights, RequireAllTerms: q.requireAll, Alpha: q.alpha, RrfK: q.rrfK,
			WindowMult: q.windowMult,
		}
		if v, ok := p.opt("vectors"); ok {
			// One vector goes as QueryVector, rows as BatchQueryVectors, as the spec sent them.
			if rows, nested := vectorsOf(v); nested {
				qp.BatchQueryVectors = rows
			} else {
				qp.QueryVector = rows[0]
			}
		}
		if v, ok := p.opt("contents"); ok {
			var s string
			if json.Unmarshal(v, &s) != nil {
				failf(codeUnsupported, "QueryContents takes one string")
			}
			qp.QueryContents = &s
		}
		input = qp
	}
	resp, err := ix.Query(ctx, input)
	native(err)
	// One list per query row: the service returns a flat list for a single query.
	var rows [][]cyborgdb.QueryResultItem
	switch r := resp.Results; {
	case r.ArrayOfArrayOfQueryResultItem != nil:
		rows = *r.ArrayOfArrayOfQueryResultItem
	case r.ArrayOfQueryResultItem != nil:
		rows = [][]cyborgdb.QueryResultItem{*r.ArrayOfQueryResultItem}
	default:
		failf(codeAdapter, "query response has no results")
	}
	out := make([][]map[string]any, len(rows))
	for r, hits := range rows {
		out[r] = make([]map[string]any, len(hits))
		for i, hit := range hits {
			out[r][i] = hitOut(hit)
		}
	}
	return out
}

// hitOut is one query result with exactly the fields the SDK returned.
func hitOut(hit cyborgdb.QueryResultItem) map[string]any {
	row := map[string]any{"id": hit.Id}
	if hit.Distance.IsSet() {
		row["distance"] = nil
		if d := hit.Distance.Get(); d != nil {
			row["distance"] = number(*d)
		}
	}
	if hit.Score.IsSet() {
		row["score"] = nil
		if s := hit.Score.Get(); s != nil {
			row["score"] = number(*s)
		}
	}
	if hit.Metadata != nil {
		row["metadata"] = hit.Metadata
	}
	if hit.Vector != nil {
		row["vector"] = f32Out(hit.Vector)
	}
	return row
}

func (h *handler) queryMetadata(p params) any {
	ix := h.indexOf(p, "key", "user_id")
	reject(p, "n_probes", "rerank_mult", "window_mult", "greedy", "alpha", "rrf_k", "include")
	q := knobsOf(p)
	o := cyborgdb.QueryMetadataParams{
		Filters: q.filters, TopK: q.topK, Text: q.text, TextFields: q.textFields,
		TextFieldWeights: q.textFieldWeights, RequireAllTerms: q.requireAll,
	}
	if v, ok := p.opt("order_by"); ok {
		var field string
		if json.Unmarshal(v, &field) != nil {
			failf(codeUnsupported, "QueryMetadataParams.OrderBy takes a field name; the {field: 1|-1} form isn't expressible")
		}
		o.OrderBy = field
	}
	if v, ok := p.opt("ascending"); ok {
		o.Ascending = ptr(flag(v, "ascending"))
	}
	resp, err := ix.QueryMetadata(ctx, o)
	native(err)
	if resp.Results == nil { // a service without rows: ids only
		out := make([]map[string]any, len(resp.Ids))
		for i, id := range resp.Ids {
			out[i] = map[string]any{"id": id}
		}
		return out
	}
	out := make([]map[string]any, 0, len(resp.Results))
	for _, r := range resp.Results {
		row := map[string]any{"id": r.Id}
		if r.Score != nil {
			row["score"] = number(*r.Score)
		}
		out = append(out, row)
	}
	return out
}

func (h *handler) createUser(p params) any {
	ix := h.indexOf(p, "key", "user_id", "user_kek")
	u, err := ix.CreateUser(ctx, stringsOf(p.need("permissions"), "permissions"))
	native(err)
	return u
}

func (h *handler) listUsers(p params) any {
	users, err := h.indexOf(p, "key").ListUsers(ctx)
	native(err)
	return users
}

func (h *handler) deleteUser(p params) any {
	ix := h.indexOf(p, "key")
	v := p.need("user_id")
	var id string
	if json.Unmarshal(v, &id) != nil {
		id = hex.EncodeToString(bytesOf(v, "user_id")) // the SDK's hex form
	}
	native(ix.DeleteUser(ctx, id))
	return nil
}
