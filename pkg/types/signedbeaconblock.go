package types

import (
	"errors"
	"math/big"

	"github.com/attestantio/go-eth2-client/spec"
	consensusspec "github.com/attestantio/go-eth2-client/spec"
)

type VersionedSignedBeaconBlock struct {
	spec.VersionedSignedBeaconBlock
}

func (v *VersionedSignedBeaconBlock) GasUsed() (uint64, error) {
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
		return v.Bellatrix.Message.Body.ExecutionPayload.GasUsed, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return 0, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Capella.Message.Body.ExecutionPayload.GasUsed, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return 0, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Deneb.Message.Body.ExecutionPayload.GasUsed, nil
	default:
		return 0, errors.New("unsupported version")
	}
}

func (v *VersionedSignedBeaconBlock) GasLimit() (uint64, error) {
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
		return v.Bellatrix.Message.Body.ExecutionPayload.GasLimit, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return 0, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Capella.Message.Body.ExecutionPayload.GasLimit, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return 0, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Deneb.Message.Body.ExecutionPayload.GasLimit, nil
	default:
		return 0, errors.New("unsupported version")
	}
}

func (v *VersionedSignedBeaconBlock) BlockHash() (Hash, error) {
	if v == nil {
		return Hash{}, errors.New("nil struct")
	}
	switch v.Version {
	case consensusspec.DataVersionBellatrix:
		if v.Bellatrix == nil {
			return Hash{}, errors.New("no data")
		}
		if v.Bellatrix.Message == nil {
			return Hash{}, errors.New("no data message")
		}
		return v.Bellatrix.Message.Body.ExecutionPayload.BlockHash, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return Hash{}, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return Hash{}, errors.New("no data message")
		}
		return v.Capella.Message.Body.ExecutionPayload.BlockHash, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return Hash{}, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return Hash{}, errors.New("no data message")
		}
		return v.Deneb.Message.Body.ExecutionPayload.BlockHash, nil
	default:
		return Hash{}, errors.New("unsupported version")
	}
}

func (v *VersionedSignedBeaconBlock) BaseFeePerGas() (*big.Int, error) {
	baseFee := new(big.Int)

	if v == nil {
		return baseFee, errors.New("nil struct")
	}
	switch v.Version {
	case consensusspec.DataVersionBellatrix:
		if v.Bellatrix == nil {
			return baseFee, errors.New("no data")
		}
		if v.Bellatrix.Message == nil {
			return baseFee, errors.New("no data message")
		}
		return baseFee.SetBytes(reverse(v.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas[:])), nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return baseFee, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return baseFee, errors.New("no data message")
		}
		return baseFee.SetBytes(reverse(v.Capella.Message.Body.ExecutionPayload.BaseFeePerGas[:])), nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return baseFee, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return baseFee, errors.New("no data message")
		}
		return v.Deneb.Message.Body.ExecutionPayload.BaseFeePerGas.ToBig(), nil
	default:
		return baseFee, errors.New("unsupported version")
	}
}
