package analysis

type InvalidBid struct {
	Reason string
	Type   uint
}

const (
	InvalidBidConsensusType uint = iota
	InvalidBidIgnoredPreferencesType
)
