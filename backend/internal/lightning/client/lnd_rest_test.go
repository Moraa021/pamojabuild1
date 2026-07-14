package client

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pamojabuild1/backend/internal/lightning"
)

func TestLNDRESTClientCreateInvoice(t *testing.T) {
	rHashBytes := make([]byte, 32)
	for i := range rHashBytes {
		rHashBytes[i] = byte(i)
	}
	expectedPaymentHash := hex.EncodeToString(rHashBytes)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/invoices" {
			t.Fatalf("expected /v1/invoices path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Grpc-Metadata-macaroon"); got != "00ff" {
			t.Fatalf("expected macaroon header, got %q", got)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["value"] != "2500" {
			t.Fatalf("expected value 2500, got %q", body["value"])
		}
		if body["memo"] != "PamojaBuild donation task=clean-water" {
			t.Fatalf("expected task slug memo, got %q", body["memo"])
		}
		if body["expiry"] != "3600" {
			t.Fatalf("expected expiry 3600, got %q", body["expiry"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"r_hash":          base64.StdEncoding.EncodeToString(rHashBytes),
			"payment_request": "lnbcrt2500n1ptestinvoice",
			"add_index":       "42",
		}); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewLNDRESTClient(LNDRESTConfig{
		BaseURL:     server.URL,
		MacaroonHex: "00ff",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	invoice, err := client.CreateInvoice(context.Background(), lightning.InvoiceRequest{
		TaskSlug:   "clean-water",
		AmountSats: 2500,
		Memo:       "PamojaBuild donation task=clean-water",
		Expiry:     time.Hour,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if invoice.PaymentRequest != "lnbcrt2500n1ptestinvoice" {
		t.Fatalf("unexpected payment request %q", invoice.PaymentRequest)
	}
	if invoice.PaymentHash != expectedPaymentHash {
		t.Fatalf("expected payment hash %q, got %q", expectedPaymentHash, invoice.PaymentHash)
	}
	if invoice.TaskSlug != "clean-water" {
		t.Fatalf("expected task slug clean-water, got %q", invoice.TaskSlug)
	}
	if invoice.AmountSats != 2500 {
		t.Fatalf("expected amount 2500, got %d", invoice.AmountSats)
	}
	if invoice.AddIndex != 42 {
		t.Fatalf("expected add index 42, got %d", invoice.AddIndex)
	}
	if invoice.ExpiresAt.Sub(invoice.CreatedAt) != time.Hour {
		t.Fatalf("expected one hour expiry, got %s", invoice.ExpiresAt.Sub(invoice.CreatedAt))
	}
}

func TestLNDRESTClientCreateInvoiceRequiresMacaroon(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client, err := NewLNDRESTClient(LNDRESTConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	_, err = client.CreateInvoice(context.Background(), lightning.InvoiceRequest{
		TaskSlug:   "clean-water",
		AmountSats: 2500,
		Memo:       "PamojaBuild donation task=clean-water",
		Expiry:     time.Hour,
	})
	if err == nil {
		t.Fatal("expected missing macaroon error")
	}
}
