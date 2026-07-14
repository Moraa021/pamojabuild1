package client

import (
	"context"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"

	"pamojabuild1/backend/internal/lightning"
)

type mockLNDLightningClient struct {
	addInvoiceRequest          *lnrpc.Invoice
	addInvoiceResponse         *lnrpc.AddInvoiceResponse
	addInvoiceErr              error
	subscribeInvoiceRequest    *lnrpc.InvoiceSubscription
	subscribeInvoiceStream     lndInvoiceStream
	subscribeInvoiceStreamErr  error
	subscribeInvoiceCalledWith []grpc.CallOption
}

func (m *mockLNDLightningClient) AddInvoice(ctx context.Context, in *lnrpc.Invoice, opts ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	m.addInvoiceRequest = in
	if m.addInvoiceErr != nil {
		return nil, m.addInvoiceErr
	}
	return m.addInvoiceResponse, nil
}

func (m *mockLNDLightningClient) SubscribeInvoices(ctx context.Context, in *lnrpc.InvoiceSubscription, opts ...grpc.CallOption) (lndInvoiceStream, error) {
	m.subscribeInvoiceRequest = in
	m.subscribeInvoiceCalledWith = opts
	if m.subscribeInvoiceStreamErr != nil {
		return nil, m.subscribeInvoiceStreamErr
	}
	return m.subscribeInvoiceStream, nil
}

type mockLNDInvoiceStream struct {
	invoices []*lnrpc.Invoice
}

func (s *mockLNDInvoiceStream) Recv() (*lnrpc.Invoice, error) {
	if len(s.invoices) == 0 {
		return nil, io.EOF
	}
	invoice := s.invoices[0]
	s.invoices = s.invoices[1:]
	return invoice, nil
}

func TestLNDGRPCClientCreateInvoice(t *testing.T) {
	rHashBytes := make([]byte, 32)
	for i := range rHashBytes {
		rHashBytes[i] = byte(i)
	}
	expectedPaymentHash := hex.EncodeToString(rHashBytes)

	node := &mockLNDLightningClient{
		addInvoiceResponse: &lnrpc.AddInvoiceResponse{
			RHash:          rHashBytes,
			PaymentRequest: "lnbcrt2500n1ptestinvoice",
			AddIndex:       42,
		},
	}
	client := newLNDGRPCClientForTest(node)

	invoice, err := client.CreateInvoice(context.Background(), lightning.InvoiceRequest{
		TaskSlug:   "clean-water",
		AmountSats: 2500,
		Memo:       "PamojaBuild donation task=clean-water",
		Expiry:     time.Hour,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if node.addInvoiceRequest == nil {
		t.Fatal("expected AddInvoice request")
	}
	if node.addInvoiceRequest.GetValue() != 2500 {
		t.Fatalf("expected invoice value 2500, got %d", node.addInvoiceRequest.GetValue())
	}
	if node.addInvoiceRequest.GetMemo() != "PamojaBuild donation task=clean-water" {
		t.Fatalf("unexpected memo %q", node.addInvoiceRequest.GetMemo())
	}
	if node.addInvoiceRequest.GetExpiry() != 3600 {
		t.Fatalf("expected expiry 3600, got %d", node.addInvoiceRequest.GetExpiry())
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

func TestLNDGRPCClientCreateInvoiceRequiresPaymentRequest(t *testing.T) {
	node := &mockLNDLightningClient{
		addInvoiceResponse: &lnrpc.AddInvoiceResponse{
			RHash: make([]byte, 32),
		},
	}
	client := newLNDGRPCClientForTest(node)

	_, err := client.CreateInvoice(context.Background(), lightning.InvoiceRequest{
		TaskSlug:   "clean-water",
		AmountSats: 2500,
	})
	if err == nil {
		t.Fatal("expected missing payment request error")
	}
}

func TestLNDGRPCClientSubscribeInvoiceSettlements(t *testing.T) {
	openHash := make([]byte, 32)
	settledHash := make([]byte, 32)
	for i := range settledHash {
		settledHash[i] = byte(255 - i)
	}
	settleDate := time.Date(2026, 7, 11, 10, 30, 0, 0, time.UTC)

	stream := &mockLNDInvoiceStream{
		invoices: []*lnrpc.Invoice{
			{
				RHash:       openHash,
				State:       lnrpc.Invoice_OPEN,
				Value:       1000,
				AddIndex:    2,
				SettleIndex: 0,
			},
			{
				RHash:          settledHash,
				State:          lnrpc.Invoice_SETTLED,
				AmtPaidSat:     2500,
				PaymentRequest: "lnbcrt2500n1psettled",
				AddIndex:       3,
				SettleIndex:    9,
				SettleDate:     settleDate.Unix(),
			},
		},
	}
	node := &mockLNDLightningClient{subscribeInvoiceStream: stream}
	client := newLNDGRPCClientForTest(node)

	var handled []*lightning.Invoice
	err := client.SubscribeInvoiceSettlements(context.Background(), 7, func(ctx context.Context, invoice *lightning.Invoice) error {
		handled = append(handled, invoice)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if node.subscribeInvoiceRequest == nil {
		t.Fatal("expected SubscribeInvoices request")
	}
	if node.subscribeInvoiceRequest.GetSettleIndex() != 7 {
		t.Fatalf("expected settle index 7, got %d", node.subscribeInvoiceRequest.GetSettleIndex())
	}
	if len(handled) != 1 {
		t.Fatalf("expected one handled settled invoice, got %d", len(handled))
	}
	if handled[0].PaymentHash != hex.EncodeToString(settledHash) {
		t.Fatalf("unexpected settled payment hash %q", handled[0].PaymentHash)
	}
	if handled[0].AmountSats != 2500 {
		t.Fatalf("expected paid amount 2500, got %d", handled[0].AmountSats)
	}
	if handled[0].Status != lightning.InvoiceStatusSettled || !handled[0].Settled {
		t.Fatalf("expected settled invoice, got status=%q settled=%v", handled[0].Status, handled[0].Settled)
	}
	if !handled[0].SettledAt.Equal(settleDate) {
		t.Fatalf("expected settle date %s, got %s", settleDate, handled[0].SettledAt)
	}
	if handled[0].SettleIndex != 9 {
		t.Fatalf("expected settle index 9, got %d", handled[0].SettleIndex)
	}
}
