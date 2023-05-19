package store

import (
	"encoding/json"
	"testing"

	"github.com/attestantio/go-builder-client/api/capella"
	"github.com/attestantio/go-builder-client/spec"
	consensusspec "github.com/attestantio/go-eth2-client/spec"
	"github.com/holiman/uint256"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

func TestBidEntryToBid(t *testing.T) {
	versionedSignedBuilderBid := &spec.VersionedSignedBuilderBid{
		Version: consensusspec.DataVersionCapella,
		Capella: &capella.SignedBuilderBid{
			Message: &capella.BuilderBid{
				Value: uint256.NewInt(1000),
			},
		},
	}
	versionedSignedBuilderBidJSON, _ := json.Marshal(versionedSignedBuilderBid)

	type args struct {
		entry *BidEntry
	}
	tests := []struct {
		name    string
		args    args
		want    *types.VersionedBid
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				entry: &BidEntry{},
			},
			wantErr: true,
		},
		{
			name: "incomplete",
			args: args{
				entry: &BidEntry{
					ID:        1,
					Bid:       string(versionedSignedBuilderBidJSON),
					Signature: "0x1234567890",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BidEntryToBid(tt.args.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("BidEntryToBid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BidEntryToBid() = %v, want %v", got, tt.want)
			}
		})
	}
}
