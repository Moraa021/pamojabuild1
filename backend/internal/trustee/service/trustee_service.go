package service

import (
	"context"
	standardecdsa "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/trustee"
)

var (
	ErrInvalidTrusteeIndex = errors.New("trustee index must be between 0 and 4")
	ErrInvalidTrustee      = errors.New("invalid trustee onboarding request")
	ErrInvalidTrusteeKeys  = errors.New("invalid trustee keys or ownership proof")
	ErrTrusteeConflict     = errors.New("trustee onboarding conflicts with an existing task relationship")
	ErrTrusteeState        = errors.New("trustee action is not allowed in the current state")
	ErrTrusteeForbidden    = errors.New("trustee action is not authorized")
)

type TrusteeService struct {
	repo     trustee.Repository
	eventBus *events.EventBus
	network  *chaincfg.Params
}

func NewTrusteeService(repo trustee.Repository, eventBus *events.EventBus, bitcoinNetwork string) *TrusteeService {
	return &TrusteeService{repo: repo, eventBus: eventBus, network: networkParams(bitcoinNetwork)}
}

func (s *TrusteeService) Nominate(ctx context.Context, taskSlug string, actorUserID int64, trusteeIndex int32, userID int64, message string) (*trustee.Assignment, error) {
	taskSlug, message = strings.TrimSpace(taskSlug), strings.TrimSpace(message)
	if trusteeIndex < 0 || trusteeIndex > 4 {
		return nil, ErrInvalidTrusteeIndex
	}
	if taskSlug == "" || len(taskSlug) > 255 || actorUserID <= 0 || userID <= 0 || actorUserID == userID || len(message) > 500 {
		return nil, ErrInvalidTrustee
	}
	a := &trustee.Assignment{TaskSlug: taskSlug, TrusteeIndex: trusteeIndex, UserID: userID, Status: trustee.StatusInvited, NominatedBy: actorUserID, NominationMessage: message}
	if err := s.repo.Nominate(ctx, a); err != nil {
		return nil, mapRepositoryError("nominate trustee", err)
	}
	return s.repo.GetAssignmentForUser(ctx, taskSlug, userID)
}

func (s *TrusteeService) Accept(ctx context.Context, taskSlug string, actorUserID int64) (*trustee.Assignment, error) {
	taskSlug = strings.TrimSpace(taskSlug)
	if taskSlug == "" || actorUserID <= 0 {
		return nil, ErrInvalidTrustee
	}
	challenge, err := newChallenge()
	if err != nil {
		return nil, fmt.Errorf("generate trustee proof challenge: %w", err)
	}
	a, err := s.repo.Accept(ctx, taskSlug, actorUserID, challenge)
	if err != nil {
		return nil, mapRepositoryError("accept trustee nomination", err)
	}
	return a, nil
}

func (s *TrusteeService) RegisterKeys(ctx context.Context, taskSlug string, actorUserID int64, registration *trustee.KeyRegistration) (*trustee.Assignment, error) {
	if registration == nil {
		return nil, ErrInvalidTrusteeKeys
	}
	registration.TaskSlug = strings.TrimSpace(taskSlug)
	registration.UserID = actorUserID
	if err := s.validateProofs(registration); err != nil {
		return nil, err
	}
	a, err := s.repo.Activate(ctx, registration)
	if err != nil {
		return nil, mapRepositoryError("activate trustee", err)
	}
	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{Type: events.TrusteeRegistered, Payload: map[string]interface{}{"task_slug": taskSlug, "trustee_index": a.TrusteeIndex}})
	}
	return a, nil
}

func (s *TrusteeService) PrepareKeyRotation(ctx context.Context, taskSlug string, actorUserID int64) (string, error) {
	if strings.TrimSpace(taskSlug) == "" || actorUserID <= 0 {
		return "", ErrInvalidTrustee
	}
	challenge, err := newChallenge()
	if err != nil {
		return "", fmt.Errorf("generate rotation challenge: %w", err)
	}
	if err := s.repo.IssueRotationChallenge(ctx, taskSlug, actorUserID, challenge); err != nil {
		return "", mapRepositoryError("prepare trustee key rotation", err)
	}
	return challenge, nil
}

func (s *TrusteeService) RotateKeys(ctx context.Context, taskSlug string, actorUserID int64, registration *trustee.KeyRegistration) error {
	if registration == nil {
		return ErrInvalidTrusteeKeys
	}
	registration.TaskSlug = strings.TrimSpace(taskSlug)
	registration.UserID = actorUserID
	registration.RotationReason = strings.TrimSpace(registration.RotationReason)
	if registration.RotationReason == "" || len(registration.RotationReason) > 500 {
		return ErrInvalidTrustee
	}
	if err := s.validateProofs(registration); err != nil {
		return err
	}
	if err := s.repo.RotateKeys(ctx, registration, actorUserID); err != nil {
		return mapRepositoryError("rotate trustee keys", err)
	}
	return nil
}

func (s *TrusteeService) Replace(ctx context.Context, taskSlug string, actorUserID int64, trusteeIndex int32, newUserID int64, reason, message string) error {
	reason, message = strings.TrimSpace(reason), strings.TrimSpace(message)
	if trusteeIndex < 0 || trusteeIndex > 4 {
		return ErrInvalidTrusteeIndex
	}
	if strings.TrimSpace(taskSlug) == "" || actorUserID <= 0 || newUserID <= 0 || reason == "" || len(reason) > 500 || len(message) > 500 {
		return ErrInvalidTrustee
	}
	r := &trustee.Replacement{TaskSlug: taskSlug, TrusteeIndex: trusteeIndex, NewUserID: newUserID, ActorUserID: actorUserID, Reason: reason, Message: message}
	if err := s.repo.Replace(ctx, r); err != nil {
		return mapRepositoryError("replace trustee", err)
	}
	return nil
}

func (s *TrusteeService) GetTaskTrustees(ctx context.Context, taskSlug string) ([]trustee.Assignment, error) {
	if strings.TrimSpace(taskSlug) == "" {
		return nil, ErrInvalidTrustee
	}
	return s.repo.GetRoster(ctx, taskSlug)
}

func (s *TrusteeService) validateProofs(r *trustee.KeyRegistration) error {
	r.Xpub = strings.TrimSpace(r.Xpub)
	r.WebCryptoPubkeyHex = strings.TrimSpace(r.WebCryptoPubkeyHex)
	r.XpubProofSignatureHex = strings.TrimSpace(r.XpubProofSignatureHex)
	r.WebCryptoProofSignatureHex = strings.TrimSpace(r.WebCryptoProofSignatureHex)
	r.ProofChallenge = strings.TrimSpace(r.ProofChallenge)
	if s.network == nil || r.TaskSlug == "" || r.UserID <= 0 || r.Xpub == "" || len(r.Xpub) > 255 || r.WebCryptoPubkeyHex == "" || len(r.WebCryptoPubkeyHex) > 512 || r.ProofChallenge == "" {
		return ErrInvalidTrusteeKeys
	}
	message := proofMessage(r)
	if !verifyXpubProof(r.Xpub, s.network, message, r.XpubProofSignatureHex) || !verifyWebCryptoProof(r.WebCryptoPubkeyHex, message, r.WebCryptoProofSignatureHex) {
		return ErrInvalidTrusteeKeys
	}
	return nil
}

// proofMessage binds both keys, the authenticated account, and the task to one
// server-issued nonce. A proof copied from another task or key pair cannot be
// reused to activate this assignment.
func proofMessage(r *trustee.KeyRegistration) []byte {
	return []byte(fmt.Sprintf("pamojabuild:trustee-key-proof:v1:%s:%d:%s:%s:%s", r.TaskSlug, r.UserID, r.ProofChallenge, r.Xpub, r.WebCryptoPubkeyHex))
}

func verifyXpubProof(value string, network *chaincfg.Params, message []byte, signatureHex string) bool {
	key, err := hdkeychain.NewKeyFromString(value)
	if err != nil || key.IsPrivate() || !key.IsForNet(network) {
		return false
	}
	child, err := key.Derive(0)
	if err != nil {
		return false
	}
	child, err = child.Derive(0)
	if err != nil {
		return false
	}
	pub, err := child.ECPubKey()
	if err != nil {
		return false
	}
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	sig, err := btcecdsa.ParseDERSignature(sigBytes)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(message)
	return sig.Verify(digest[:], pub)
}

func verifyWebCryptoProof(publicHex string, message []byte, signatureHex string) bool {
	publicBytes, err := hex.DecodeString(publicHex)
	if err != nil {
		return false
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), publicBytes)
	if x == nil {
		return false
	}
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(message)
	return standardecdsa.VerifyASN1(&standardecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, digest[:], sig)
}

func newChallenge() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func networkParams(value string) *chaincfg.Params {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "mainnet", "main":
		return &chaincfg.MainNetParams
	case "regtest", "regression":
		return &chaincfg.RegressionNetParams
	case "signet":
		return &chaincfg.SigNetParams
	case "", "testnet", "testnet3":
		return &chaincfg.TestNet3Params
	default:
		// An unknown deployment value must not silently reinterpret production
		// xpubs as another network. Key activation fails closed instead.
		return nil
	}
}

func mapRepositoryError(action string, err error) error {
	switch {
	case errors.Is(err, trustee.ErrNotTaskCreator):
		return ErrTrusteeForbidden
	case errors.Is(err, trustee.ErrInvalidState), errors.Is(err, trustee.ErrAssignmentNotFound):
		return ErrTrusteeState
	case errors.Is(err, trustee.ErrRegistrationConflict):
		return ErrTrusteeConflict
	case errors.Is(err, trustee.ErrTaskNotFound):
		return fmt.Errorf("%s: %w", action, trustee.ErrTaskNotFound)
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}
