package consensus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/common/math"
	lru "github.com/hashicorp/golang-lru"
	"github.com/holiman/uint256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/protolambda/eth2api"
	"github.com/protolambda/eth2api/client/beaconapi"
	"github.com/protolambda/eth2api/client/configapi"
	"github.com/protolambda/eth2api/client/validatorapi"
	"github.com/protolambda/zrnt/eth2/beacon/bellatrix"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/r3labs/sse/v2"
	"github.com/ralexstokes/relay-monitor/pkg/crypto"
	"github.com/ralexstokes/relay-monitor/pkg/metrics"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const (
	clientTimeoutSec                = 30
	cacheSize                       = 1024
	GasElasticityMultiplier         = 2
	BaseFeeChangeDenominator uint64 = 8
)

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
)

type ValidatorInfo struct {
	publicKey types.PublicKey
	index     types.ValidatorIndex
}

type Client struct {
	logger *zap.Logger
	client *eth2api.Eth2HttpClient

	SlotsPerEpoch         uint64
	SecondsPerSlot        uint64
	GenesisTime           uint64
	genesisForkVersion    types.ForkVersion
	GenesisValidatorsRoot types.Root
	altairForkVersion     types.ForkVersion
	altairForkEpoch       types.Epoch
	bellatrixForkVersion  types.ForkVersion
	bellatrixForkEpoch    types.Epoch
	capellaForkVersion    types.ForkVersion
	capellaForkEpoch      types.Epoch

	builderSignatureDomain *crypto.Domain

	// slot -> ValidatorInfo
	proposerCache *lru.Cache
	// slot -> SignedBeaconBlock
	blockCache *lru.Cache
	// blockNumber -> slot
	blockNumberToSlotIndex *lru.Cache
	validatorLock          sync.RWMutex
	// publicKey -> Validator
	validatorCache map[types.PublicKey]*eth2api.ValidatorResponse
	// validatorIndex -> publicKey, note: points into `validatorCache`
	validatorIndexCache map[types.ValidatorIndex]*types.PublicKey
}

func NewClient(ctx context.Context, endpoint string, logger *zap.Logger) (*Client, error) {
	httpClient := &eth2api.Eth2HttpClient{
		Addr: endpoint,
		Cli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 128,
			},
			Timeout: clientTimeoutSec * time.Second,
		},
		Codec: eth2api.JSONCodec{},
	}

	proposerCache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	blockCache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	blockNumberToSlotIndex, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	validatorCache := make(map[types.PublicKey]*eth2api.ValidatorResponse)
	validatorIndexCache := make(map[types.ValidatorIndex]*types.PublicKey)

	client := &Client{
		logger:                 logger,
		client:                 httpClient,
		proposerCache:          proposerCache,
		blockCache:             blockCache,
		blockNumberToSlotIndex: blockNumberToSlotIndex,
		validatorCache:         validatorCache,
		validatorIndexCache:    validatorIndexCache,
	}

	err = client.FetchGenesis(ctx)
	if err != nil {
		logger := client.logger.Sugar()
		logger.Fatalf("could not load genesis info: %v", err)
	}

	err = client.fetchSpec(ctx)
	if err != nil {
		logger := client.logger.Sugar()
		logger.Fatalf("could not load spec configuration: %v", err)
	}

	return client, nil
}

func (c *Client) SignatureDomainForBuilder() crypto.Domain {
	if c.builderSignatureDomain == nil {
		domain := crypto.Domain(crypto.ComputeDomain(crypto.DomainTypeAppBuilder, c.genesisForkVersion, types.Root{}))
		c.builderSignatureDomain = &domain
	}
	return *c.builderSignatureDomain
}

func (c *Client) SignatureDomain(slot types.Slot) crypto.Domain {
	forkVersion := c.GetForkVersion(slot)
	return crypto.ComputeDomain(crypto.DomainTypeBeaconProposer, forkVersion, c.GenesisValidatorsRoot)
}

func (c *Client) LoadCurrentContext(ctx context.Context, currentSlot types.Slot, currentEpoch types.Epoch) error {
	logger := c.logger.Sugar()

	for i := uint64(0); i < c.SlotsPerEpoch; i++ {
		err := c.FetchBlock(ctx, currentSlot-i)
		if err != nil {
			logger.Warnf("could not fetch latest block for slot %d: %v", currentSlot, err)
		}
	}

	// Start with default values for now, may need to update
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Second * 60
	b.Reset()

	err := backoff.Retry(func() error {
		err := c.FetchProposers(ctx, currentEpoch)
		if err != nil {
			logger.Info("could not load proposers, backing off an retrying")
		}
		return err
	}, b)
	if err != nil {
		return fmt.Errorf("could not load proposers: %v", err)
	}

	b.Reset()

	nextEpoch := currentEpoch + 1
	err = backoff.Retry(func() error {
		err = c.FetchProposers(ctx, nextEpoch)
		if err != nil {
			logger.Infof("could not load consensus state for epoch %d: %v", nextEpoch, err)
			return err
		}
		return err
	}, b)
	if err != nil {
		return fmt.Errorf("could not load next epoch proposers: %v", err)
	}
	b.Reset()

	return nil
}

func (c *Client) FetchGenesis(ctx context.Context) error {
	var resp eth2api.GenesisResponse
	exists, err := beaconapi.Genesis(ctx, c.client, &resp)
	if !exists {
		return fmt.Errorf("genesis information does not exist")
	}
	if err != nil {
		return err
	}

	c.GenesisTime = uint64(resp.GenesisTime)
	c.genesisForkVersion = types.ForkVersion(resp.GenesisForkVersion)
	c.GenesisValidatorsRoot = types.Hash(resp.GenesisValidatorsRoot)
	return nil
}

func (c *Client) fetchSpec(ctx context.Context) error {
	var spec common.Spec
	err := configapi.Spec(ctx, c.client, &spec)
	if err != nil {
		return err
	}

	c.SlotsPerEpoch = uint64(spec.Phase0Preset.SLOTS_PER_EPOCH)
	c.SecondsPerSlot = uint64(spec.Config.SECONDS_PER_SLOT)
	c.altairForkVersion = types.ForkVersion(spec.Config.ALTAIR_FORK_VERSION)
	c.altairForkEpoch = types.Epoch(spec.Config.ALTAIR_FORK_EPOCH)
	c.bellatrixForkVersion = types.ForkVersion(spec.Config.BELLATRIX_FORK_VERSION)
	c.bellatrixForkEpoch = types.Epoch(spec.Config.BELLATRIX_FORK_EPOCH)
	c.capellaForkVersion = types.ForkVersion(spec.Config.CAPELLA_FORK_VERSION)
	c.capellaForkEpoch = types.Epoch(spec.Config.CAPELLA_FORK_EPOCH)
	return nil
}

// NOTE: this assumes the fork schedule is presented in ascending order
func (c *Client) GetForkVersion(slot types.Slot) types.ForkVersion {
	epoch := slot / c.SlotsPerEpoch
	if epoch >= c.capellaForkEpoch {
		return c.capellaForkVersion
	} else if epoch >= c.bellatrixForkEpoch {
		return c.bellatrixForkVersion
	} else if epoch >= c.altairForkEpoch {
		return c.altairForkVersion
	} else {
		return c.genesisForkVersion
	}
}

func (c *Client) GetProposer(slot types.Slot) (*ValidatorInfo, error) {
	val, ok := c.proposerCache.Get(slot)
	if !ok {
		return nil, fmt.Errorf("could not find proposer for slot %d", slot)
	}
	validator, ok := val.(ValidatorInfo)
	if !ok {
		return nil, fmt.Errorf("internal: proposer cache contains an unexpected type %T", val)
	}
	return &validator, nil
}

func (c *Client) GetBlock(slot types.Slot) (eth2api.SignedBeaconBlock, error) {
	val, ok := c.blockCache.Get(slot)
	if !ok {
		// TODO pipe in context
		err := c.FetchBlock(context.Background(), slot)
		if err != nil {
			return nil, err
		}
		val, ok = c.blockCache.Get(slot)
		if !ok {
			return nil, fmt.Errorf("could not find block for slot %d", slot)
		}
	}
	block, ok := val.(eth2api.SignedBeaconBlock)
	if !ok {
		return nil, fmt.Errorf("internal: block cache contains an unexpected value %v with type %T", val, val)
	}
	return block, nil
}

func (c *Client) GetValidator(publicKey *types.PublicKey) (*eth2api.ValidatorResponse, error) {
	c.validatorLock.RLock()
	defer c.validatorLock.RUnlock()

	validator, ok := c.validatorCache[*publicKey]
	if !ok {

		pubKeys, _ := publicKey.MarshalText()
		var x common.BLSPubkey
		err := x.UnmarshalText(pubKeys)
		if err != nil {
			return nil, err
		}

		filter := eth2api.ValidatorIdPubkey(x)
		exists, err := beaconapi.StateValidator(context.Background(), c.client, eth2api.StateHead, filter, validator)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("could not fetch validators from remote endpoint because they do not exist")
		}

		publicKey := validator.Validator.Pubkey
		key := types.PublicKey(publicKey)
		c.validatorCache[key] = validator
		c.validatorIndexCache[uint64(validator.Index)] = &key
	}
	return validator, nil
}

func (c *Client) GetParentHash(ctx context.Context, slot types.Slot) (types.Hash, error) {

	t := prometheus.NewTimer(metrics.GetParentHash)
	defer t.ObserveDuration()

	targetSlot := slot - 1
	block, err := c.GetBlock(targetSlot)
	if err != nil {
		return types.Hash{}, err
	}
	return c.getBlockHash(block)
}

func (c *Client) GetProposerPublicKey(ctx context.Context, slot types.Slot) (*types.PublicKey, error) {

	t := prometheus.NewTimer(metrics.GetProposerPubKey)
	defer t.ObserveDuration()

	validator, err := c.GetProposer(slot)
	if err != nil {
		// TODO consider fallback to grab the assignments for the missing epoch...
		return nil, fmt.Errorf("missing proposer for slot %d: %v", slot, err)
	}
	return &validator.publicKey, nil
}

func (c *Client) FetchProposers(ctx context.Context, epoch types.Epoch) error {
	var proposerDuties eth2api.DependentProposerDuty
	syncing, err := validatorapi.ProposerDuties(ctx, c.client, common.Epoch(epoch), &proposerDuties)
	if syncing {
		return fmt.Errorf("could not fetch proposal duties in epoch %d because node is syncing", epoch)
	} else if err != nil {
		return err
	}

	// TODO handle reorgs, etc.
	for _, duty := range proposerDuties.Data {
		c.proposerCache.Add(uint64(duty.Slot), ValidatorInfo{
			publicKey: types.PublicKey(duty.Pubkey),
			index:     uint64(duty.ValidatorIndex),
		})
	}

	return nil
}

func (c *Client) FetchBlockRequest(ctx context.Context, slot types.Slot, dest *eth2api.VersionedSignedBeaconBlock) (bool, error) {
	blockID := eth2api.BlockIdSlot(slot)
	exists, err := beaconapi.BlockV2(ctx, c.client, blockID, dest)
	return exists, err
}

func (c *Client) RetryBlockRequest(ctx context.Context, slot types.Slot, dest *eth2api.VersionedSignedBeaconBlock) error {
	// Retry previous slot 5 times
	logger := c.logger.Sugar()

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Second * 12
	b.InitialInterval = time.Second
	b.Reset()

	err := backoff.Retry(func() error {
		logger.Warnf("could not find slot: %d. Retrying in %vs.", slot, b.NextBackOff().Seconds())
		exists, err := c.FetchBlockRequest(ctx, slot, dest)
		if err != nil {
			return err
		}

		if exists {
			return nil
		} else {
			return fmt.Errorf("could not find block for slot %d", slot)
		}
	}, b)

	if err == nil {
		return nil
	}

	// Try 3 previous slots
	for i := 1; i < 4; i++ {
		targetSlot := slot - uint64(i)
		logger.Warnf("could not find slot: %d. Retrying with previous slot %d", slot, targetSlot)
		exists, err := c.FetchBlockRequest(ctx, targetSlot, dest)
		if exists && err == nil {
			return err
		}
	}
	logger.Errorf("all block requests have failed starting at slot %d", slot)
	return errors.New("all block requests have failed")
}

func (c *Client) FetchBlock(ctx context.Context, slot types.Slot) error {
	// TODO handle reorgs, etc.
	var signedBeaconBlock eth2api.VersionedSignedBeaconBlock
	exists, err := c.FetchBlockRequest(ctx, slot, &signedBeaconBlock)
	// NOTE: need to check `exists` first...
	if !exists {
		err := c.RetryBlockRequest(ctx, slot, &signedBeaconBlock)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	c.blockCache.Add(slot, signedBeaconBlock.Data)

	blockNumber, err := c.getBlockNumber(signedBeaconBlock.Data)
	if err != nil {
		return err
	}

	c.blockNumberToSlotIndex.Add(blockNumber, slot)
	return nil
}

func (c *Client) getBlockNumber(signedBlock eth2api.SignedBeaconBlock) (uint64, error) {
	var blockNumber uint64
	switch block := signedBlock.(type) {
	case *bellatrix.SignedBeaconBlock:
		blockNumber = uint64(block.Message.Body.ExecutionPayload.BlockNumber)
	case *capella.SignedBeaconBlock:
		blockNumber = uint64(block.Message.Body.ExecutionPayload.BlockNumber)
	default:
		return 0, fmt.Errorf("unexpected block type %T", block)
	}

	return blockNumber, nil
}

func (c *Client) getBlockHash(signedBlock eth2api.SignedBeaconBlock) (types.Hash, error) {
	var blockHash types.Hash
	switch block := signedBlock.(type) {
	case *bellatrix.SignedBeaconBlock:
		blockHash = types.Hash(block.Message.Body.ExecutionPayload.BlockHash)
	case *capella.SignedBeaconBlock:
		blockHash = types.Hash(block.Message.Body.ExecutionPayload.BlockHash)
	default:
		return types.Hash{}, fmt.Errorf("unexpected block type %T", block)
	}

	return blockHash, nil
}

type GasDetails struct {
	GasLimit      uint64
	GasUsed       uint64
	BaseFeePerGas uint256.Int
}

func (c *Client) getGasDetails(versionedBlock eth2api.SignedBeaconBlock) (*GasDetails, error) {
	gasDetails := &GasDetails{}
	switch block := versionedBlock.(type) {
	case *bellatrix.SignedBeaconBlock:
		gasDetails.GasLimit = uint64(block.Message.Body.ExecutionPayload.GasLimit)
		gasDetails.GasUsed = uint64(block.Message.Body.ExecutionPayload.GasUsed)
		gasDetails.BaseFeePerGas = (uint256.Int)(block.Message.Body.ExecutionPayload.BaseFeePerGas)
	case *capella.SignedBeaconBlock:
		gasDetails.GasLimit = uint64(block.Message.Body.ExecutionPayload.GasLimit)
		gasDetails.GasUsed = uint64(block.Message.Body.ExecutionPayload.GasUsed)
		gasDetails.BaseFeePerGas = (uint256.Int)(block.Message.Body.ExecutionPayload.BaseFeePerGas)
	default:
		return nil, fmt.Errorf("unexpected block type %T", block)
	}

	return gasDetails, nil
}

type headEvent struct {
	Slot  string     `json:"slot"`
	Block types.Root `json:"block"`
}

func (c *Client) StreamHeads(ctx context.Context) <-chan types.Coordinate {
	logger := c.logger.Sugar()

	sseClient := sse.NewClient(c.client.Addr + "/eth/v1/events?topics=head")
	ch := make(chan types.Coordinate, 1)
	go func() {
		err := sseClient.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
			var event headEvent
			err := json.Unmarshal(msg.Data, &event)
			if err != nil {
				logger.Warnf("could not unmarshal `head` node event: %v", err)
				return
			}
			slot, err := strconv.Atoi(event.Slot)
			if err != nil {
				logger.Warnf("could not unmarshal slot from `head` node event: %v", err)
				return
			}
			head := types.Coordinate{
				Slot: types.Slot(slot),
				Root: event.Block,
			}
			ch <- head
		})
		if err != nil {
			logger.Errorw("could not subscribe to head event", "error", err)
		}
	}()
	return ch
}

// TODO handle reorgs
func (c *Client) FetchValidators(ctx context.Context, validatorIds []eth2api.ValidatorId, statusFilter []eth2api.ValidatorStatus) error {
	var response []eth2api.ValidatorResponse
	exists, err := beaconapi.StateValidators(ctx, c.client, eth2api.StateHead, validatorIds, statusFilter, &response)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("could not fetch validators from remote endpoint because they do not exist")
	}

	c.validatorLock.Lock()
	defer c.validatorLock.Unlock()

	for _, validator := range response {
		publicKey := validator.Validator.Pubkey
		key := types.PublicKey(publicKey)
		c.validatorCache[key] = &validator
		c.validatorIndexCache[uint64(validator.Index)] = &key
	}

	return nil
}

func (c *Client) GetValidatorStatus(publicKey *types.PublicKey) (ValidatorStatus, error) {
	validator, err := c.GetValidator(publicKey)
	if err != nil {
		return StatusValidatorUnknown, err
	}
	validatorStatus := string(validator.Status)
	if strings.Contains(validatorStatus, "active") {
		return StatusValidatorActive, nil
	} else if strings.Contains(validatorStatus, "pending") {
		return StatusValidatorPending, nil
	} else {
		return StatusValidatorUnknown, nil
	}
}

func (c *Client) GetRandomnessForProposal(slot types.Slot /*, proposerPublicKey *types.PublicKey */) (types.Hash, error) {
	targetSlot := slot - 1
	// TODO support branches w/ proposer public key
	// TODO pipe in context
	// TODO or consider getting for each head and caching locally...
	return FetchRandao(context.Background(), c.client, targetSlot)
}

func (c *Client) GetBlockNumberForProposal(slot types.Slot /*, proposerPublicKey *types.PublicKey */) (uint64, error) {
	// TODO support branches w/ proposer public key
	parentBlock, err := c.GetBlock(slot - 1)
	if err != nil {
		return 0, err
	}

	blockNumber, err := c.getBlockNumber(parentBlock)
	if err != nil {
		return 0, err
	}

	return blockNumber + 1, nil
}

func computeBaseFee(parentGasTarget, parentGasUsed uint64, parentBaseFee *big.Int) *types.Uint256 {
	// NOTE: following the `geth` implementation here:
	result := uint256.NewInt(0)
	if parentGasUsed == parentGasTarget {
		result.SetFromBig(parentBaseFee)
		return result
	} else if parentGasUsed > parentGasTarget {
		x := big.NewInt(int64(parentGasUsed - parentGasTarget))
		y := big.NewInt(int64(parentGasTarget))
		x.Mul(x, parentBaseFee)
		x.Div(x, y)
		x.Div(x, y.SetUint64(BaseFeeChangeDenominator))
		baseFeeDelta := math.BigMax(x, bigOne)

		x = x.Add(parentBaseFee, baseFeeDelta)
		result.SetFromBig(x)
	} else {
		x := big.NewInt(int64(parentGasTarget - parentGasUsed))
		y := big.NewInt(int64(parentGasTarget))
		x.Mul(x, parentBaseFee)
		x.Div(x, y)
		x.Div(x, y.SetUint64(BaseFeeChangeDenominator))

		baseFee := x.Sub(parentBaseFee, x)
		result.SetFromBig(math.BigMax(baseFee, bigZero))
	}
	return result
}

func (c *Client) GetBaseFeeForProposal(slot types.Slot /*, proposerPublicKey *types.PublicKey */) (*types.Uint256, error) {
	// TODO support multiple branches of block tree
	parentBlock, err := c.GetBlock(slot - 1)
	if err != nil {
		return nil, err
	}

	parentGasDetails, err := c.getGasDetails(parentBlock)
	if err != nil {
		return nil, err
	}

	parentGasTarget := parentGasDetails.GasLimit / GasElasticityMultiplier
	return computeBaseFee(parentGasTarget, parentGasDetails.GasUsed, parentGasDetails.BaseFeePerGas.ToBig()), nil
}

func (c *Client) GetParentGasLimit(ctx context.Context, blockNumber uint64) (uint64, error) {
	// TODO support branches w/ proposer public key
	slotValue, ok := c.blockNumberToSlotIndex.Get(blockNumber)
	if !ok {
		return 0, fmt.Errorf("missing block for block number %d", blockNumber)
	}
	slot, ok := slotValue.(uint64)
	if !ok {
		return 0, fmt.Errorf("internal: unexpected type %T in block number to slot index", slotValue)
	}
	parentBlock, err := c.GetBlock(slot - 1)
	if err != nil {
		return 0, err
	}

	gasDetails, err := c.getGasDetails(parentBlock)
	if err != nil {
		return 0, err
	}

	return gasDetails.GasLimit, nil
}

func (c *Client) GetPublicKeyForIndex(ctx context.Context, validatorIndex types.ValidatorIndex) (*types.PublicKey, error) {
	c.validatorLock.RLock()
	defer c.validatorLock.RUnlock()

	key, ok := c.validatorIndexCache[validatorIndex]
	if !ok {
		// TODO consider fetching here if not in cache
		return nil, fmt.Errorf("could not find public key for validator index %d", validatorIndex)
	}
	return key, nil
}
