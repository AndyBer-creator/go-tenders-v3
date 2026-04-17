package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteErrorIncludesRequestID(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", "req-123")
	writeError(rec, 400, "bad request")
	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content-type: %s", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	for _, part := range []string{"reason", "bad request", "request_id", "req-123"} {
		if !strings.Contains(body, part) {
			t.Fatalf("body missing %q: %s", part, body)
		}
	}
}
