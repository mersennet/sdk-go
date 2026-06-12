package mersennet

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// Orders provides CLOB interaction for Mersennet
type Orders struct {
	provider *Provider
}

// NewOrders creates a new Orders client
func NewOrders(provider *Provider) *Orders {
	return &Orders{provider: provider}
}

// toHexAmount normalizes a decimal (or already-hex) amount string to a
// 0x-prefixed hex string, mirroring the TS SDK's toHexAmount. Non-numeric
// input falls back to "0x0".
func toHexAmount(s string) string {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return "0x0"
	}
	return "0x" + n.Text(16)
}

// AddMarket adds a new market (admin). Returns market ID.
func (o *Orders) AddMarket(base, quote, lot, tick string) (uint64, error) {
	symbol := base + "-" + quote
	result, err := o.provider.request("mersennet_orders_addMarket", []interface{}{
		symbol,
		toHexAmount(tick),
		toHexAmount(lot),
	})
	if err != nil {
		return 0, err
	}
	var s string
	if err := json.Unmarshal(result, &s); err != nil {
		return 0, err
	}
	var n uint64
	if _, err := fmt.Sscanf(s, "0x%x", &n); err != nil {
		return 0, err
	}
	return n, nil
}

// PlaceOrder places an order
func (o *Orders) PlaceOrder(market uint64, side, price, amount, tif, owner string) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"owner":     owner,
		"market_id": market,
		"side":      side,
		"price":     toHexAmount(price),
		"size":      toHexAmount(amount),
		"tif":       tif,
	}
	result, err := o.provider.request("mersennet_orders_submitOrder", []interface{}{params})
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(result, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// CancelOrder cancels an order by ID
func (o *Orders) CancelOrder(orderID uint64) (bool, error) {
	result, err := o.provider.request("mersennet_orders_cancelOrder", []interface{}{
		fmt.Sprintf("0x%x", orderID),
	})
	if err != nil {
		return false, err
	}
	var b bool
	if err := json.Unmarshal(result, &b); err != nil {
		return false, err
	}
	return b, nil
}

// GetOrderBook returns the order book for a market
func (o *Orders) GetOrderBook(market uint64) (*OrderBook, error) {
	result, err := o.provider.request("mersennet_orders_getOrderBook", []interface{}{
		fmt.Sprintf("0x%x", market),
	})
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return &OrderBook{Bids: []OrderBookLevel{}, Asks: []OrderBookLevel{}}, nil
	}
	var book OrderBook
	if err := json.Unmarshal(result, &book); err != nil {
		return nil, err
	}
	return &book, nil
}
