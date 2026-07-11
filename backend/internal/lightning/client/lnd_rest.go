package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"pamojabuild1/backend/internal/lightning"
)

const maxLNDResponseBytes = 1 << 20

type LNDRESTConfig struct {
	BaseURL      string
	MacaroonPath string
	MacaroonHex  string
	TLSCertPath  string
	HTTPClient   *http.Client
}

type LNDRESTClient struct {
	baseURL      string
	macaroonPath string
	macaroonHex  string
	httpClient   *http.Client
}

func NewLNDRESTClient(config LNDRESTConfig) (*LNDRESTClient, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return nil, errors.New("lnd base url is required")
	}
	if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + baseURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse lnd base url: %w", err)
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("unsupported lnd url scheme %q", parsedURL.Scheme)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient, err = newHTTPClient(config.TLSCertPath)
		if err != nil {
			return nil, err
		}
	}

	return &LNDRESTClient{
		baseURL:      strings.TrimRight(parsedURL.String(), "/"),
		macaroonPath: strings.TrimSpace(config.MacaroonPath),
		macaroonHex:  strings.TrimSpace(config.MacaroonHex),
		httpClient:   httpClient,
	}, nil
}

func (c *LNDRESTClient) CreateInvoice(ctx context.Context, request lightning.InvoiceRequest) (*lightning.Invoice, error) {
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

	body := map[string]string{
		"value":  strconv.FormatInt(request.AmountSats, 10),
		"memo":   request.Memo,
		"expiry": strconv.FormatInt(int64(expiry.Seconds()), 10),
	}

	var response lndAddInvoiceResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/invoices", body, &response); err != nil {
		return nil, err
	}
	if response.PaymentRequest == "" {
		return nil, errors.New("lnd invoice response missing payment request")
	}

	paymentHash, err := decodePaymentHash(response.RHash, "")
	if err != nil {
		return nil, fmt.Errorf("decode lnd payment hash: %w", err)
	}

	now := time.Now().UTC()
	addIndex, err := parseOptionalInt64(response.AddIndex)
	if err != nil {
		return nil, fmt.Errorf("parse lnd add index: %w", err)
	}

	return &lightning.Invoice{
		PaymentRequest: response.PaymentRequest,
		PaymentHash:    paymentHash,
		AmountSats:     request.AmountSats,
		TaskSlug:       request.TaskSlug,
		Status:         lightning.InvoiceStatusPending,
		CreatedAt:      now,
		ExpiresAt:      now.Add(expiry),
		AddIndex:       addIndex,
	}, nil
}

func (c *LNDRESTClient) SubscribeInvoiceSettlements(ctx context.Context, sinceSettleIndex int64, handler lightning.SettlementHandler) error {
	if handler == nil {
		return errors.New("settlement handler is required")
	}

	endpoint := "/v1/invoices/subscribe"
	if sinceSettleIndex > 0 {
		endpoint += "?settle_index=" + strconv.FormatInt(sinceSettleIndex, 10)
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("subscribe to lnd invoice updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, maxLNDResponseBytes))
		return fmt.Errorf("lnd subscribe invoices returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var envelope lndInvoiceEnvelope
		if err := decoder.Decode(&envelope); err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("decode lnd invoice update: %w", err)
		}
		if envelope.Error != nil {
			return fmt.Errorf("lnd invoice stream error %d: %s", envelope.Error.Code, envelope.Error.Message)
		}
		if !strings.EqualFold(envelope.Result.State, "SETTLED") {
			continue
		}

		invoice, err := envelope.Result.toDomainInvoice()
		if err != nil {
			return err
		}
		if err := handler(ctx, invoice); err != nil {
			return err
		}
	}
}

func (c *LNDRESTClient) doJSON(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode lnd request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call lnd %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	limitedBody := io.LimitReader(resp.Body, maxLNDResponseBytes)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(limitedBody)
		return fmt.Errorf("lnd returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	if responseBody == nil {
		_, err = io.Copy(io.Discard, limitedBody)
		return err
	}
	if err := json.NewDecoder(limitedBody).Decode(responseBody); err != nil {
		return fmt.Errorf("decode lnd response: %w", err)
	}
	return nil
}

func (c *LNDRESTClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	macaroon, err := c.macaroon()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create lnd request: %w", err)
	}
	req.Header.Set("Grpc-Metadata-macaroon", macaroon)
	return req, nil
}

func (c *LNDRESTClient) macaroon() (string, error) {
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

func newHTTPClient(tlsCertPath string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}

	if tlsCertPath != "" {
		certBytes, err := os.ReadFile(tlsCertPath)
		if err != nil {
			return nil, fmt.Errorf("read lnd tls cert: %w", err)
		}
		roots := x509.NewCertPool()
		if ok := roots.AppendCertsFromPEM(certBytes); !ok {
			return nil, errors.New("lnd tls cert file did not contain a valid PEM certificate")
		}
		transport.TLSClientConfig.RootCAs = roots
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}

func decodePaymentHash(base64Hash string, hexHash string) (string, error) {
	if hexHash != "" {
		if _, err := hex.DecodeString(hexHash); err != nil {
			return "", err
		}
		return strings.ToLower(hexHash), nil
	}
	if base64Hash == "" {
		return "", errors.New("missing payment hash")
	}
	decoded, err := base64.StdEncoding.DecodeString(base64Hash)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(decoded), nil
}

func parseOptionalInt64(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

type lndAddInvoiceResponse struct {
	RHash          string `json:"r_hash"`
	PaymentRequest string `json:"payment_request"`
	AddIndex       string `json:"add_index"`
}

type lndInvoiceEnvelope struct {
	Result lndInvoice `json:"result"`
	Error  *lndError  `json:"error"`
}

type lndError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type lndInvoice struct {
	RHash          string `json:"r_hash"`
	RHashStr       string `json:"r_hash_str"`
	PaymentRequest string `json:"payment_request"`
	Value          string `json:"value"`
	AmtPaidSat     string `json:"amt_paid_sat"`
	State          string `json:"state"`
	SettleDate     string `json:"settle_date"`
	AddIndex       string `json:"add_index"`
	SettleIndex    string `json:"settle_index"`
}

func (i lndInvoice) toDomainInvoice() (*lightning.Invoice, error) {
	paymentHash, err := decodePaymentHash(i.RHash, i.RHashStr)
	if err != nil {
		return nil, fmt.Errorf("decode lnd settlement payment hash: %w", err)
	}

	amount := i.AmtPaidSat
	if amount == "" {
		amount = i.Value
	}
	amountSats, err := parseOptionalInt64(amount)
	if err != nil {
		return nil, fmt.Errorf("parse lnd settlement amount: %w", err)
	}
	settleIndex, err := parseOptionalInt64(i.SettleIndex)
	if err != nil {
		return nil, fmt.Errorf("parse lnd settle index: %w", err)
	}
	addIndex, err := parseOptionalInt64(i.AddIndex)
	if err != nil {
		return nil, fmt.Errorf("parse lnd add index: %w", err)
	}

	settledAt := time.Now().UTC()
	if i.SettleDate != "" {
		seconds, err := strconv.ParseInt(i.SettleDate, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse lnd settle date: %w", err)
		}
		settledAt = time.Unix(seconds, 0).UTC()
	}

	return &lightning.Invoice{
		PaymentRequest: i.PaymentRequest,
		PaymentHash:    paymentHash,
		AmountSats:     amountSats,
		Status:         lightning.InvoiceStatusSettled,
		Settled:        true,
		SettledAt:      settledAt,
		AddIndex:       addIndex,
		SettleIndex:    settleIndex,
	}, nil
}
