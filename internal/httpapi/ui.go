package httpapi

import "net/http"

func portfolioUIHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(portfolioUIHTML))
}

const portfolioUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Tenders API Portfolio UI</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 20px; background: #f7f8fa; color: #111; }
    h1 { margin-bottom: 6px; }
    .muted { color: #666; margin-top: 0; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 12px; }
    .card { background: #fff; border: 1px solid #ddd; border-radius: 8px; padding: 12px; }
    .row { display: grid; gap: 8px; margin-bottom: 8px; }
    input, select, button, textarea { padding: 8px; font-size: 14px; }
    button { cursor: pointer; }
    pre { background: #0b1020; color: #d4f8d4; padding: 10px; border-radius: 8px; overflow: auto; min-height: 160px; }
    label { font-size: 12px; color: #444; }
  </style>
</head>
<body>
  <h1>Tenders API Demo UI</h1>
  <p class="muted">Minimal portfolio interface for core API actions.</p>

  <div class="grid">
    <section class="card">
      <h3>Ping</h3>
      <button onclick="ping()">GET /api/ping</button>
    </section>

    <section class="card">
      <h3>Create Tender</h3>
      <div class="row">
        <label>Name</label><input id="tName" value="Portfolio Tender">
        <label>Description</label><input id="tDesc" value="Created from portfolio UI">
        <label>Service Type</label>
        <select id="tService"><option>Construction</option><option selected>Delivery</option><option>Manufacture</option></select>
        <label>Organization ID</label><input id="tOrg" value="1" type="number">
        <label>Creator Username</label><input id="tUser" value="alice">
      </div>
      <button onclick="createTender()">POST /api/tenders/new</button>
      <button onclick="listTenders()">GET /api/tenders</button>
      <button onclick="myTenders()">GET /api/tenders/my</button>
    </section>

    <section class="card">
      <h3>Tender Actions</h3>
      <div class="row">
        <label>Tender ID</label><input id="taTenderId">
        <label>Username</label><input id="taUser" value="alice">
        <label>Status</label><select id="taStatus"><option>Created</option><option selected>Published</option><option>Closed</option></select>
        <label>Edit Name</label><input id="taEditName" value="Edited Tender Name">
        <label>Rollback Version</label><input id="taVersion" value="1" type="number">
      </div>
      <button onclick="tenderStatusGet()">GET status</button>
      <button onclick="tenderStatusPut()">PUT status</button>
      <button onclick="tenderEdit()">PATCH edit</button>
      <button onclick="tenderRollback()">PUT rollback</button>
    </section>

    <section class="card">
      <h3>Create Bid</h3>
      <div class="row">
        <label>Name</label><input id="bName" value="Portfolio Bid">
        <label>Description</label><input id="bDesc" value="Bid from portfolio UI">
        <label>Tender ID</label><input id="bTenderId">
        <label>Organization ID (0 for user bid)</label><input id="bOrgId" value="0" type="number">
        <label>Creator Username</label><input id="bUser" value="bob">
      </div>
      <button onclick="createBid()">POST /api/bids/new</button>
      <button onclick="myBids()">GET /api/bids/my</button>
      <button onclick="bidsForTender()">GET /api/bids/{tenderId}/list</button>
    </section>

    <section class="card">
      <h3>Bid Actions</h3>
      <div class="row">
        <label>Bid ID</label><input id="baBidId">
        <label>Tender ID (for reviews)</label><input id="baTenderId">
        <label>Username</label><input id="baUser" value="alice">
        <label>Status</label><select id="baStatus"><option>Created</option><option selected>Published</option><option>Canceled</option><option>Approved</option><option>Rejected</option></select>
        <label>Decision</label><select id="baDecision"><option selected>Approved</option><option>Rejected</option></select>
        <label>Feedback</label><input id="baFeedback" value="Great offer">
        <label>Author Username (reviews)</label><input id="baAuthor" value="bob">
      </div>
      <button onclick="bidStatusGet()">GET status</button>
      <button onclick="bidStatusPut()">PUT status</button>
      <button onclick="bidEdit()">PATCH edit</button>
      <button onclick="bidRollback()">PUT rollback</button>
      <button onclick="bidDecision()">PUT submit_decision</button>
      <button onclick="bidFeedback()">PUT feedback</button>
      <button onclick="bidReviews()">GET reviews</button>
    </section>
  </div>

  <h3>Response</h3>
  <pre id="out">Ready.</pre>

  <script>
    async function req(method, path, body) {
      const headers = { "Content-Type": "application/json" };
      const res = await fetch(path, { method, headers, body: body ? JSON.stringify(body) : undefined });
      const text = await res.text();
      let parsed = text;
      try { parsed = JSON.parse(text); } catch (_) {}
      document.getElementById("out").textContent = JSON.stringify({
        status: res.status,
        request_id: res.headers.get("X-Request-ID"),
        body: parsed
      }, null, 2);
      return { res, parsed };
    }

    const v = (id) => document.getElementById(id).value.trim();

    function ping() { return req("GET", "/api/ping"); }
    function createTender() {
      return req("POST", "/api/tenders/new", {
        name: v("tName"),
        description: v("tDesc"),
        serviceType: v("tService"),
        status: "Created",
        organizationId: Number(v("tOrg")),
        creatorUsername: v("tUser")
      });
    }
    function listTenders() { return req("GET", "/api/tenders"); }
    function myTenders() { return req("GET", "/api/tenders/my?username=" + encodeURIComponent(v("tUser"))); }
    function tenderStatusGet() { return req("GET", "/api/tenders/" + v("taTenderId") + "/status?username=" + encodeURIComponent(v("taUser"))); }
    function tenderStatusPut() { return req("PUT", "/api/tenders/" + v("taTenderId") + "/status?status=" + encodeURIComponent(v("taStatus")) + "&username=" + encodeURIComponent(v("taUser"))); }
    function tenderEdit() {
      return req("PATCH", "/api/tenders/" + v("taTenderId") + "/edit?username=" + encodeURIComponent(v("taUser")), {
        name: v("taEditName"),
        description: "Edited via UI",
        serviceType: "Delivery"
      });
    }
    function tenderRollback() { return req("PUT", "/api/tenders/" + v("taTenderId") + "/rollback/" + encodeURIComponent(v("taVersion")) + "?username=" + encodeURIComponent(v("taUser"))); }

    function createBid() {
      return req("POST", "/api/bids/new", {
        name: v("bName"),
        description: v("bDesc"),
        status: "Created",
        tenderId: v("bTenderId"),
        organizationId: Number(v("bOrgId")),
        creatorUsername: v("bUser")
      });
    }
    function myBids() { return req("GET", "/api/bids/my?username=" + encodeURIComponent(v("bUser"))); }
    function bidsForTender() { return req("GET", "/api/bids/" + v("bTenderId") + "/list?username=" + encodeURIComponent(v("baUser"))); }
    function bidStatusGet() { return req("GET", "/api/bids/" + v("baBidId") + "/status?username=" + encodeURIComponent(v("baUser"))); }
    function bidStatusPut() { return req("PUT", "/api/bids/" + v("baBidId") + "/status?status=" + encodeURIComponent(v("baStatus")) + "&username=" + encodeURIComponent(v("baUser"))); }
    function bidEdit() {
      return req("PATCH", "/api/bids/" + v("baBidId") + "/edit?username=" + encodeURIComponent(v("baUser")), {
        name: "Edited Bid",
        description: "Edited via UI"
      });
    }
    function bidRollback() { return req("PUT", "/api/bids/" + v("baBidId") + "/rollback/1?username=" + encodeURIComponent(v("baUser"))); }
    function bidDecision() { return req("PUT", "/api/bids/" + v("baBidId") + "/submit_decision?decision=" + encodeURIComponent(v("baDecision")) + "&username=" + encodeURIComponent(v("baUser"))); }
    function bidFeedback() { return req("PUT", "/api/bids/" + v("baBidId") + "/feedback?bidFeedback=" + encodeURIComponent(v("baFeedback")) + "&username=" + encodeURIComponent(v("baUser"))); }
    function bidReviews() { return req("GET", "/api/bids/" + v("baTenderId") + "/reviews?authorUsername=" + encodeURIComponent(v("baAuthor")) + "&requesterUsername=" + encodeURIComponent(v("baUser"))); }
  </script>
</body>
</html>`
