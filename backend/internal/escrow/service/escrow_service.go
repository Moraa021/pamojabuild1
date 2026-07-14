package service

import (
	"context"
	"encoding/hex"
	"errors"
	"pamojabuild1/internal/escrow"
	"pamojabuild1/internal/trustee"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

const signatureThreshold = 3

type Repository interface {
	SaveSignatureFragment(ctx context.Context, sig *escrow.SignatureCollection) error
	GetSignatureCount(ctx context.Context, taskSlug string) (int, error)
	GetAllSignatures(ctx context.Context, taskSlug string) ([]escrow.SignatureCollection, error)
	SavePayoutManifest(ctx context.Context, taskSlug string, unsignedPsbtHex string, volunteerInvoice string) error
	GetPayoutManifest(ctx context.Context, taskSlug string) (string, string, error)
}

type TrusteeKeyRepo interface {
	GetKeysByTask(ctx context.Context, taskSlug string) ([]trustee.TrusteeKey, error)
}

type EscrowService struct {
	Repo        Repository
	TrusteeRepo TrusteeKeyRepo
	NetParams   *chaincfg.Params
}

func NewEscrowService(repo Repository, trusteeRepo TrusteeKeyRepo, net *chaincfg.Params) *EscrowService {
	return &EscrowService{Repo: repo, TrusteeRepo: trusteeRepo, NetParams: net}
}

func (s *EscrowService) Derive3Of5MultiSigAddress(xpubs []string, index uint32) (string, error) {
	if len(xpubs) != 5 {
		return "", errors.New("exactly 5 xpubs required for 3-of-5 multisig")
	}
	var pubKeys []*btcec.PublicKey
	for _, xpub := range xpubs {
		b, err := hex.DecodeString(xpub)
		if err != nil {
			return "", errors.New("invalid xpub hex: " + err.Error())
		}
		pk, err := btcec.ParsePubKey(b)
		if err != nil {
			return "", errors.New("failed to parse public key: " + err.Error())
		}
		pubKeys = append(pubKeys, pk)
	}
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_3)
	for _, pk := range pubKeys {
		builder.AddData(pk.SerializeCompressed())
	}
	builder.AddOp(txscript.OP_5)
	builder.AddOp(txscript.OP_CHECKMULTISIG)
	redeemScript, err := builder.Script()
	if err != nil {
		return "", err
	}
	addr, err := btcutil.NewAddressWitnessScriptHash(
		btcutil.Hash160(redeemScript), s.NetParams,
	)
	if err != nil {
		return "", err
	}
	return addr.EncodeAddress(), nil
}

func (s *EscrowService) PreparePayoutManifest(ctx context.Context, taskSlug string, destinationAddress string, volunteerInvoice string) (*escrow.SignatureCollection, error) {
	msgTx := wire.NewMsgTx(wire.TxVersion)
	_ = msgTx
	unsignedPsbtHex := "unsigned_psbt_placeholder_" + taskSlug
	if err := s.Repo.SavePayoutManifest(ctx, taskSlug, unsignedPsbtHex, volunteerInvoice); err != nil {
		return nil, err
	}
	return &escrow.SignatureCollection{TaskSlug: taskSlug}, nil
}

func (s *EscrowService) SubmitTrusteeSignature(ctx context.Context, taskSlug string, payload *escrow.SignatureCollection) (bool, error) {
	if payload.TrusteePublicKeyHex == "" || payload.L1SignatureFragment == "" || payload.L2WebCryptoSignature == "" {
		return false, errors.New("all signature fields are required")
	}
	payload.TaskSlug = taskSlug
	if err := s.Repo.SaveSignatureFragment(ctx, payload); err != nil {
		return false, err
	}
	count, err := s.Repo.GetSignatureCount(ctx, taskSlug)
	if err != nil {
		return false, err
	}
	return count >= signatureThreshold, nil
}

func (s *EscrowService) FinalizeAndBroadcastPayout(ctx context.Context, taskSlug string) error {
	count, err := s.Repo.GetSignatureCount(ctx, taskSlug)
	if err != nil {
		return err
	}
	if count < signatureThreshold {
		return errors.New("not enough signatures: need 3")
	}
	sigs, err := s.Repo.GetAllSignatures(ctx, taskSlug)
	if err != nil {
		return err
	}
	_ = sigs
	return nil
}
