package types

import (
	"encoding/json"
	"errors"
	"fmt"

	v1 "github.com/attestantio/go-builder-client/api/v1"
	"github.com/attestantio/go-builder-client/spec"
	consensusapiv1capella "github.com/attestantio/go-eth2-client/api/v1/capella"
	consensusspec "github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

type (
	Slot                        = phase0.Slot
	Epoch                       = uint64
	ForkVersion                 = phase0.Version
	Uint256                     = uint256.Int
	Hash                        = phase0.Hash32
	Bid                         = VersionedSignedBuilderBid
	Root                        = phase0.Root
	ValidatorIndex              = uint64
	SignedValidatorRegistration = v1.SignedValidatorRegistration
	SignedBlindedBeaconBlock    = consensusapiv1capella.SignedBlindedBeaconBlock
)

type VersionedSignedBuilderBid struct {
	spec.VersionedSignedBuilderBid
}

func (v *VersionedSignedBuilderBid) GasUsed() (uint64, error) {
	if v == nil {
		return 0, errors.New("nil struct")
	}
	switch v.Version {
	case consensusspec.DataVersionBellatrix:
		if v.Bellatrix == nil {
			return 0, errors.New("no data")
		}
		if v.Bellatrix.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Bellatrix.Message.Header.GasUsed, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return 0, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Capella.Message.Header.GasUsed, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return 0, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Deneb.Message.Header.GasUsed, nil
	default:
		return 0, errors.New("unsupported version")
	}
}

func (v *VersionedSignedBuilderBid) GasLimit() (uint64, error) {
	if v == nil {
		return 0, errors.New("nil struct")
	}
	switch v.Version {
	case consensusspec.DataVersionBellatrix:
		if v.Bellatrix == nil {
			return 0, errors.New("no data")
		}
		if v.Bellatrix.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Bellatrix.Message.Header.GasLimit, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return 0, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Capella.Message.Header.GasLimit, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return 0, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Deneb.Message.Header.GasLimit, nil
	default:
		return 0, errors.New("unsupported version")
	}
}

func (v *VersionedSignedBuilderBid) HashTreeRoot() ([32]byte, error) {
	if v == nil {
		return [32]byte{}, errors.New("nil struct")
	}
	switch v.Version {
	case consensusspec.DataVersionBellatrix:
		if v.Bellatrix == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Bellatrix.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Bellatrix.Message.HashTreeRoot()
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Capella.Message.HashTreeRoot()
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Deneb.Message.HashTreeRoot()
	default:
		return [32]byte{}, errors.New("unsupported version")
	}
}

func (v *VersionedSignedBuilderBid) BaseFeePerGas() ([32]byte, error) {
	if v == nil {
		return [32]byte{}, errors.New("nil struct")
	}
	switch v.Version {
	case consensusspec.DataVersionBellatrix:
		if v.Bellatrix == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Bellatrix.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Bellatrix.Message.Header.BaseFeePerGas, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Capella.Message.Header.BaseFeePerGas, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Deneb.Message.Header.BaseFeePerGas.Bytes32(), nil
	default:
		return [32]byte{}, errors.New("unsupported version")
	}
}

func (v *VersionedSignedBuilderBid) Random() ([32]byte, error) {
	if v == nil {
		return [32]byte{}, errors.New("nil struct")
	}
	switch v.Version {
	case consensusspec.DataVersionBellatrix:
		if v.Bellatrix == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Bellatrix.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Bellatrix.Message.Header.PrevRandao, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Capella.Message.Header.PrevRandao, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return [32]byte{}, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return [32]byte{}, errors.New("no data message")
		}
		return v.Deneb.Message.Header.PrevRandao, nil
	default:
		return [32]byte{}, errors.New("unsupported version")
	}
}

var (
	ErrLength = fmt.Errorf("incorrect byte length")
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

type PublicKey [48]byte

func (p PublicKey) MarshalText() ([]byte, error) {
	return hexutil.Bytes(p[:]).MarshalText()
}

func (p *PublicKey) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(p[:])
	if err := b.UnmarshalJSON(input); err != nil {
		return err
	}
	return p.FromSlice(b)
}

func (p *PublicKey) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(p[:])
	if err := b.UnmarshalText(input); err != nil {
		return err
	}
	return p.FromSlice(b)
}

func (p PublicKey) String() string {
	return hexutil.Bytes(p[:]).String()
}

func (p *PublicKey) FromSlice(x []byte) error {
	if len(x) != 48 {
		return ErrLength
	}
	copy(p[:], x)
	return nil
}

// GetHeaderResponse is the response payload from the getHeader request: https://github.com/ethereum/builder-specs/pull/2/files#diff-c80f52e38c99b1049252a99215450a29fd248d709ffd834a9480c98a233bf32c
type GetHeaderResponse = *VersionedSignedBuilderBid

type VersionString string

func (s VersionString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s *VersionString) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return ErrLength
	}
	*s = VersionString(b[1 : len(b)-1])
	return nil
}
