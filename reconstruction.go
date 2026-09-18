package mersennet

// Client-side portfolio reconstruction (ADR-019 `balances:read`).
//
// The privacy fork deliberately keeps the node from ever returning a decrypted
// per-account balance. Instead the owner - or a grantee holding a scoped
// viewing grant - decrypts the notes addressed to them (ScanGrantedNotes) and
// reconstructs spendable balances locally, here.
//
// Reconstruction rule: a note contributes to the balance only if its nullifier
// has NOT been published on chain. Faithful port of sdk-ts/src/reconstruction.ts.

import (
	"crypto/sha256"
	"math/big"
	"strings"
)

// NullifierDeriver maps an owned note to its on-chain nullifier (0x-prefixed hex).
type NullifierDeriver func(ShieldedNote) string

// PortfolioNote is a decrypted note tagged with its derived nullifier and spent flag.
type PortfolioNote struct {
	Note      ShieldedNote
	Nullifier string
	Spent     bool
}

// ReconstructedPortfolio holds spendable balances per asset (unspent notes only).
type ReconstructedPortfolio struct {
	PerAsset         map[uint32]*big.Int
	UnspentNoteCount int
	SpentNoteCount   int
	TotalNoteCount   int
	Notes            []PortfolioNote
}

// BalanceReconstructionResult is the end-to-end grant-gated balance read result.
type BalanceReconstructionResult struct {
	ReconstructedPortfolio
	GrantID                 string
	BlockNumber             uint64
	ShieldedStateRoot       string
	TotalEncryptedNoteCount int
	FetchedEncryptedNoteCnt int
	SkippedMalformedCount   int
	SpentNullifierCount     int
}

// ReconstructOptions configures reconstruction.
type ReconstructOptions struct {
	DeriveNullifier NullifierDeriver
	SpentNullifiers []string
	IsSpent         func(string) bool
}

func normalizeHex(v string) string {
	body := v
	if strings.HasPrefix(strings.ToLower(v), "0x") {
		body = v[2:]
	}
	return "0x" + strings.ToLower(body)
}

// DefaultNullifierDeriver mirrors the chain's mock scheme:
// sha256(ownerPk || rho || psi). Production wallets override this with their
// real nullifier-viewing-key derivation (chain scheme: Poseidon(spend_sk, rho)).
func DefaultNullifierDeriver(note ShieldedNote) string {
	h := sha256.New()
	h.Write(mustDecodeHexLenient(note.OwnerPK))
	h.Write(mustDecodeHexLenient(note.Rho))
	h.Write(mustDecodeHexLenient(note.Psi))
	return bytesToHex(h.Sum(nil))
}

// ReconstructPortfolio reconstructs spendable balances from decrypted notes.
// Notes are de-duplicated by nullifier; only unspent notes contribute.
func ReconstructPortfolio(notes []ShieldedNote, opts *ReconstructOptions) ReconstructedPortfolio {
	deriver := DefaultNullifierDeriver
	var spentPredicate func(string) bool = func(string) bool { return false }
	if opts != nil {
		if opts.DeriveNullifier != nil {
			deriver = opts.DeriveNullifier
		}
		if opts.IsSpent != nil {
			spentPredicate = opts.IsSpent
		} else if opts.SpentNullifiers != nil {
			spent := make(map[string]struct{}, len(opts.SpentNullifiers))
			for _, n := range opts.SpentNullifiers {
				spent[normalizeHex(n)] = struct{}{}
			}
			spentPredicate = func(nh string) bool {
				_, ok := spent[normalizeHex(nh)]
				return ok
			}
		}
	}

	perAsset := make(map[uint32]*big.Int)
	portfolioNotes := make([]PortfolioNote, 0, len(notes))
	seen := make(map[string]struct{})
	unspent, spentCount := 0, 0

	for _, note := range notes {
		nullifier := normalizeHex(deriver(note))
		if _, ok := seen[nullifier]; ok {
			continue
		}
		seen[nullifier] = struct{}{}

		isSpent := spentPredicate(nullifier)
		portfolioNotes = append(portfolioNotes, PortfolioNote{Note: note, Nullifier: nullifier, Spent: isSpent})
		if isSpent {
			spentCount++
			continue
		}
		unspent++
		if perAsset[note.AssetID] == nil {
			perAsset[note.AssetID] = new(big.Int)
		}
		perAsset[note.AssetID].Add(perAsset[note.AssetID], note.Value)
	}

	return ReconstructedPortfolio{
		PerAsset:         perAsset,
		UnspentNoteCount: unspent,
		SpentNoteCount:   spentCount,
		TotalNoteCount:   len(portfolioNotes),
		Notes:            portfolioNotes,
	}
}

// ScanAndReconstructBalances pages mersennet_viewBalances, decrypts owned
// notes, collects the spent-nullifier set, and reconstructs spendable
// per-asset balances locally (Workstream F2 + F5).
func ScanAndReconstructBalances(provider *Provider, material GrantedViewingMaterial, opts *GrantedNoteScanOptions, deriveNullifier NullifierDeriver) (*BalanceReconstructionResult, error) {
	ignoreMalformed := true
	var limitPtr *int
	var cursorPtr *string
	var maxPages int
	if opts != nil {
		ignoreMalformed = opts.IgnoreMalformed
		if opts.Limit > 0 {
			limitPtr = &opts.Limit
		}
		if opts.CursorHex != "" {
			cursorPtr = &opts.CursorHex
		}
		maxPages = opts.MaxPages
	}

	pages := 0
	var blockNumber uint64
	shieldedStateRoot := "0x"
	totalEncrypted := 0
	fetched := 0
	skipped := 0
	owned := make([]ShieldedNote, 0)
	spent := make(map[string]struct{})

	for {
		page, err := provider.ViewBalances(material.GrantID, limitPtr, cursorPtr)
		if err != nil {
			return nil, err
		}
		blockNumber = page.BlockNumber
		shieldedStateRoot = page.ShieldedStateRoot
		totalEncrypted = page.TotalEncryptedNoteCount
		fetched += page.ReturnedEncryptedNoteCount
		for _, n := range page.SpentNullifiers {
			spent[normalizeHex(n)] = struct{}{}
		}

		for _, entry := range page.Notes {
			envelope, err := ParseEncryptedNotePayload(entry.EncryptedNote)
			if err != nil {
				if ignoreMalformed {
					skipped++
					continue
				}
				return nil, err
			}
			if material.RecipientPublicKey != "" && !hexEqual(envelope.Recipient, material.RecipientPublicKey) {
				continue
			}
			plaintext, err := material.Decrypt(GrantedNoteDecryptInput{
				NoteCommitment:   entry.NoteCommitment,
				EncryptedNoteHex: entry.EncryptedNote,
				Envelope:         *envelope,
			})
			if err != nil {
				return nil, err
			}
			if len(plaintext) == 0 {
				continue
			}
			note, err := ParseShieldedNotePlaintext(plaintext)
			if err != nil {
				if ignoreMalformed {
					skipped++
					continue
				}
				return nil, err
			}
			owned = append(owned, *note)
		}

		pages++
		if page.NextCursor == nil {
			break
		}
		cursorPtr = page.NextCursor
		if maxPages > 0 && pages >= maxPages {
			break
		}
	}

	spentList := make([]string, 0, len(spent))
	for n := range spent {
		spentList = append(spentList, n)
	}
	portfolio := ReconstructPortfolio(owned, &ReconstructOptions{
		DeriveNullifier: deriveNullifier,
		SpentNullifiers: spentList,
	})

	return &BalanceReconstructionResult{
		ReconstructedPortfolio:  portfolio,
		GrantID:                 material.GrantID,
		BlockNumber:             blockNumber,
		ShieldedStateRoot:       shieldedStateRoot,
		TotalEncryptedNoteCount: totalEncrypted,
		FetchedEncryptedNoteCnt: fetched,
		SkippedMalformedCount:   skipped,
		SpentNullifierCount:     len(spent),
	}, nil
}
