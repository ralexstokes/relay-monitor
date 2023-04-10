package consensus

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/protolambda/eth2api"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

// NOTE: code in this file supports the unreleased API to fetch the `randao` value from the beacon state
// See: https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Beacon/getStateRandao

type RandaoResponse struct {
	Randao common.Root `json:"randao"`
}

func FetchRandao(ctx context.Context, httpClient *eth2api.Eth2HttpClient, slot types.Slot) (phase0.Hash32, error) {
	var dest RandaoResponse
	_, err := eth2api.SimpleRequest(ctx, httpClient, eth2api.FmtGET("/eth/v1/beacon/states/%d/randao", slot), eth2api.Wrap(&dest))
	return phase0.Hash32(dest.Randao), err
}
