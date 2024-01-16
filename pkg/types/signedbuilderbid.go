package types

import (
	"errors"

	"github.com/attestantio/go-builder-client/spec"
	consensusspec "github.com/attestantio/go-eth2-client/spec"
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

func (v *VersionedSignedBuilderBid) Message() (interface{}, error) {
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
		return v.Bellatrix, nil
	case consensusspec.DataVersionCapella:
		if v.Capella == nil {
			return 0, errors.New("no data")
		}
		if v.Capella.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Capella, nil
	case consensusspec.DataVersionDeneb:
		if v.Deneb == nil {
			return 0, errors.New("no data")
		}
		if v.Deneb.Message == nil {
			return 0, errors.New("no data message")
		}
		return v.Deneb, nil
	default:
		return 0, errors.New("unsupported version")
	}
}
