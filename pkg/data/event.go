package data

import "github.com/ralexstokes/relay-monitor/pkg/types"

type Event struct {
	Relay types.PublicKey
	Bid   *types.Bid
}
