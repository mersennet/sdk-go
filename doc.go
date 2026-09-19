// Package mersennet is the official Go client for Mersennet, the private, verifiable
// network (https://mersennet.com).
//
// It covers the JSON-RPC API (eth_* plus the mersennet_* namespace), the
// on-chain order book exposed by the MersennetOrders precompile (markets,
// protocol switches, positions, collateral, agent keys, liquidatable
// accounts), and the shielded layer (viewing keys, note scanning, client-side
// balance and position reconstruction, portfolio attestations).
//
// Quick start:
//
//	p := mersennet.NewProvider("https://rpc.mersennet.com")
//	orders := mersennet.NewOrders(p)
//	markets, err := orders.GetMarkets()
//
// Documentation: https://docs.mersennet.com/developers/sdks/go/
package mersennet
