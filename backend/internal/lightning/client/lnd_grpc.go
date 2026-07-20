package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"pamojabuild1/backend/internal/lightning"
)

type LNDGRPCConfig struct {
	Host         string
	MacaroonPath string
	MacaroonHex  string
	TLSCertPath  string
}

type LNDGRPCClient struct {
	client lndLightningClient
	conn   *grpc.ClientConn
}

type lndRPCClient struct {
	client lnrpc.LightningClient
}

type lndLightningClient interface {
	AddInvoice(ctx context.Context, in *lnrpc.Invoice, opts ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error)
	SubscribeInvoices(ctx context.Context, in *lnrpc.InvoiceSubscription, opts ...grpc.CallOption) (lndInvoiceStream, error)
}

type lndInvoiceStream interface {
	Recv() (*lnrpc.Invoice, error)
}

func NewLNDGRPCClient(config LNDGRPCConfig) (*LNDGRPCClient, error) {
	host := strings.TrimSpace(config.Host)
	if host == "" {
		return nil, errors.New("lnd grpc host is required")
	}

	tlsCredentials, err := newGRPCTLSCredentials(config.TLSCertPath)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(
		host,
		grpc.WithTransportCredentials(tlsCredentials),
		grpc.WithPerRPCCredentials(lndMacaroonCredential{
			macaroonPath: strings.TrimSpace(config.MacaroonPath),
			macaroonHex:  strings.TrimSpace(config.MacaroonHex),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to lnd grpc: %w", err)
	}

	return &LNDGRPCClient{
		client: lndRPCClient{client: lnrpc.NewLightningClient(conn)},
		conn:   conn,
	}, nil
}

func (c lndRPCClient) AddInvoice(ctx context.Context, in *lnrpc.Invoice, opts ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	return c.client.AddInvoice(ctx, in, opts...)
}

func (c lndRPCClient) SubscribeInvoices(ctx context.Context, in *lnrpc.InvoiceSubscription, opts ...grpc.CallOption) (lndInvoiceStream, error) {
	return c.client.SubscribeInvoices(ctx, in, opts...)
}

func newLNDGRPCClientForTest(client lndLightningClient) *LNDGRPCClient {
	return &LNDGRPCClient{client: client}
}

func (c *LNDGRPCClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *LNDGRPCClient) CreateInvoice(ctx context.Context, request lightning.InvoiceRequest) (*lightning.Invoice, error) {
	if request.TaskSlug == "" {
		return nil, errors.New("task slug is required")
	}
	if request.AmountSats <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	expiry := request.Expiry
	if expiry <= 0 {
		expiry = time.Hour
	}

	response, err := c.client.AddInvoice(ctx, &lnrpc.Invoice{
		Memo:   request.Memo,
		Value:  request.AmountSats,
		Expiry: int64(expiry.Seconds()),
	})
	if err != nil {
		return nil, fmt.Errorf("lnd add invoice: %w", err)
	}
	if response.GetPaymentRequest() == "" {
		return nil, errors.New("lnd invoice response missing payment request")
	}

	paymentHash, err := paymentHashFromBytes(response.GetRHash())
	if err != nil {
		return nil, fmt.Errorf("decode lnd payment hash: %w", err)
	}
	addIndex, err := uint64ToInt64(response.GetAddIndex(), "add index")
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &lightning.Invoice{
		PaymentRequest: response.GetPaymentRequest(),
		PaymentHash:    paymentHash,
		AmountSats:     request.AmountSats,
		TaskSlug:       request.TaskSlug,
		Status:         lightning.InvoiceStatusPending,
		CreatedAt:      now,
		ExpiresAt:      now.Add(expiry),
		AddIndex:       addIndex,
	}, nil
}

func (c *LNDGRPCClient) SubscribeInvoiceSettlements(ctx context.Context, sinceSettleIndex int64, handler lightning.SettlementHandler) error {
	if handler == nil {
		return errors.New("settlement handler is required")
	}
	if sinceSettleIndex < 0 {
		return errors.New("settle index cannot be negative")
	}

	stream, err := c.client.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{
		SettleIndex: uint64(sinceSettleIndex),
	})
	if err != nil {
		return fmt.Errorf("subscribe to lnd invoices: %w", err)
	}

	for {
		invoice, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("receive lnd invoice update: %w", err)
		}
		if invoice.GetState() != lnrpc.Invoice_SETTLED {
			continue
		}

		domainInvoice, err := grpcInvoiceToDomain(invoice)
		if err != nil {
			return err
		}
		if err := handler(ctx, domainInvoice); err != nil {
			return err
		}
	}
}

func grpcInvoiceToDomain(invoice *lnrpc.Invoice) (*lightning.Invoice, error) {
	paymentHash, err := paymentHashFromBytes(invoice.GetRHash())
	if err != nil {
		return nil, fmt.Errorf("decode lnd settlement payment hash: %w", err)
	}
	addIndex, err := uint64ToInt64(invoice.GetAddIndex(), "add index")
	if err != nil {
		return nil, err
	}
	settleIndex, err := uint64ToInt64(invoice.GetSettleIndex(), "settle index")
	if err != nil {
		return nil, err
	}

	amountSats := invoice.GetAmtPaidSat()
	if amountSats == 0 {
		amountSats = invoice.GetValue()
	}

	settledAt := time.Now().UTC()
	if invoice.GetSettleDate() > 0 {
		settledAt = time.Unix(invoice.GetSettleDate(), 0).UTC()
	}

	return &lightning.Invoice{
		PaymentRequest: invoice.GetPaymentRequest(),
		PaymentHash:    paymentHash,
		AmountSats:     amountSats,
		Status:         lightning.InvoiceStatusSettled,
		Settled:        true,
		SettledAt:      settledAt,
		AddIndex:       addIndex,
		SettleIndex:    settleIndex,
	}, nil
}

func paymentHashFromBytes(paymentHash []byte) (string, error) {
	if len(paymentHash) != lndPaymentHashBytes {
		return "", fmt.Errorf("payment hash must be %d bytes", lndPaymentHashBytes)
	}
	return hex.EncodeToString(paymentHash), nil
}

func uint64ToInt64(value uint64, field string) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("lnd %s overflows int64", field)
	}
	return int64(value), nil
}

func newGRPCTLSCredentials(tlsCertPath string) (credentials.TransportCredentials, error) {
	// LND commonly uses a node-generated TLS certificate. When a cert path is
	// provided we trust only that certificate pool for the gRPC connection,
	// which avoids silently accepting an unexpected node in production.
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}

	if tlsCertPath != "" {
		certBytes, err := os.ReadFile(tlsCertPath)
		if err != nil {
			return nil, fmt.Errorf("read lnd tls cert: %w", err)
		}
		roots := x509.NewCertPool()
		if ok := roots.AppendCertsFromPEM(certBytes); !ok {
			return nil, errors.New("lnd tls cert file did not contain a valid PEM certificate")
		}
		tlsConfig.RootCAs = roots
	}

	return credentials.NewTLS(tlsConfig), nil
}

type lndMacaroonCredential struct {
	macaroonPath string
	macaroonHex  string
}

func (c lndMacaroonCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	macaroonHex, err := c.macaroon()
	if err != nil {
		return nil, err
	}
	return map[string]string{"macaroon": macaroonHex}, nil
}

func (c lndMacaroonCredential) RequireTransportSecurity() bool {
	return true
}

func (c lndMacaroonCredential) macaroon() (string, error) {
	if c.macaroonHex != "" {
		if _, err := hex.DecodeString(c.macaroonHex); err != nil {
			return "", fmt.Errorf("lnd macaroon hex is invalid: %w", err)
		}
		return c.macaroonHex, nil
	}
	if c.macaroonPath == "" {
		return "", errors.New("lnd macaroon path or hex value is required")
	}

	macaroonBytes, err := os.ReadFile(c.macaroonPath)
	if err != nil {
		return "", fmt.Errorf("read lnd macaroon: %w", err)
	}
	if len(macaroonBytes) == 0 {
		return "", errors.New("lnd macaroon file is empty")
	}
	return hex.EncodeToString(macaroonBytes), nil
}
