package consensus

type ValidatorStatus string

const (
	StatusValidatorUnknown ValidatorStatus = "unknown"
	StatusValidatorActive  ValidatorStatus = "active"
	StatusValidatorPending ValidatorStatus = "pending"
)
