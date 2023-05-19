package api

import (
	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/reporter"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

type Server struct {
	config *Config
	logger *zap.SugaredLogger

	analyzer        *analysis.Analyzer
	events          chan<- data.Event
	clock           *consensus.Clock
	store           store.Storer
	reporter        *reporter.Reporter
	consensusClient *consensus.Client
}

type Config struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type Span struct {
	Start types.Epoch `json:"start_epoch,string"`
	End   types.Epoch `json:"end_epoch,string"`
}

type ScoreReportResponse struct {
	SlotBounds types.SlotBounds  `json:"slot_bounds"`
	Report     types.ScoreReport `json:"report"`
}

type ScoreReponse struct {
	SlotBounds types.SlotBounds `json:"slot_bounds"`
	Score      types.Score      `json:"score"`
}

type FaultStatsResponse struct {
	SlotBounds types.SlotBounds `json:"slot_bounds"`
	Data       types.FaultStats `json:"data"`
}

type FaultRecordsResponse struct {
	SlotBounds types.SlotBounds   `json:"slot_bounds"`
	Data       types.FaultRecords `json:"data"`
}

type FaultStatsReportResponse struct {
	SlotBounds types.SlotBounds       `json:"slot_bounds"`
	Data       types.FaultStatsReport `json:"data"`
}

type FaultRecordsReportResponse struct {
	SlotBounds types.SlotBounds         `json:"slot_bounds"`
	Data       types.FaultRecordsReport `json:"data"`
}

type CountResponse struct {
	Count uint `json:"count"`
}
