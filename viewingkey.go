package mersennet

// Viewing-key helpers and owner viewing material (Workstream F2).
// Parity with sdk-ts ViewingKeyHelpers / createOwnerViewingMaterial.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// ViewingKey is a shielded viewing/spend keypair.
type ViewingKey struct {
	SpendPK string
	SpendSK string
	ViewPK  string
	ViewSK  string
}

var two256 = new(big.Int).Lsh(big.NewInt(1), 256)

// simpleHash is a deterministic placeholder hash matching the TS simpleHash
// (rolling polynomial over UTF-8 code points mod 2^256). Reproducible for
// tests only; production wallets use a hardened BIP-32/keccak derivation.
func simpleHash(value string) string {
	h := new(big.Int)
	for _, ch := range value {
		// h = (h*31 + ch) mod 2^256   (matches (h<<5)-h + ch)
		h.Mul(h, big.NewInt(31))
		h.Add(h, big.NewInt(int64(ch)))
		h.Mod(h, two256)
	}
	hexStr := fmt.Sprintf("%064x", h)
	return "0x" + hexStr[:64]
}

// ViewingKeyFromSeed derives a viewing key from a seed (parity with
// ViewingKeyHelpers.fromSeed). Deterministic placeholder; do not use for real funds.
func ViewingKeyFromSeed(seed string) ViewingKey {
	h := simpleHash(seed + ":spend")
	v := simpleHash(seed + ":view")
	return ViewingKey{
		SpendSK: "0x" + h[2:],
		SpendPK: "0x" + simpleHash(h + ":pub")[2:],
		ViewSK:  "0x" + v[2:],
		ViewPK:  "0x" + simpleHash(v + ":pub")[2:],
	}
}

// DelegateViewToken encodes a one-shot viewing token an auditor/counterparty
// can submit for selective disclosure. Base64-encoded JSON, parity with TS.
func DelegateViewToken(vk ViewingKey, scope []string, expiry int64) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"vk":     vk.ViewSK,
		"scope":  scope,
		"expiry": expiry,
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(payload), nil
}

// CreateOwnerViewingMaterial builds the viewing material a wallet uses to scan
// its OWN notes. The decryptor is keyed by the wallet's secret viewing scalar;
// the scan is filtered to notes addressed to its public viewing key.
func CreateOwnerViewingMaterial(vk ViewingKey, grantIDHex string) GrantedViewingMaterial {
	return GrantedViewingMaterial{
		GrantID:            grantIDHex,
		RecipientPublicKey: vk.ViewPK,
		Decrypt:            NewMockNoteDecryptor(vk.ViewSK),
	}
}
