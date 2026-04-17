package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateCreateTenderRequest(t *testing.T) {
	t.Parallel()
	base := createTenderRequest{
		Name:            "Tender",
		Description:     strings.Repeat("a", 100),
		ServiceType:     "Delivery",
		Status:          "Created",
		OrganizationID:  1,
		CreatorUsername: "user1",
	}
	t.Run("ok full", func(t *testing.T) {
		t.Parallel()
		req := base
		if err := validateCreateTenderRequest(req); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ok empty status", func(t *testing.T) {
		t.Parallel()
		req := base
		req.Status = ""
		if err := validateCreateTenderRequest(req); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fail bad status", func(t *testing.T) {
		t.Parallel()
		req := base
		req.Status = "Published"
		if err := validateCreateTenderRequest(req); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("fail empty name", func(t *testing.T) {
		t.Parallel()
		req := base
		req.Name = "   "
		if err := validateCreateTenderRequest(req); err == nil || err.Error() != "invalid name" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("fail long name", func(t *testing.T) {
		t.Parallel()
		req := base
		req.Name = strings.Repeat("x", 101)
		if err := validateCreateTenderRequest(req); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("fail bad service type", func(t *testing.T) {
		t.Parallel()
		req := base
		req.ServiceType = "Invalid"
		if err := validateCreateTenderRequest(req); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("fail org id", func(t *testing.T) {
		t.Parallel()
		req := base
		req.OrganizationID = 0
		if err := validateCreateTenderRequest(req); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateEditTenderRequest(t *testing.T) {
	t.Parallel()
	t.Run("empty patch ok", func(t *testing.T) {
		t.Parallel()
		if err := validateEditTenderRequest(editTenderRequest{}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ok fields", func(t *testing.T) {
		t.Parallel()
		n := "Name"
		d := strings.Repeat("d", 400)
		st := "Construction"
		if err := validateEditTenderRequest(editTenderRequest{Name: &n, Description: &d, ServiceType: &st}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fail empty name", func(t *testing.T) {
		t.Parallel()
		n := "  "
		if err := validateEditTenderRequest(editTenderRequest{Name: &n}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateCreateBidRequest(t *testing.T) {
	t.Parallel()
	base := createBidRequest{
		Name:            "Bid",
		Description:     "desc",
		Status:          "Created",
		TenderID:        "550e8400-e29b-41d4-a716-446655440000",
		OrganizationID:  0,
		CreatorUsername: "user1",
	}
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		req := base
		if err := validateCreateBidRequest(req); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ok empty status", func(t *testing.T) {
		t.Parallel()
		req := base
		req.Status = ""
		if err := validateCreateBidRequest(req); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fail bad status", func(t *testing.T) {
		t.Parallel()
		req := base
		req.Status = "Approved"
		if err := validateCreateBidRequest(req); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("fail empty tender id", func(t *testing.T) {
		t.Parallel()
		req := base
		req.TenderID = ""
		if err := validateCreateBidRequest(req); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("fail negative org id", func(t *testing.T) {
		t.Parallel()
		req := base
		req.OrganizationID = -1
		if err := validateCreateBidRequest(req); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateEditBidRequest(t *testing.T) {
	t.Parallel()
	t.Run("empty ok", func(t *testing.T) {
		t.Parallel()
		if err := validateEditBidRequest(editBidRequest{}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		n := "N"
		d := "D"
		if err := validateEditBidRequest(editBidRequest{Name: &n, Description: &d}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestParsePagination(t *testing.T) {
	t.Parallel()
	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/api/tenders", nil)
		limit, offset, err := parsePagination(r)
		if err != nil {
			t.Fatal(err)
		}
		if limit != 5 || offset != 0 {
			t.Fatalf("got limit=%d offset=%d", limit, offset)
		}
	})
	t.Run("explicit", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=10&offset=3", nil)
		limit, offset, err := parsePagination(r)
		if err != nil {
			t.Fatal(err)
		}
		if limit != 10 || offset != 3 {
			t.Fatalf("got limit=%d offset=%d", limit, offset)
		}
	})
	t.Run("limit zero explicit", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=0", nil)
		limit, offset, err := parsePagination(r)
		if err != nil {
			t.Fatal(err)
		}
		if limit != 0 || offset != 0 {
			t.Fatalf("got limit=%d offset=%d", limit, offset)
		}
	})
	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?limit=51", nil)
		_, _, err := parsePagination(r)
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("invalid offset", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/?offset=-1", nil)
		_, _, err := parsePagination(r)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
