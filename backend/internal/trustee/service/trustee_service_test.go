package service

import (
	"context"
	standardecdsa "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"pamojabuild1/backend/internal/trustee"
)

type mockTrusteeRepo struct {
	assignment *trustee.Assignment
	nominated  *trustee.Assignment
	activated  *trustee.KeyRegistration
	rotated    *trustee.KeyRegistration
	replaced   *trustee.Replacement
	challenge  string
	err        error
}

func (m *mockTrusteeRepo) Nominate(_ context.Context, a *trustee.Assignment) error {
	m.nominated = a
	return m.err
}
func (m *mockTrusteeRepo) Accept(_ context.Context, slug string, userID int64, challenge string) (*trustee.Assignment, error) {
	m.challenge = challenge
	if m.err != nil {
		return nil, m.err
	}
	return &trustee.Assignment{TaskSlug: slug, UserID: userID, Status: trustee.StatusAccepted, ProofChallenge: challenge}, nil
}
func (m *mockTrusteeRepo) Activate(_ context.Context, r *trustee.KeyRegistration) (*trustee.Assignment, error) {
	m.activated = r
	if m.err != nil {
		return nil, m.err
	}
	return &trustee.Assignment{TaskSlug: r.TaskSlug, UserID: r.UserID, Status: trustee.StatusActive}, nil
}
func (m *mockTrusteeRepo) IssueRotationChallenge(_ context.Context, _ string, _ int64, c string) error {
	m.challenge = c
	return m.err
}
func (m *mockTrusteeRepo) RotateKeys(_ context.Context, r *trustee.KeyRegistration, _ int64) error {
	m.rotated = r
	return m.err
}
func (m *mockTrusteeRepo) Replace(_ context.Context, r *trustee.Replacement) error {
	m.replaced = r
	return m.err
}
func (m *mockTrusteeRepo) GetAssignmentForUser(_ context.Context, _ string, _ int64) (*trustee.Assignment, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.assignment != nil {
		return m.assignment, nil
	}
	return m.nominated, nil
}
func (m *mockTrusteeRepo) GetRoster(context.Context, string) ([]trustee.Assignment, error) {
	return nil, m.err
}

func TestNominationRejectsCreatorAsTrustee(t *testing.T) {
	svc := NewTrusteeService(&mockTrusteeRepo{}, nil, "testnet3")
	_, err := svc.Nominate(context.Background(), "task", 7, 0, 7, "")
	if !errors.Is(err, ErrInvalidTrustee) {
		t.Fatalf("expected invalid trustee, got %v", err)
	}
}
func TestNominationMapsCreatorAuthorization(t *testing.T) {
	repo := &mockTrusteeRepo{err: trustee.ErrNotTaskCreator}
	svc := NewTrusteeService(repo, nil, "testnet3")
	_, err := svc.Nominate(context.Background(), "task", 7, 0, 8, "")
	if !errors.Is(err, ErrTrusteeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
func TestAcceptCreatesOneTimeChallenge(t *testing.T) {
	repo := &mockTrusteeRepo{}
	svc := NewTrusteeService(repo, nil, "testnet3")
	a, err := svc.Accept(context.Background(), "task", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ProofChallenge) != 64 || a.ProofChallenge != repo.challenge {
		t.Fatalf("unexpected challenge %q", a.ProofChallenge)
	}
}

func TestRegisterKeysValidatesBothOwnershipProofs(t *testing.T) {
	repo := &mockTrusteeRepo{}
	svc := NewTrusteeService(repo, nil, "testnet3")
	r := validRegistration(t, "task", 8, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	a, err := svc.RegisterKeys(context.Background(), "task", 8, r)
	if err != nil {
		t.Fatalf("register valid keys: %v", err)
	}
	if a.Status != trustee.StatusActive || repo.activated == nil {
		t.Fatalf("expected active assignment")
	}
}

func TestRegisterKeysRejectsProofForDifferentTask(t *testing.T) {
	svc := NewTrusteeService(&mockTrusteeRepo{}, nil, "testnet3")
	r := validRegistration(t, "task-a", 8, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if _, err := svc.RegisterKeys(context.Background(), "task-b", 8, r); !errors.Is(err, ErrInvalidTrusteeKeys) {
		t.Fatalf("expected invalid proof, got %v", err)
	}
}

func TestPrepareRotationPersistsFreshChallenge(t *testing.T) {
	repo := &mockTrusteeRepo{}
	svc := NewTrusteeService(repo, nil, "testnet3")
	value, err := svc.PrepareKeyRotation(context.Background(), "task", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 64 || repo.challenge != value {
		t.Fatalf("unexpected challenge")
	}
}

func validRegistration(t *testing.T, taskSlug string, userID int64, challenge string) *trustee.KeyRegistration {
	t.Helper()
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		t.Fatal(err)
	}
	master, err := hdkeychain.NewMaster(seed, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	public, err := master.Neuter()
	if err != nil {
		t.Fatal(err)
	}
	webPrivate, err := standardecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	webPublic := elliptic.Marshal(elliptic.P256(), webPrivate.X, webPrivate.Y)
	r := &trustee.KeyRegistration{TaskSlug: taskSlug, UserID: userID, Xpub: public.String(), WebCryptoPubkeyHex: hex.EncodeToString(webPublic), ProofChallenge: challenge}
	message := proofMessage(r)
	digest := sha256.Sum256(message)
	child, err := master.Derive(0)
	if err != nil {
		t.Fatal(err)
	}
	child, err = child.Derive(0)
	if err != nil {
		t.Fatal(err)
	}
	private, err := child.ECPrivKey()
	if err != nil {
		t.Fatal(err)
	}
	// Parse through btcec to keep the test coupled to the exact public derivation
	// used by production rather than accepting a different signing key.
	parsedPrivate, _ := btcec.PrivKeyFromBytes(private.Serialize())
	r.XpubProofSignatureHex = hex.EncodeToString(btcecdsa.Sign(parsedPrivate, digest[:]).Serialize())
	webSig, err := standardecdsa.SignASN1(rand.Reader, webPrivate, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	r.WebCryptoProofSignatureHex = hex.EncodeToString(webSig)
	return r
}
