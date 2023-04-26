package website

import (
	_ "embed"
	"math/big"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	// Printer for pretty printing numbers
	printer = message.NewPrinter(language.English)

	// Caser is used for casing strings
	caser = cases.Title(language.English)
)

type StatusHTMLData struct { //nolint:musttag
	Network string

	BellatrixForkVersion        string
	CapellaForkVersion          string
	GenesisForkVersion          string
	GenesisValidatorsRoot       string
	BuilderSigningDomain        string
	BeaconProposerSigningDomain string

	HeadSlot               uint64
	Relays                 []*types.Relay
	FaultStatsReport       map[string]*types.FaultStats
	FaultRecordsReport     map[string]*types.FaultRecords
	ReputationScoreReport  map[string]*types.Score
	BidDeliveryScoreReport map[string]*types.Score

	CountValidators              uint64
	CountValidatorsRegistrations uint64
	CountBidsAnalyzed            uint64
	CountBidsAnalyzedValid       uint64
	CountBidsAnalyzedFault       uint64

	ShowConfigDetails     bool
	LinkBeaconchain       string
	LinkEtherscan         string
	LinkRelayMonitorAPI   string
	LinkRelayMonitorNotes string

	LookbackSlotsValue uint64
}

func weiToEth(wei string) string {
	weiBigInt := new(big.Int)
	weiBigInt.SetString(wei, 10)
	ethValue := weiBigIntToEthBigFloat(weiBigInt)
	return ethValue.String()
}

func weiBigIntToEthBigFloat(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return
}

func prettyInt(i uint64) string {
	return printer.Sprintf("%d", i)
}

func caseIt(s string) string {
	return caser.String(s)
}

func truncate(s string) string {
	return s[:5] + "..." + s[len(s)-4:]
}

var funcMap = template.FuncMap{
	"weiToEth":  weiToEth,
	"prettyInt": prettyInt,
	"caseIt":    caseIt,
	"truncate":  truncate,
}

//go:embed website.html
var htmlContent string

func ParseIndexTemplate() (*template.Template, error) {
	return template.New("index").Funcs(funcMap).Funcs(sprig.FuncMap()).Parse(htmlContent)
}
