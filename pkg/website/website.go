// Package website contains the service delivering the website
package website

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/mev-boost-relay/common"
	"github.com/gorilla/mux"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/monitor"
	"github.com/ralexstokes/relay-monitor/pkg/reporter"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
	uberatomic "go.uber.org/atomic"
	"go.uber.org/zap"
)

var (
	ErrServerAlreadyStarted = errors.New("server was already started")
	EnablePprof             = os.Getenv("PPROF") == "1"
)

type WebsiteConfig struct {
	Host                string `yaml:"host"`
	Port                uint16 `yaml:"port"`
	ShowConfigDetails   bool   `yaml:"show_config_details"`
	LinkBeaconchain     string `yaml:"link_beaconchain"`
	LinkEtherscan       string `yaml:"link_etherscan"`
	LinkRelayMonitorAPI string `yaml:"link_relay_monitor_api"`
}

type Config struct {
	Network   *monitor.NetworkConfig   `yaml:"network"`
	Consensus *monitor.ConsensusConfig `yaml:"consensus"`
	Store     *monitor.StoreConfig     `yaml:"store"`
	Website   *WebsiteConfig           `yaml:"website"`
}

type WebserverOpts struct {
	ListenAddress  string
	NetworkDetails *common.EthNetworkDetails

	Reporter *reporter.Reporter
	Store    store.Storer
	Clock    *consensus.Clock
	Log      *zap.SugaredLogger

	ShowConfigDetails   bool
	LinkBeaconchain     string
	LinkEtherscan       string
	LinkRelayMonitorAPI string

	LookbackSlotsValue uint64
}

type Webserver struct {
	opts *WebserverOpts
	log  *zap.SugaredLogger

	reporter *reporter.Reporter
	store    store.Storer
	clock    *consensus.Clock

	lookbackSlotsValue uint64

	srv        *http.Server
	srvStarted uberatomic.Bool

	indexTemplate    *template.Template
	statusHTMLData   StatusHTMLData
	rootResponseLock sync.RWMutex

	htmlDefault *[]byte

	minifier *minify.M
}

func NewWebserver(opts *WebserverOpts) (*Webserver, error) {
	var err error

	minifier := minify.New()
	minifier.AddFunc("text/css", html.Minify)
	minifier.AddFunc("text/html", html.Minify)

	server := &Webserver{
		opts:     opts,
		log:      opts.Log,
		reporter: opts.Reporter,
		store:    opts.Store,
		clock:    opts.Clock,

		lookbackSlotsValue: opts.LookbackSlotsValue,

		htmlDefault: &[]byte{},

		minifier: minifier,
	}

	server.indexTemplate, err = ParseIndexTemplate()
	if err != nil {
		return nil, err
	}

	server.statusHTMLData = StatusHTMLData{
		Network:                      opts.NetworkDetails.Name,
		CountValidators:              0,
		CountValidatorsRegistrations: 0,
		CountBidsAnalyzed:            0,
		CountBidsAnalyzedValid:       0,
		CountBidsAnalyzedFault:       0,
		BellatrixForkVersion:         opts.NetworkDetails.BellatrixForkVersionHex,
		CapellaForkVersion:           opts.NetworkDetails.CapellaForkVersionHex,
		GenesisForkVersion:           opts.NetworkDetails.GenesisForkVersionHex,
		GenesisValidatorsRoot:        opts.NetworkDetails.GenesisValidatorsRootHex,
		BuilderSigningDomain:         hexutil.Encode(opts.NetworkDetails.DomainBuilder[:]),
		BeaconProposerSigningDomain:  hexutil.Encode(opts.NetworkDetails.DomainBeaconProposerBellatrix[:]),
		HeadSlot:                     0,
		ShowConfigDetails:            opts.ShowConfigDetails,
		LinkBeaconchain:              opts.LinkBeaconchain,
		LinkEtherscan:                opts.LinkEtherscan,
		LinkRelayMonitorAPI:          opts.LinkRelayMonitorAPI,
		LookbackSlotsValue:           opts.LookbackSlotsValue,
	}

	return server, nil
}

func (srv *Webserver) StartServer() (err error) {
	if srv.srvStarted.Swap(true) {
		return ErrServerAlreadyStarted
	}

	// Start background task to regularly update status HTML data
	go func() {
		for {
			srv.updateHTML()
			time.Sleep(10 * time.Second)
		}
	}()

	srv.srv = &http.Server{
		Addr:    srv.opts.ListenAddress,
		Handler: srv.getRouter(),

		ReadTimeout:       600 * time.Millisecond,
		ReadHeaderTimeout: 400 * time.Millisecond,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       3 * time.Second,
	}

	err = srv.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (srv *Webserver) getRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", srv.handleRoot).Methods(http.MethodGet)
	if EnablePprof {
		srv.log.Info("pprof API enabled")
		r.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	}

	withGz := gziphandler.GzipHandler(r)
	return withGz
}

func (srv *Webserver) updateHTML() {

	// Fetch relay monitor stats. First fetch validator stats.
	_countValidators, err := srv.store.GetCountValidators(context.Background())
	if err != nil {
		srv.log.Error("error getting number of validators")
	}
	_countValidatorsRegistrations, err := srv.store.GetCountValidatorsRegistrations(context.Background())
	if err != nil {
		srv.log.Error("error getting number of validator")
	}
	// Fetch bid analysis stats.
	_countBidsAnalyzed, err := srv.store.GetCountAnalysisLookbackSlots(context.Background(), srv.lookbackSlotsValue, nil)
	if err != nil {
		srv.log.Error("error getting number of bids analyzed")
	}
	_countBidsAnalyzedValid, err := srv.store.GetCountAnalysisLookbackSlots(context.Background(), srv.lookbackSlotsValue, &types.AnalysisQueryFilter{
		Category:   types.ValidBidCategory,
		Comparator: "=",
	})
	if err != nil {
		srv.log.Error("error getting number of bids analyzed")
	}
	_countBidsAnalyzedFault, err := srv.store.GetCountAnalysisLookbackSlots(context.Background(), srv.lookbackSlotsValue, &types.AnalysisQueryFilter{
		Category:   types.ValidBidCategory,
		Comparator: "!=",
	})
	if err != nil {
		srv.log.Error("error getting number of bids analyzed")
	}

	// Fetch monitored relays.
	_relays, err := srv.store.GetRelays(context.Background())
	if err != nil {
		srv.log.Error("error getting relays")
	}

	// Fetch current slot.
	_currentSlot := srv.clock.CurrentSlot(time.Now().Unix())
	_startSlot := _currentSlot - phase0.Slot(srv.lookbackSlotsValue)

	// Use current slot to build a slot bound (lookback) and fetch fault stats report.
	slotBounds := &types.SlotBounds{
		StartSlot: &_startSlot,
		EndSlot:   nil,
	}
	_faultStatsReport, err := srv.reporter.GetFaultStatsReport(context.Background(), slotBounds)
	if err != nil {
		srv.log.Error("error getting fault stats report")
	}

	// Fetch fault records report.
	_faultRecordsReport, err := srv.reporter.GetFaultRecordsReport(context.Background(), slotBounds)
	if err != nil {
		srv.log.Error("error getting fault records report")
	}

	// Fetch score reports.
	_reputationScoreReport, err := srv.reporter.GetReputationScoreReport(context.Background(), slotBounds, _currentSlot)
	if err != nil {
		srv.log.Error("error getting reputation score report")
	}
	_bidDeliveryScoreReport, err := srv.reporter.GetBidDeliveryScoreReport(context.Background(), slotBounds, _currentSlot)
	if err != nil {
		srv.log.Error("error getting bid delivery score report")
	}

	srv.statusHTMLData.Relays = _relays
	srv.statusHTMLData.FaultStatsReport = _faultStatsReport
	srv.statusHTMLData.FaultRecordsReport = _faultRecordsReport
	srv.statusHTMLData.ReputationScoreReport = _reputationScoreReport
	srv.statusHTMLData.BidDeliveryScoreReport = _bidDeliveryScoreReport

	srv.statusHTMLData.CountValidators = uint64(_countValidators)
	srv.statusHTMLData.CountValidatorsRegistrations = uint64(_countValidatorsRegistrations)
	srv.statusHTMLData.CountBidsAnalyzed = uint64(_countBidsAnalyzed)
	srv.statusHTMLData.CountBidsAnalyzedValid = uint64(_countBidsAnalyzedValid)
	srv.statusHTMLData.CountBidsAnalyzedFault = uint64(_countBidsAnalyzedFault)

	srv.statusHTMLData.HeadSlot = uint64(_currentSlot)

	// Now generate the HTML
	htmlDefault := bytes.Buffer{}

	// default view
	if err := srv.indexTemplate.Execute(&htmlDefault, srv.statusHTMLData); err != nil {
		srv.log.Error("error rendering template")
	}

	// Minify
	htmlDefaultBytes, err := srv.minifier.Bytes("text/html", htmlDefault.Bytes())
	if err != nil {
		srv.log.Error("error minifying htmlDefault")
	}

	// Swap the html pointers
	srv.rootResponseLock.Lock()
	srv.htmlDefault = &htmlDefaultBytes
	srv.rootResponseLock.Unlock()
}

func (srv *Webserver) handleRoot(w http.ResponseWriter, req *http.Request) {
	var err error

	srv.rootResponseLock.RLock()
	defer srv.rootResponseLock.RUnlock()
	_, err = w.Write(*srv.htmlDefault)
	if err != nil {
		srv.log.Error("error writing template")
	}
}
