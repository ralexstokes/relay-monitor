package types

import (
	"fmt"

	builderApiCapella "github.com/attestantio/go-builder-client/api/capella"
	builderApiV1 "github.com/attestantio/go-builder-client/api/v1"
	consensusapiv1capella "github.com/attestantio/go-eth2-client/api/v1/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

type (
	Slot                        = uint64
	Epoch                       = uint64
	Uint256                     = uint256.Int
	Hash                        = phase0.Hash32
	Bid                         = builderApiCapella.SignedBuilderBid
	PublicKey                   = phase0.BLSPubKey
	Root                        = phase0.Root
	ForkVersion                 = phase0.Version
	ValidatorIndex              = uint64
	SignedValidatorRegistration = builderApiV1.SignedValidatorRegistration
	SignedBlindedBeaconBlock    = consensusapiv1capella.SignedBlindedBeaconBlock
)

type Coordinate struct {
	Slot Slot
	Root Root
}

type AuctionTranscript struct {
	Bid        Bid                                            `json:"bid"`
	Acceptance consensusapiv1capella.SignedBlindedBeaconBlock `json:"acceptance"`
}

type BidContext struct {
	Slot              Slot      `json:"slot,omitempty"`
	ParentHash        Hash      `json:"parent_hash,omitempty"`
	ProposerPublicKey PublicKey `json:"proposer_public_key,omitempty"`
	RelayPublicKey    PublicKey `json:"relay_public_key,omitempty"`
	Error             error     `json:"error,omitempty"`
}

type ErrorType string

const (
	ParentHashErr ErrorType = "ParentHashError"
	PubKeyErr     ErrorType = "PublicKeyError"
	EmptyBidError ErrorType = "EmptyBidError"
	RelayError    ErrorType = "RelayError"
	ValidationErr ErrorType = "ValidationError"
)

type ClientError struct {
	Type    ErrorType `json:"errorType,omitempty"`
	Code    int       `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("Type: %s Code: %d Message: %s", e.Type, e.Code, e.Message)
}

type GetHeaderResponse struct {
	Version string
	Data    *builderApiCapella.SignedBuilderBid
}

var (
	ErrLength = fmt.Errorf("incorrect byte length")
	ErrSign   = fmt.Errorf("negative value casted as unsigned int")
)

func FromSlice(x []byte, p *PublicKey) error {
	if len(x) != 48 {
		return ErrLength
	}
	copy(p[:], x)
	return nil
}

func UnmarshalText(input []byte, p *PublicKey) error {
	b := hexutil.Bytes(p[:])
	if err := b.UnmarshalText(input); err != nil {
		return err
	}
	return FromSlice(b, p)
}
