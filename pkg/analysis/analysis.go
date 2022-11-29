package analysis

type InvalidBid struct {
	Reason  string
	Type    uint
	Context map[string]interface{}
}

const (
	InvalidBidConsensusType uint = iota
	InvalidBidIgnoredPreferencesType
)
