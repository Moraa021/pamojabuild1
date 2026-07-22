package http

import (
	"time"

	"pamojabuild1/backend/internal/trustee"
)

type NominateTrusteeRequest struct {
	TrusteeIndex int32  `json:"trustee_index" binding:"min=0,max=4"`
	UserID       int64  `json:"user_id" binding:"required,gt=0"`
	Message      string `json:"message" binding:"max=500"`
}

type RegisterTrusteeKeysRequest struct {
	Xpub                       string `json:"xpub" binding:"required,max=255"`
	WebCryptoPubkeyHex         string `json:"web_crypto_pubkey_hex" binding:"required,max=512"`
	XpubProofSignatureHex      string `json:"xpub_proof_signature_hex" binding:"required,max=160"`
	WebCryptoProofSignatureHex string `json:"web_crypto_proof_signature_hex" binding:"required,max=256"`
	ProofChallenge             string `json:"proof_challenge" binding:"required,len=64,hexadecimal"`
}

type RotateTrusteeKeysRequest struct {
	RegisterTrusteeKeysRequest
	Reason string `json:"reason" binding:"required,max=500"`
}

type ReplaceTrusteeRequest struct {
	UserID  int64  `json:"user_id" binding:"required,gt=0"`
	Reason  string `json:"reason" binding:"required,max=500"`
	Message string `json:"message" binding:"max=500"`
}

type AssignmentResponse struct {
	TaskSlug          string                   `json:"task_slug"`
	TrusteeIndex      int32                    `json:"trustee_index"`
	UserID            int64                    `json:"user_id"`
	DisplayName       string                   `json:"display_name"`
	Status            trustee.AssignmentStatus `json:"status"`
	NominationMessage string                   `json:"nomination_message"`
	ProofChallenge    string                   `json:"proof_challenge,omitempty"`
	InvitedAt         time.Time                `json:"invited_at"`
	AcceptedAt        *time.Time               `json:"accepted_at,omitempty"`
	ActivatedAt       *time.Time               `json:"activated_at,omitempty"`
}

type RosterResponse struct {
	Trustees []AssignmentResponse `json:"trustees"`
}
type ChallengeResponse struct {
	ProofChallenge string `json:"proof_challenge"`
}
