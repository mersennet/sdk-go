package mersennet

// Compliance / view-key attestation (selective disclosure, ADR-019).
//
// A grantee holding a scoped viewing grant reconstructs a grantor's shielded
// portfolio locally and produces a portable, tamper-evident attestation: a
// compact document asserting the balances observed as of a specific block and
// shielded state root, bound to the grant. Cross-language design shared with
// the TypeScript and Python SDKs.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
)

// AttestationVersion is the canonical attestation version string.
const AttestationVersion = "mersennet-attestation-v1"

// Signer signs a digest and returns a signature (both 0x-hex).
type Signer func(digestHex string) (string, error)

// Verifier verifies a signature over a digest by the named attester.
type Verifier func(digestHex, signatureHex, attester string) bool

// ComplianceAttestation is the portable disclosure artifact.
type ComplianceAttestation struct {
	Version               string            `json:"version"`
	GrantID               string            `json:"grantId"`
	GrantorCommitment     string            `json:"grantorCommitment"`
	BlockNumber           uint64            `json:"blockNumber"`
	ShieldedStateRoot     string            `json:"shieldedStateRoot"`
	PerAsset              map[string]string `json:"perAsset"` // assetId -> balance, decimal strings
	UnspentNoteCount      int               `json:"unspentNoteCount"`
	SpentNullifierCount   int               `json:"spentNullifierCount"`
	PortfolioDigest       string            `json:"portfolioDigest"`
	IssuedAt              int64             `json:"issuedAt"`
	Scope                 string            `json:"scope"`
	OnchainPortfolioDigst *string           `json:"onchainPortfolioDigest"`
	DigestMatchesOnchain  *bool             `json:"digestMatchesOnchain"`
	Attester              string            `json:"attester,omitempty"`
	Signature             string            `json:"signature,omitempty"`
}

func canonicalPreimage(grantID, grantorCommitment string, blockNumber uint64, shieldedStateRoot string, perAsset map[uint32]*big.Int, unspent, spentNullifiers int) string {
	ids := make([]int, 0, len(perAsset))
	for id := range perAsset {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d:%s", id, perAsset[uint32(id)].String()))
	}
	return strings.Join([]string{
		AttestationVersion,
		strings.ToLower(grantID),
		strings.ToLower(grantorCommitment),
		fmt.Sprintf("%d", blockNumber),
		strings.ToLower(shieldedStateRoot),
		strings.Join(parts, "|"),
		fmt.Sprintf("%d", unspent),
		fmt.Sprintf("%d", spentNullifiers),
	}, "\n")
}

// ComputePortfolioDigest returns the deterministic 0x-hex SHA-256 digest over
// the disclosed facts. Identical across the TS/Python/Go SDKs.
func ComputePortfolioDigest(grantID, grantorCommitment string, blockNumber uint64, shieldedStateRoot string, perAsset map[uint32]*big.Int, unspent, spentNullifiers int) string {
	sum := sha256.Sum256([]byte(canonicalPreimage(grantID, grantorCommitment, blockNumber, shieldedStateRoot, perAsset, unspent, spentNullifiers)))
	return "0x" + hex.EncodeToString(sum[:])
}

// BuildAttestationOptions configures BuildPortfolioAttestation.
type BuildAttestationOptions struct {
	GrantorCommitment     string
	Scope                 string
	OnchainPortfolioDigst string
	Attester              string
	Sign                  Signer
	IssuedAt              int64 // 0 => now
}

// BuildPortfolioAttestation builds a compliance attestation from a
// reconstructed balance result.
func BuildPortfolioAttestation(result *BalanceReconstructionResult, opts BuildAttestationOptions) (*ComplianceAttestation, error) {
	scope := opts.Scope
	if scope == "" {
		scope = "balances:read"
	}
	grantorCommitment := opts.GrantorCommitment
	if grantorCommitment == "" {
		grantorCommitment = "0x"
	}

	digest := ComputePortfolioDigest(
		result.GrantID, grantorCommitment, result.BlockNumber, result.ShieldedStateRoot,
		result.PerAsset, result.UnspentNoteCount, result.SpentNullifierCount,
	)

	perAsset := make(map[string]string, len(result.PerAsset))
	for id, bal := range result.PerAsset {
		perAsset[fmt.Sprintf("%d", id)] = bal.String()
	}

	att := &ComplianceAttestation{
		Version:             AttestationVersion,
		GrantID:             result.GrantID,
		GrantorCommitment:   grantorCommitment,
		BlockNumber:         result.BlockNumber,
		ShieldedStateRoot:   result.ShieldedStateRoot,
		PerAsset:            perAsset,
		UnspentNoteCount:    result.UnspentNoteCount,
		SpentNullifierCount: result.SpentNullifierCount,
		PortfolioDigest:     digest,
		Scope:               scope,
		IssuedAt:            opts.IssuedAt,
	}
	if att.IssuedAt == 0 {
		att.IssuedAt = time.Now().Unix()
	}
	if opts.OnchainPortfolioDigst != "" {
		att.OnchainPortfolioDigst = &opts.OnchainPortfolioDigst
		match := strings.EqualFold(opts.OnchainPortfolioDigst, digest)
		att.DigestMatchesOnchain = &match
	}
	if opts.Sign != nil {
		sig, err := opts.Sign(digest)
		if err != nil {
			return nil, err
		}
		att.Signature = sig
		att.Attester = opts.Attester
	} else if opts.Attester != "" {
		att.Attester = opts.Attester
	}
	return att, nil
}

// perAssetToBig reparses the attestation's decimal-string balances for
// digest recomputation.
func (a *ComplianceAttestation) perAssetToBig() (map[uint32]*big.Int, error) {
	out := make(map[uint32]*big.Int, len(a.PerAsset))
	for k, v := range a.PerAsset {
		var id uint32
		if _, err := fmt.Sscanf(k, "%d", &id); err != nil {
			return nil, err
		}
		bal, ok := new(big.Int).SetString(v, 10)
		if !ok {
			return nil, fmt.Errorf("invalid balance %q", v)
		}
		out[id] = bal
	}
	return out, nil
}

// VerifyAttestation checks integrity (recompute digest from the attestation's
// own disclosed fields) and, if a Verifier is supplied and a signature is
// present, the signature. Returns (ok, reason).
func VerifyAttestation(a *ComplianceAttestation, verify Verifier) (bool, string) {
	perAsset, err := a.perAssetToBig()
	if err != nil {
		return false, "malformed perAsset: " + err.Error()
	}
	recomputed := ComputePortfolioDigest(
		a.GrantID, a.GrantorCommitment, a.BlockNumber, a.ShieldedStateRoot,
		perAsset, a.UnspentNoteCount, a.SpentNullifierCount,
	)
	if !strings.EqualFold(recomputed, a.PortfolioDigest) {
		return false, "digest mismatch: attestation fields do not hash to the embedded digest"
	}
	if a.DigestMatchesOnchain != nil && !*a.DigestMatchesOnchain {
		return false, "reconstructed digest did not match the on-chain portfolio digest"
	}
	if verify != nil {
		if a.Signature == "" {
			return false, "signature required but missing"
		}
		if !verify(a.PortfolioDigest, a.Signature, a.Attester) {
			return false, "signature verification failed"
		}
	}
	return true, "ok"
}
