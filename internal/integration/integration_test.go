//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"go-tenders-v3-main/internal/httpapi"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable"
	}
	return dsn
}

func schemaPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	return filepath.Join(dir, "..", "..", "schema.sql")
}

func resetDB(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	b, err := os.ReadFile(schemaPath(t))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(b)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}

func seedUsersAndOrg(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, `
		INSERT INTO employee (username) VALUES ('alice'), ('bob');
		INSERT INTO organization (name, type) VALUES ('Org A', 'LLC');
		INSERT INTO organization_responsible (organization_id, user_id)
		SELECT 1, id FROM employee WHERE username = 'alice';
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", testDSN(t))
	if err != nil {
		t.Fatalf("sql open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skipf("postgres not available (set TEST_POSTGRES_DSN): %v", err)
	}
	return db
}

func TestIntegration_Ping(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != "ok" {
		t.Fatalf("body %q", body)
	}
}

func TestIntegration_TenderPublishAndList(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	createBody := `{
		"name": "T1",
		"description": "d",
		"serviceType": "Delivery",
		"status": "Created",
		"organizationId": 1,
		"creatorUsername": "alice"
	}`
	res, err := client.Post(srv.URL+"/api/tenders/new", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("create tender: %d %s", res.StatusCode, b)
	}
	var tender struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tender); err != nil {
		t.Fatal(err)
	}
	if tender.Status != "Created" {
		t.Fatalf("status %q", tender.Status)
	}

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/tenders/"+tender.ID+"/status?status=Published&username=alice", nil)
	res2, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res2.Body)
		t.Fatalf("publish: %d %s", res2.StatusCode, b)
	}

	res3, err := client.Get(srv.URL + "/api/tenders")
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("list: %d", res3.StatusCode)
	}
	var list []map[string]any
	if err := json.NewDecoder(res3.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tender, got %d", len(list))
	}
}

func TestIntegration_BidUserFlow(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	// tender Created + Published
	createT := `{"name":"T","description":"` + strings.Repeat("x", 10) + `","serviceType":"Manufacture","status":"Created","organizationId":1,"creatorUsername":"alice"}`
	res, err := client.Post(srv.URL+"/api/tenders/new", "application/json", strings.NewReader(createT))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create tender: %d %s", res.StatusCode, body)
	}
	var tender struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &tender); err != nil {
		t.Fatal(err)
	}

	reqPub, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/tenders/"+tender.ID+"/status?status=Published&username=alice", nil)
	resPub, err := client.Do(reqPub)
	if err != nil {
		t.Fatal(err)
	}
	resPub.Body.Close()
	if resPub.StatusCode != http.StatusOK {
		t.Fatalf("publish tender: %d", resPub.StatusCode)
	}

	bidBody := `{
		"name": "B1",
		"description": "` + strings.Repeat("b", 20) + `",
		"status": "Created",
		"tenderId": "` + tender.ID + `",
		"organizationId": 0,
		"creatorUsername": "bob"
	}`
	resBid, err := client.Post(srv.URL+"/api/bids/new", "application/json", strings.NewReader(bidBody))
	if err != nil {
		t.Fatal(err)
	}
	bidRaw, _ := io.ReadAll(resBid.Body)
	resBid.Body.Close()
	if resBid.StatusCode != http.StatusOK {
		t.Fatalf("create bid: %d %s", resBid.StatusCode, bidRaw)
	}
	var bid map[string]any
	if err := json.Unmarshal(bidRaw, &bid); err != nil {
		t.Fatal(err)
	}
	bidID, _ := bid["id"].(string)
	if bidID == "" {
		t.Fatalf("no bid id: %s", bidRaw)
	}

	// tender org can list bids
	resList, err := client.Get(srv.URL + "/api/bids/" + tender.ID + "/list?username=alice")
	if err != nil {
		t.Fatal(err)
	}
	listRaw, _ := io.ReadAll(resList.Body)
	resList.Body.Close()
	if resList.StatusCode != http.StatusOK {
		t.Fatalf("list bids: %d %s", resList.StatusCode, listRaw)
	}
}

func createAndPublishTender(t *testing.T, client *http.Client, baseURL string) string {
	t.Helper()
	createT := `{"name":"T","description":"` + strings.Repeat("x", 10) + `","serviceType":"Manufacture","status":"Created","organizationId":1,"creatorUsername":"alice"}`
	res, err := client.Post(baseURL+"/api/tenders/new", "application/json", strings.NewReader(createT))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create tender: %d %s", res.StatusCode, body)
	}
	var tender struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &tender); err != nil {
		t.Fatal(err)
	}

	reqPub, _ := http.NewRequest(http.MethodPut, baseURL+"/api/tenders/"+tender.ID+"/status?status=Published&username=alice", nil)
	resPub, err := client.Do(reqPub)
	if err != nil {
		t.Fatal(err)
	}
	pubBody, _ := io.ReadAll(resPub.Body)
	_ = resPub.Body.Close()
	if resPub.StatusCode != http.StatusOK {
		t.Fatalf("publish tender: %d %s", resPub.StatusCode, pubBody)
	}
	return tender.ID
}

func createBid(t *testing.T, client *http.Client, baseURL, tenderID, username string) string {
	t.Helper()
	bidBody := `{
		"name": "B1",
		"description": "` + strings.Repeat("b", 20) + `",
		"status": "Created",
		"tenderId": "` + tenderID + `",
		"organizationId": 0,
		"creatorUsername": "` + username + `"
	}`
	resBid, err := client.Post(baseURL+"/api/bids/new", "application/json", strings.NewReader(bidBody))
	if err != nil {
		t.Fatal(err)
	}
	bidRaw, _ := io.ReadAll(resBid.Body)
	_ = resBid.Body.Close()
	if resBid.StatusCode != http.StatusOK {
		t.Fatalf("create bid: %d %s", resBid.StatusCode, bidRaw)
	}
	var bid map[string]any
	if err := json.Unmarshal(bidRaw, &bid); err != nil {
		t.Fatal(err)
	}
	bidID, _ := bid["id"].(string)
	if bidID == "" {
		t.Fatalf("no bid id: %s", bidRaw)
	}
	return bidID
}

func TestIntegration_DecisionQuorumClosesTender(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	// add second responsible to make quorum=2
	_, err := db.Exec(`INSERT INTO organization_responsible (organization_id, user_id)
		SELECT 1, id FROM employee WHERE username = 'bob'`)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	tenderID := createAndPublishTender(t, client, srv.URL)
	bidID := createBid(t, client, srv.URL, tenderID, "bob")

	req1, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/bids/"+bidID+"/submit_decision?decision=Approved&username=alice", nil)
	res1, err := client.Do(req1)
	if err != nil {
		t.Fatal(err)
	}
	if res1.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res1.Body)
		t.Fatalf("decision1: %d %s", res1.StatusCode, b)
	}
	_ = res1.Body.Close()

	req2, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/bids/"+bidID+"/submit_decision?decision=Approved&username=bob", nil)
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	raw2, _ := io.ReadAll(res2.Body)
	_ = res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("decision2: %d %s", res2.StatusCode, raw2)
	}
	var bid map[string]any
	_ = json.Unmarshal(raw2, &bid)
	if bid["status"] != "Approved" {
		t.Fatalf("expected bid Approved, got %v", bid["status"])
	}

	// tender should be closed automatically
	res3, err := client.Get(srv.URL + "/api/tenders/" + tenderID + "/status?username=alice")
	if err != nil {
		t.Fatal(err)
	}
	statusRaw, _ := io.ReadAll(res3.Body)
	_ = res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("tender status: %d %s", res3.StatusCode, statusRaw)
	}
	if strings.Trim(string(statusRaw), "\"\n ") != "Closed" {
		t.Fatalf("expected Closed, got %s", statusRaw)
	}
}

func TestIntegration_DecisionRejectWins(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	tenderID := createAndPublishTender(t, client, srv.URL)
	bidID := createBid(t, client, srv.URL, tenderID, "bob")

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/bids/"+bidID+"/submit_decision?decision=Rejected&username=alice", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("decision reject: %d %s", res.StatusCode, raw)
	}
	var bid map[string]any
	_ = json.Unmarshal(raw, &bid)
	if bid["status"] != "Rejected" {
		t.Fatalf("expected Rejected, got %v", bid["status"])
	}
}

func TestIntegration_FeedbackAndReviews(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	tenderID := createAndPublishTender(t, client, srv.URL)
	bidID := createBid(t, client, srv.URL, tenderID, "bob")

	reqFb, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/bids/"+bidID+"/feedback?bidFeedback=good+offer&username=alice", nil)
	resFb, err := client.Do(reqFb)
	if err != nil {
		t.Fatal(err)
	}
	fbRaw, _ := io.ReadAll(resFb.Body)
	_ = resFb.Body.Close()
	if resFb.StatusCode != http.StatusOK {
		t.Fatalf("feedback: %d %s", resFb.StatusCode, fbRaw)
	}

	resRev, err := client.Get(srv.URL + "/api/bids/" + tenderID + "/reviews?authorUsername=bob&requesterUsername=alice")
	if err != nil {
		t.Fatal(err)
	}
	revRaw, _ := io.ReadAll(resRev.Body)
	_ = resRev.Body.Close()
	if resRev.StatusCode != http.StatusOK {
		t.Fatalf("reviews: %d %s", resRev.StatusCode, revRaw)
	}
	var reviews []map[string]any
	if err := json.Unmarshal(revRaw, &reviews); err != nil {
		t.Fatal(err)
	}
	if len(reviews) == 0 {
		t.Fatalf("expected at least one review, got %s", revRaw)
	}
}

func TestIntegration_BidEditForbiddenForTenderResponsible(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	tenderID := createAndPublishTender(t, client, srv.URL)
	bidID := createBid(t, client, srv.URL, tenderID, "bob")

	editBody := `{"name":"Hacked"}`
	reqEdit, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/bids/"+bidID+"/edit?username=alice", strings.NewReader(editBody))
	reqEdit.Header.Set("Content-Type", "application/json")
	resEdit, err := client.Do(reqEdit)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resEdit.Body)
	_ = resEdit.Body.Close()
	if resEdit.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", resEdit.StatusCode, raw)
	}
}

func TestIntegration_TendersMyEditRollback(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	tenderID := createAndPublishTender(t, client, srv.URL)

	// my list
	resMy, err := client.Get(srv.URL + "/api/tenders/my?username=alice")
	if err != nil {
		t.Fatal(err)
	}
	rawMy, _ := io.ReadAll(resMy.Body)
	_ = resMy.Body.Close()
	if resMy.StatusCode != http.StatusOK {
		t.Fatalf("tenders my: %d %s", resMy.StatusCode, rawMy)
	}

	// edit
	editBody := `{"name":"T edited","description":"edited","serviceType":"Delivery"}`
	reqEdit, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/tenders/"+tenderID+"/edit?username=alice", strings.NewReader(editBody))
	reqEdit.Header.Set("Content-Type", "application/json")
	resEdit, err := client.Do(reqEdit)
	if err != nil {
		t.Fatal(err)
	}
	rawEdit, _ := io.ReadAll(resEdit.Body)
	_ = resEdit.Body.Close()
	if resEdit.StatusCode != http.StatusOK {
		t.Fatalf("tender edit: %d %s", resEdit.StatusCode, rawEdit)
	}

	// rollback to v1
	reqRb, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/tenders/"+tenderID+"/rollback/1?username=alice", nil)
	resRb, err := client.Do(reqRb)
	if err != nil {
		t.Fatal(err)
	}
	rawRb, _ := io.ReadAll(resRb.Body)
	_ = resRb.Body.Close()
	if resRb.StatusCode != http.StatusOK {
		t.Fatalf("tender rollback: %d %s", resRb.StatusCode, rawRb)
	}
}

func TestIntegration_BidsMyStatusEditRollback(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	tenderID := createAndPublishTender(t, client, srv.URL)
	bidID := createBid(t, client, srv.URL, tenderID, "bob")

	// my list
	resMy, err := client.Get(srv.URL + "/api/bids/my?username=bob")
	if err != nil {
		t.Fatal(err)
	}
	rawMy, _ := io.ReadAll(resMy.Body)
	_ = resMy.Body.Close()
	if resMy.StatusCode != http.StatusOK {
		t.Fatalf("bids my: %d %s", resMy.StatusCode, rawMy)
	}

	// get status as tender responsible (view)
	resSt, err := client.Get(srv.URL + "/api/bids/" + bidID + "/status?username=alice")
	if err != nil {
		t.Fatal(err)
	}
	rawSt, _ := io.ReadAll(resSt.Body)
	_ = resSt.Body.Close()
	if resSt.StatusCode != http.StatusOK {
		t.Fatalf("bid status get: %d %s", resSt.StatusCode, rawSt)
	}

	// update status as bid creator (mutable)
	reqPut, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/bids/"+bidID+"/status?status=Published&username=bob", nil)
	resPut, err := client.Do(reqPut)
	if err != nil {
		t.Fatal(err)
	}
	rawPut, _ := io.ReadAll(resPut.Body)
	_ = resPut.Body.Close()
	if resPut.StatusCode != http.StatusOK {
		t.Fatalf("bid status put: %d %s", resPut.StatusCode, rawPut)
	}

	// edit as creator
	reqEdit, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/bids/"+bidID+"/edit?username=bob", strings.NewReader(`{"name":"B2","description":"d2"}`))
	reqEdit.Header.Set("Content-Type", "application/json")
	resEdit, err := client.Do(reqEdit)
	if err != nil {
		t.Fatal(err)
	}
	rawEdit, _ := io.ReadAll(resEdit.Body)
	_ = resEdit.Body.Close()
	if resEdit.StatusCode != http.StatusOK {
		t.Fatalf("bid edit: %d %s", resEdit.StatusCode, rawEdit)
	}

	// rollback as creator
	reqRb, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/bids/"+bidID+"/rollback/1?username=bob", nil)
	resRb, err := client.Do(reqRb)
	if err != nil {
		t.Fatal(err)
	}
	rawRb, _ := io.ReadAll(resRb.Body)
	_ = resRb.Body.Close()
	if resRb.StatusCode != http.StatusOK {
		t.Fatalf("bid rollback: %d %s", resRb.StatusCode, rawRb)
	}
}

func TestIntegration_ForbiddenAndNotFoundBranches(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetDB(t, db)
	seedUsersAndOrg(t, db)

	srv := httptest.NewServer(httpapi.NewRouter(db))
	defer srv.Close()
	client := srv.Client()

	tenderID := createAndPublishTender(t, client, srv.URL)
	bidID := createBid(t, client, srv.URL, tenderID, "bob")

	// non-responsible cannot list bids for tender
	resList, err := client.Get(srv.URL + "/api/bids/" + tenderID + "/list?username=bob")
	if err != nil {
		t.Fatal(err)
	}
	_ = resList.Body.Close()
	if resList.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for list, got %d", resList.StatusCode)
	}

	// non-responsible cannot submit decision
	reqDecision, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/bids/"+bidID+"/submit_decision?decision=Approved&username=bob", nil)
	resDecision, err := client.Do(reqDecision)
	if err != nil {
		t.Fatal(err)
	}
	_ = resDecision.Body.Close()
	if resDecision.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for decision, got %d", resDecision.StatusCode)
	}

	// reviews author not in tender -> 404
	resRev, err := client.Get(srv.URL + "/api/bids/" + tenderID + "/reviews?authorUsername=alice&requesterUsername=alice")
	if err != nil {
		t.Fatal(err)
	}
	_ = resRev.Body.Close()
	if resRev.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for reviews author not found, got %d", resRev.StatusCode)
	}

	// bad ids -> 400
	resBadTender, err := client.Get(srv.URL + "/api/tenders/not-uuid/status?username=alice")
	if err != nil {
		t.Fatal(err)
	}
	_ = resBadTender.Body.Close()
	if resBadTender.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad tender id, got %d", resBadTender.StatusCode)
	}

	resBadBid, err := client.Get(srv.URL + "/api/bids/not-uuid/status?username=alice")
	if err != nil {
		t.Fatal(err)
	}
	_ = resBadBid.Body.Close()
	if resBadBid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad bid id, got %d", resBadBid.StatusCode)
	}
}
