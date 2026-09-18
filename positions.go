package mersennet

// Client-side open-order and position reconstruction (Workstream F5,
// `orders:read` / `positions:read`).
//
// On a fully shielded chain the node never sees order/position plaintext, so a
// wallet reconstructs its own trading state locally from the OrderRecords it
// authored and the FillRecords it attributed from public clearing events.
// Faithful port of sdk-ts/src/positions.ts.

import (
	"fmt"
	"math/big"
	"sort"
)

// OrderRecord is an order the wallet submitted, as recorded at submit time.
type OrderRecord struct {
	OrderID  string
	MarketID int
	Side     string // "buy" | "sell"
	Price    *big.Int
	Size     *big.Int
	Status   string // "" | "open" | "cancelled"
}

// FillRecord is a fill the wallet attributed to one of its orders.
type FillRecord struct {
	OrderID  string
	MarketID int
	Side     string
	Price    *big.Int
	Size     *big.Int
}

// OpenOrder is a still-resting order with its unfilled remainder.
type OpenOrder struct {
	OrderID   string
	MarketID  int
	Side      string
	Price     *big.Int
	Remaining *big.Int
}

// ReconstructedPosition is a per-market net position.
type ReconstructedPosition struct {
	MarketID    int
	NetSize     *big.Int // signed: positive long, negative short, 0 flat
	EntryPrice  *big.Int // size-weighted average entry of open exposure
	RealizedPnl *big.Int
}

func sumFilledByOrder(fills []FillRecord) (map[string]*big.Int, error) {
	filled := make(map[string]*big.Int)
	for _, f := range fills {
		if f.Size.Sign() < 0 {
			return nil, fmt.Errorf("fill size must be non-negative (order %s)", f.OrderID)
		}
		if filled[f.OrderID] == nil {
			filled[f.OrderID] = new(big.Int)
		}
		filled[f.OrderID].Add(filled[f.OrderID], f.Size)
	}
	return filled, nil
}

// ReconstructOpenOrders reports every non-cancelled order whose filled size is
// below its original size, with the unfilled remainder.
func ReconstructOpenOrders(orders []OrderRecord, fills []FillRecord) ([]OpenOrder, error) {
	filled, err := sumFilledByOrder(fills)
	if err != nil {
		return nil, err
	}
	open := make([]OpenOrder, 0)
	for _, order := range orders {
		if order.Status == "cancelled" {
			continue
		}
		if order.Size.Sign() < 0 {
			return nil, fmt.Errorf("order size must be non-negative (order %s)", order.OrderID)
		}
		done := filled[order.OrderID]
		if done == nil {
			done = new(big.Int)
		}
		remaining := new(big.Int).Sub(order.Size, done)
		if remaining.Sign() > 0 {
			open = append(open, OpenOrder{
				OrderID:   order.OrderID,
				MarketID:  order.MarketID,
				Side:      order.Side,
				Price:     order.Price,
				Remaining: remaining,
			})
		}
	}
	return open, nil
}

type mutablePosition struct {
	netSize     *big.Int
	entryPrice  *big.Int
	realizedPnl *big.Int
}

func absBig(x *big.Int) *big.Int { return new(big.Int).Abs(x) }

func applyFill(pos *mutablePosition, signedSize *big.Int, price *big.Int) {
	sameDirection := pos.netSize.Sign() == 0 || (pos.netSize.Sign() > 0) == (signedSize.Sign() > 0)

	if sameDirection {
		prevAbs := absBig(pos.netSize)
		addAbs := absBig(signedSize)
		totalAbs := new(big.Int).Add(prevAbs, addAbs)
		if totalAbs.Sign() == 0 {
			pos.entryPrice = new(big.Int)
		} else {
			num := new(big.Int).Add(
				new(big.Int).Mul(pos.entryPrice, prevAbs),
				new(big.Int).Mul(price, addAbs),
			)
			pos.entryPrice = num.Div(num, totalAbs)
		}
		pos.netSize.Add(pos.netSize, signedSize)
		return
	}

	// Opposite direction: close against existing exposure first.
	closingAbs := absBig(signedSize)
	openAbs := absBig(pos.netSize)
	matched := closingAbs
	if openAbs.Cmp(closingAbs) < 0 {
		matched = openAbs
	}
	wasLong := pos.netSize.Sign() > 0
	var pnl *big.Int
	if wasLong {
		pnl = new(big.Int).Mul(new(big.Int).Sub(price, pos.entryPrice), matched)
	} else {
		pnl = new(big.Int).Mul(new(big.Int).Sub(pos.entryPrice, price), matched)
	}
	pos.realizedPnl.Add(pos.realizedPnl, pnl)

	remainderAbs := new(big.Int).Sub(closingAbs, matched)
	pos.netSize.Add(pos.netSize, signedSize)

	if pos.netSize.Sign() == 0 {
		pos.entryPrice = new(big.Int)
	} else if remainderAbs.Sign() > 0 {
		pos.entryPrice = new(big.Int).Set(price)
	}
}

// ReconstructPositions reconstructs per-market positions from fills using
// average-cost accounting. Pass fills chronologically.
func ReconstructPositions(fills []FillRecord) ([]ReconstructedPosition, error) {
	byMarket := make(map[int]*mutablePosition)
	for _, f := range fills {
		if f.Size.Sign() < 0 {
			return nil, fmt.Errorf("fill size must be non-negative (order %s)", f.OrderID)
		}
		pos := byMarket[f.MarketID]
		if pos == nil {
			pos = &mutablePosition{netSize: new(big.Int), entryPrice: new(big.Int), realizedPnl: new(big.Int)}
			byMarket[f.MarketID] = pos
		}
		signed := new(big.Int).Set(f.Size)
		if f.Side != "buy" {
			signed.Neg(signed)
		}
		applyFill(pos, signed, f.Price)
	}

	markets := make([]int, 0, len(byMarket))
	for m := range byMarket {
		markets = append(markets, m)
	}
	sort.Ints(markets)

	out := make([]ReconstructedPosition, 0, len(markets))
	for _, m := range markets {
		pos := byMarket[m]
		out = append(out, ReconstructedPosition{
			MarketID:    m,
			NetSize:     pos.netSize,
			EntryPrice:  pos.entryPrice,
			RealizedPnl: pos.realizedPnl,
		})
	}
	return out, nil
}
