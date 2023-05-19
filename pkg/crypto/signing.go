package crypto

import (
	boostTypes "github.com/flashbots/go-boost-utils/types"
)

var (
	VerifySignature          = boostTypes.VerifySignature
	DomainTypeAppBuilder     = boostTypes.DomainTypeAppBuilder
	DomainTypeBeaconProposer = boostTypes.DomainTypeBeaconProposer
	ComputeDomain            = boostTypes.ComputeDomain
)

type Domain = boostTypes.Domain
