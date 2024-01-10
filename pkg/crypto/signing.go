package crypto

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/flashbots/go-boost-utils/ssz"
)

var (
	VerifySignature          = ssz.VerifySignature
	DomainTypeAppBuilder     = ssz.DomainTypeAppBuilder
	DomainTypeBeaconProposer = ssz.DomainTypeBeaconProposer
	ComputeDomain            = ssz.ComputeDomain
)

type Domain = phase0.Domain
