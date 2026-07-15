package mersennet

import (
	"math/big"
	"testing"
)

func mkNote(value int64, assetID uint32, tag byte) ShieldedNote {
	rho := "0x"
	for i := 0; i < 32; i++ {
		rho += string("0123456789abcdef"[tag>>4]) + string("0123456789abcdef"[tag&0xf])
	}
	return ShieldedNote{
		Value:   big.NewInt(value),
		AssetID: assetID,
		OwnerPK: "0x" + repeat("11", 32),
		Rho:     rho,
		Psi:     "0x" + repeat("99", 32),
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestReconstructPortfolioSumsUnspent(t *testing.T) {
	notes := []ShieldedNote{mkNote(100, 0, 0xaa), mkNote(50, 0, 0xbb), mkNote(7, 1, 0xcc)}
	p := ReconstructPortfolio(notes, nil)
	if p.PerAsset[0].Cmp(big.NewInt(150)) != 0 {
		t.Fatalf("asset 0 = %s, want 150", p.PerAsset[0])
	}
	if p.PerAsset[1].Cmp(big.NewInt(7)) != 0 {
		t.Fatalf("asset 1 = %s, want 7", p.PerAsset[1])
	}
	if p.UnspentNoteCount != 3 {
		t.Fatalf("unspent = %d, want 3", p.UnspentNoteCount)
	}
}

func TestReconstructPortfolioExcludesSpent(t *testing.T) {
	spentNote := mkNote(100, 0, 0xaa)
	liveNote := mkNote(50, 0, 0xbb)
	spentNullifier := DefaultNullifierDeriver(spentNote)
	p := ReconstructPortfolio([]ShieldedNote{spentNote, liveNote}, &ReconstructOptions{
		SpentNullifiers: []string{spentNullifier},
	})
	if p.PerAsset[0].Cmp(big.NewInt(50)) != 0 {
		t.Fatalf("asset 0 = %s, want 50", p.PerAsset[0])
	}
	if p.SpentNoteCount != 1 || p.UnspentNoteCount != 1 {
		t.Fatalf("spent=%d unspent=%d, want 1/1", p.SpentNoteCount, p.UnspentNoteCount)
	}
}

func TestReconstructPortfolioDedupes(t *testing.T) {
	n := mkNote(100, 0, 0xaa)
	p := ReconstructPortfolio([]ShieldedNote{n, n, n}, nil)
	if p.TotalNoteCount != 1 {
		t.Fatalf("total = %d, want 1", p.TotalNoteCount)
	}
}

func TestDefaultNullifierDeterministic(t *testing.T) {
	n := mkNote(1, 0, 0xaa)
	a := DefaultNullifierDeriver(n)
	b := DefaultNullifierDeriver(n)
	if a != b {
		t.Fatalf("nullifier not deterministic: %s != %s", a, b)
	}
	if len(a) != 66 {
		t.Fatalf("nullifier length = %d, want 66", len(a))
	}
}

func TestReconstructOpenOrders(t *testing.T) {
	orders := []OrderRecord{
		{OrderID: "o1", MarketID: 1, Side: "buy", Price: big.NewInt(100), Size: big.NewInt(10)},
		{OrderID: "o2", MarketID: 1, Side: "sell", Price: big.NewInt(200), Size: big.NewInt(5), Status: "cancelled"},
	}
	fills := []FillRecord{{OrderID: "o1", MarketID: 1, Side: "buy", Price: big.NewInt(100), Size: big.NewInt(4)}}
	open, err := ReconstructOpenOrders(orders, fills)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].OrderID != "o1" || open[0].Remaining.Cmp(big.NewInt(6)) != 0 {
		t.Fatalf("unexpected open orders: %+v", open)
	}
}

func TestReconstructPositionsAvgCostAndPnl(t *testing.T) {
	fills := []FillRecord{
		{OrderID: "a", MarketID: 1, Side: "buy", Price: big.NewInt(100), Size: big.NewInt(10)},
		{OrderID: "b", MarketID: 1, Side: "buy", Price: big.NewInt(120), Size: big.NewInt(10)},
		{OrderID: "c", MarketID: 1, Side: "sell", Price: big.NewInt(150), Size: big.NewInt(5)},
	}
	positions, err := ReconstructPositions(fills)
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) != 1 {
		t.Fatalf("positions = %d, want 1", len(positions))
	}
	pos := positions[0]
	if pos.NetSize.Cmp(big.NewInt(15)) != 0 {
		t.Fatalf("net = %s, want 15", pos.NetSize)
	}
	if pos.EntryPrice.Cmp(big.NewInt(110)) != 0 {
		t.Fatalf("entry = %s, want 110", pos.EntryPrice)
	}
	if pos.RealizedPnl.Cmp(big.NewInt(200)) != 0 {
		t.Fatalf("pnl = %s, want 200", pos.RealizedPnl)
	}
}

func TestReconstructPositionsFlip(t *testing.T) {
	fills := []FillRecord{
		{OrderID: "a", MarketID: 2, Side: "buy", Price: big.NewInt(100), Size: big.NewInt(5)},
		{OrderID: "b", MarketID: 2, Side: "sell", Price: big.NewInt(120), Size: big.NewInt(8)},
	}
	pos, err := ReconstructPositions(fills)
	if err != nil {
		t.Fatal(err)
	}
	if pos[0].NetSize.Cmp(big.NewInt(-3)) != 0 {
		t.Fatalf("net = %s, want -3", pos[0].NetSize)
	}
	if pos[0].EntryPrice.Cmp(big.NewInt(120)) != 0 {
		t.Fatalf("entry = %s, want 120", pos[0].EntryPrice)
	}
	if pos[0].RealizedPnl.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("pnl = %s, want 100", pos[0].RealizedPnl)
	}
}

func sampleResult() *BalanceReconstructionResult {
	return &BalanceReconstructionResult{
		ReconstructedPortfolio: ReconstructedPortfolio{
			PerAsset:         map[uint32]*big.Int{0: big.NewInt(150), 1: big.NewInt(7)},
			UnspentNoteCount: 3,
			SpentNoteCount:   1,
			TotalNoteCount:   4,
		},
		GrantID:             "0xGRANT",
		BlockNumber:         4242,
		ShieldedStateRoot:   "0xROOT",
		SpentNullifierCount: 1,
	}
}

func TestAttestationRoundtrip(t *testing.T) {
	att, err := BuildPortfolioAttestation(sampleResult(), BuildAttestationOptions{GrantorCommitment: "0xC", IssuedAt: 1000})
	if err != nil {
		t.Fatal(err)
	}
	ok, reason := VerifyAttestation(att, nil)
	if !ok {
		t.Fatalf("verify failed: %s", reason)
	}
	if att.Version != AttestationVersion {
		t.Fatalf("version = %s", att.Version)
	}
}

func TestAttestationDigestOrderIndependent(t *testing.T) {
	a := ComputePortfolioDigest("0xg", "0xc", 1, "0xr", map[uint32]*big.Int{1: big.NewInt(5), 0: big.NewInt(9)}, 2, 0)
	b := ComputePortfolioDigest("0xg", "0xc", 1, "0xr", map[uint32]*big.Int{0: big.NewInt(9), 1: big.NewInt(5)}, 2, 0)
	if a != b {
		t.Fatalf("digest not order-independent: %s != %s", a, b)
	}
}

func TestCrossLanguageDigestVector(t *testing.T) {
	// MUST equal the vector pinned in the TS (attestation.test.js) and Python
	// (test_reconstruction.py) suites - proves byte-identical digests.
	const crossLang = "0x574bebc386931031d68b18ccb0af7ac37b14278217badf0a73fc85146816c1f5"
	got := ComputePortfolioDigest("0xg", "0xc", 1, "0xr", map[uint32]*big.Int{0: big.NewInt(9), 1: big.NewInt(5)}, 2, 0)
	if got != crossLang {
		t.Fatalf("cross-language digest mismatch: %s != %s", got, crossLang)
	}
}

func TestAttestationTamperDetected(t *testing.T) {
	att, _ := BuildPortfolioAttestation(sampleResult(), BuildAttestationOptions{GrantorCommitment: "0xC", IssuedAt: 1})
	att.PerAsset["0"] = "999999"
	if ok, _ := VerifyAttestation(att, nil); ok {
		t.Fatal("tampered attestation verified as OK")
	}
}

func TestAttestationOnchainMismatch(t *testing.T) {
	att, _ := BuildPortfolioAttestation(sampleResult(), BuildAttestationOptions{
		GrantorCommitment: "0xC", OnchainPortfolioDigst: "0xdeadbeef", IssuedAt: 1,
	})
	if att.DigestMatchesOnchain == nil || *att.DigestMatchesOnchain {
		t.Fatal("expected onchain digest mismatch flagged")
	}
	if ok, _ := VerifyAttestation(att, nil); ok {
		t.Fatal("attestation with onchain mismatch verified as OK")
	}
}

func TestAttestationSignatureHook(t *testing.T) {
	sign := func(digest string) (string, error) { return "0xSIG:" + digest[len(digest)-8:], nil }
	verify := func(digest, sig, attester string) bool { return sig == "0xSIG:"+digest[len(digest)-8:] }
	att, err := BuildPortfolioAttestation(sampleResult(), BuildAttestationOptions{
		GrantorCommitment: "0xC", Attester: "auditor-1", Sign: sign, IssuedAt: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ok, reason := VerifyAttestation(att, verify)
	if !ok {
		t.Fatalf("signed attestation verify failed: %s", reason)
	}
}

func TestViewingKeyDeterministic(t *testing.T) {
	a := ViewingKeyFromSeed("recovery phrase")
	b := ViewingKeyFromSeed("recovery phrase")
	if a.ViewPK != b.ViewPK {
		t.Fatal("viewing key not deterministic")
	}
	if a.SpendSK == a.ViewSK {
		t.Fatal("spend and view keys should differ")
	}
	material := CreateOwnerViewingMaterial(a, "0xgrant")
	if material.RecipientPublicKey != a.ViewPK || material.GrantID != "0xgrant" {
		t.Fatal("owner viewing material mismatch")
	}
}
