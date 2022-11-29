package crypto

import "github.com/flashbots/go-boost-utils/types"

var (
	VerifySignature          = types.VerifySignature
	DomainTypeAppBuilder     = types.DomainTypeAppBuilder
	DomainTypeBeaconProposer = types.DomainTypeBeaconProposer
	ComputeDomain            = types.ComputeDomain
)

type Domain = types.Domain
