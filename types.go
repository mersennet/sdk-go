package mersennet

// Block represents block data from eth_getBlockByNumber / eth_getBlockByHash.
// JSON tags match the camelCase field names emitted by the node's BlockDto.
type Block struct {
	Number       string        `json:"number"`
	Hash         string        `json:"hash"`
	GasLimit     string        `json:"gasLimit"`
	GasUsed      string        `json:"gasUsed"`
	BaseFee      string        `json:"baseFeePerGas"`
	StateRoot    string        `json:"stateRoot"`
	Transactions []interface{} `json:"transactions"`
	DomainEvents []interface{} `json:"domainEvents,omitempty"`
}

// Transaction represents transaction data.
// JSON tags match the camelCase field names emitted by the node's TxDto.
type Transaction struct {
	Hash     string  `json:"hash"`
	From     string  `json:"from"`
	To       *string `json:"to"`
	Value    string  `json:"value"`
	Nonce    string  `json:"nonce"`
	Gas      string  `json:"gas"`
	GasPrice string  `json:"gasPrice"`
	Input    string  `json:"input"`
}

// Receipt represents a transaction receipt.
// JSON tags match the camelCase field names emitted by the node's ReceiptDto.
type Receipt struct {
	TransactionHash   string     `json:"transactionHash"`
	BlockHash         string     `json:"blockHash"`
	BlockNumber       string     `json:"blockNumber"`
	TransactionIndex  string     `json:"transactionIndex"`
	From              string     `json:"from"`
	To                *string    `json:"to"`
	GasUsed           string     `json:"gasUsed"`
	CumulativeGasUsed string     `json:"cumulativeGasUsed"`
	EffectiveGasPrice string     `json:"effectiveGasPrice"`
	Status            string     `json:"status"`
	ContractAddress   *string    `json:"contractAddress"`
	LogsBloom         string     `json:"logsBloom"`
	Type              string     `json:"type"`
	Logs              []LogEntry `json:"logs"`
}

// LogEntry represents a log entry.
// JSON tags match the camelCase field names emitted by the node's LogDto.
type LogEntry struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	BlockHash        string   `json:"blockHash"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
	LogIndex         string   `json:"logIndex"`
	Removed          bool     `json:"removed"`
}

// Order represents an order in the order book
type Order struct {
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Price    string `json:"price"`
	Size     string `json:"size"`
	TIF      string `json:"tif"`
}

// OrderBookLevel represents a price level
type OrderBookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderBook represents the full order book
type OrderBook struct {
	Bids []OrderBookLevel `json:"bids"`
	Asks []OrderBookLevel `json:"asks"`
}

// Trade represents a trade execution
type Trade struct {
	Taker    string `json:"taker"`
	Maker    string `json:"maker"`
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Price    string `json:"price"`
	Size     string `json:"size"`
}

// ViewNotesEntry represents one encrypted note returned by mersennet_viewNotes.
type ViewNotesEntry struct {
	NoteCommitment string `json:"noteCommitment"`
	EncryptedNote  string `json:"encryptedNote"`
}

// ViewNotesResult represents the mersennet_viewNotes response.
type ViewNotesResult struct {
	GrantID                    string           `json:"grantId"`
	GrantorCommitment          string           `json:"grantorCommitment"`
	BlockNumber                uint64           `json:"blockNumber"`
	ShieldedStateRoot          string           `json:"shieldedStateRoot"`
	TotalEncryptedNoteCount    int              `json:"totalEncryptedNoteCount"`
	ReturnedEncryptedNoteCount int              `json:"returnedEncryptedNoteCount"`
	NextCursor                 *string          `json:"nextCursor"`
	Notes                      []ViewNotesEntry `json:"notes"`
	SignatureVerified          bool             `json:"signatureVerified"`
}

// ViewBalancesResult is the mersennet_viewBalances response: an encrypted-note
// page plus the spent-nullifier set for client-side balance reconstruction.
type ViewBalancesResult struct {
	GrantID                    string           `json:"grantId"`
	GrantorCommitment          string           `json:"grantorCommitment"`
	BlockNumber                uint64           `json:"blockNumber"`
	ShieldedStateRoot          string           `json:"shieldedStateRoot"`
	TotalEncryptedNoteCount    int              `json:"totalEncryptedNoteCount"`
	ReturnedEncryptedNoteCount int              `json:"returnedEncryptedNoteCount"`
	NextCursor                 *string          `json:"nextCursor"`
	Notes                      []ViewNotesEntry `json:"notes"`
	SpentNullifiers            []string         `json:"spentNullifiers"`
	SpentNullifierCount        int              `json:"spentNullifierCount"`
	Reconstruction             string           `json:"reconstruction"`
	SignatureVerified          bool             `json:"signatureVerified"`
}

// ViewMarketAggregate is one public per-market aggregate in a trading view read.
type ViewMarketAggregate struct {
	MarketID          int    `json:"marketId"`
	MarkPrice         string `json:"markPrice"`
	LongOpenInterest  string `json:"longOpenInterest"`
	ShortOpenInterest string `json:"shortOpenInterest"`
	LastClearingPrice string `json:"lastClearingPrice"`
	LastVolume        string `json:"lastVolume"`
	LiquidatableCount int    `json:"liquidatableCount"`
}

// ViewTradingResult is the mersennet_viewPositions / mersennet_viewOrders
// response: public market context + grant binding only (rows reconstructed
// client-side).
type ViewTradingResult struct {
	GrantID           string `json:"grantId"`
	GrantorCommitment string `json:"grantorCommitment"`
	BlockNumber       uint64 `json:"blockNumber"`
	ShieldedStateRoot string `json:"shieldedStateRoot"`
	MarketAggregates  struct {
		Markets []ViewMarketAggregate `json:"markets"`
	} `json:"marketAggregates"`
	Reconstruction    string `json:"reconstruction"`
	SignatureVerified bool   `json:"signatureVerified"`
}

// ViewGrantStatus is the mersennet_viewGrantStatus response.
type ViewGrantStatus struct {
	Exists            bool                   `json:"exists"`
	Status            string                 `json:"status"`
	ActiveNow         bool                   `json:"activeNow"`
	SignatureVerified bool                   `json:"signatureVerified"`
	Revoked           bool                   `json:"revoked"`
	RevokedAtBlock    *uint64                `json:"revokedAtBlock"`
	GrantToken        map[string]interface{} `json:"grantToken"`
}
