package consensus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	eth2Api "github.com/attestantio/go-eth2-client/api"
	eth2HttpApi "github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/common/math"
	lru "github.com/hashicorp/golang-lru"
	"github.com/holiman/uint256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/protolambda/eth2api"
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
	client *eth2HttpApi.Service

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
	denebForkVersion      types.ForkVersion
	denebForkEpoch        types.Epoch

	builderSignatureDomain *crypto.Domain

	// slot -> ValidatorInfo
	proposerCache *lru.Cache
	// slot -> SignedBeaconBlock
	blockCache *lru.Cache
	// blockNumber -> slot
	blockNumberToSlotIndex *lru.Cache
	validatorLock          sync.RWMutex
	// publicKey -> Validator
	validatorCache map[types.PublicKey]*types.ValidatorResponse
	// validatorIndex -> publicKey, note: points into `validatorCache`
	validatorIndexCache map[types.ValidatorIndex]*types.PublicKey
}

func NewClient(ctx context.Context, endpoint string, logger *zap.Logger) (*Client, error) {
	eth2Service, err := eth2HttpApi.New(ctx, eth2HttpApi.WithTimeout(clientTimeoutSec*time.Second), eth2HttpApi.WithAddress(endpoint))
	if err != nil {
		return nil, err
	}
	eth2Client, ok := eth2Service.(*eth2HttpApi.Service)
	if !ok {
		return nil, fmt.Errorf("could not cast eth2 service to http service")
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

	validatorCache := make(map[types.PublicKey]*types.ValidatorResponse)
	validatorIndexCache := make(map[types.ValidatorIndex]*types.PublicKey)

	client := &Client{
		logger:                 logger,
		client:                 eth2Client,
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
		err := c.FetchBlock(ctx, currentSlot-types.Slot(i))
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

	rsp, err := c.client.Genesis(ctx, &eth2Api.GenesisOpts{})
	if err != nil {
		return err
	}

	c.GenesisTime = uint64(rsp.Data.GenesisTime.Unix())
	c.genesisForkVersion = types.ForkVersion(rsp.Data.GenesisForkVersion)
	c.GenesisValidatorsRoot = types.Root(rsp.Data.GenesisValidatorsRoot)

	return nil
}

func (c *Client) fetchSpec(ctx context.Context) error {
	rsp, err := c.client.Spec(ctx, &eth2Api.SpecOpts{})
	if err != nil {
		return err
	}

	c.SlotsPerEpoch = rsp.Data["SLOTS_PER_EPOCH"].(uint64)
	c.SecondsPerSlot = uint64(rsp.Data["SECONDS_PER_SLOT"].(time.Duration).Seconds())
	c.altairForkVersion = rsp.Data["ALTAIR_FORK_VERSION"].(types.ForkVersion)
	c.altairForkEpoch = rsp.Data["ALTAIR_FORK_EPOCH"].(types.Epoch)
	c.bellatrixForkVersion = rsp.Data["BELLATRIX_FORK_VERSION"].(types.ForkVersion)
	c.bellatrixForkEpoch = rsp.Data["BELLATRIX_FORK_EPOCH"].(types.Epoch)
	c.capellaForkVersion = rsp.Data["CAPELLA_FORK_VERSION"].(types.ForkVersion)
	c.capellaForkEpoch = rsp.Data["CAPELLA_FORK_EPOCH"].(types.Epoch)
	c.denebForkVersion = rsp.Data["DENEB_FORK_VERSION"].(types.ForkVersion)
	c.denebForkEpoch = rsp.Data["DENEB_FORK_EPOCH"].(types.Epoch)

	return nil
}

// NOTE: this assumes the fork schedule is presented in ascending order
func (c *Client) GetForkVersion(slot types.Slot) types.ForkVersion {
	epoch := uint64(slot) / c.SlotsPerEpoch
	if epoch >= c.denebForkEpoch {
		return c.denebForkVersion
	} else if epoch >= c.capellaForkEpoch {
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
	val, ok := c.proposerCache.Get(uint64(slot))
	if !ok {
		return nil, fmt.Errorf("could not find proposer for slot %d", slot)
	}
	validator, ok := val.(ValidatorInfo)
	if !ok {
		return nil, fmt.Errorf("internal: proposer cache contains an unexpected type %T", val)
	}
	return &validator, nil
}

func (c *Client) GetBlock(slot types.Slot) (*types.VersionedSignedBeaconBlock, error) {
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
	block, ok := val.(*types.VersionedSignedBeaconBlock)
	if !ok {
		return nil, fmt.Errorf("internal: block cache contains an unexpected value %v with type %T", val, val)
	}
	return block, nil
}

func (c *Client) GetValidator(publicKey *types.PublicKey) (*types.ValidatorResponse, error) {
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

		validatorRsp, err := c.client.Validators(context.Background(), &eth2Api.ValidatorsOpts{PubKeys: []phase0.BLSPubKey{phase0.BLSPubKey(*publicKey)}})
		if err != nil {
			return nil, err
		}

		for k, v := range validatorRsp.Data {
			publicKey := v.Validator.PublicKey
			key := types.PublicKey(publicKey)

			validator = &types.ValidatorResponse{
				Index:     k,
				Balance:   v.Validator.EffectiveBalance,
				Validator: *v.Validator,
				Status:    types.ValidatorStatus(v.Status.String()),
			}

			c.validatorCache[key] = validator
			c.validatorIndexCache[uint64(k)] = &key
		}
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

	return block.BlockHash()
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
	syncState, err := c.client.NodeSyncing(ctx, &eth2Api.NodeSyncingOpts{})
	if err != nil {
		return err
	}

	if syncState.Data.IsSyncing {
		return fmt.Errorf("could not fetch proposal duties in epoch %d because node is syncing", epoch)
	}

	proposers, err := c.client.ProposerDuties(ctx, &eth2Api.ProposerDutiesOpts{Epoch: phase0.Epoch(epoch)})
	if err != nil {
		return err
	}

	for _, duty := range proposers.Data {
		c.proposerCache.Add(uint64(duty.Slot), ValidatorInfo{
			publicKey: types.PublicKey(duty.PubKey),
			index:     uint64(duty.ValidatorIndex),
		})
	}

	return nil
}

func (c *Client) FetchBlockRequest(ctx context.Context, slot types.Slot) (*spec.VersionedSignedBeaconBlock, error) {
	blockID := eth2api.BlockIdSlot(slot)
	block, err := c.client.SignedBeaconBlock(ctx, &eth2Api.SignedBeaconBlockOpts{Block: blockID.BlockId()})
	if err != nil {
		return nil, err
	}

	return block.Data, nil
}

func (c *Client) RetryBlockRequest(ctx context.Context, slot types.Slot) (*spec.VersionedSignedBeaconBlock, error) {
	// Retry previous slot 5 times
	logger := c.logger.Sugar()

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Second * 12
	b.InitialInterval = time.Second
	b.Reset()

	var block *spec.VersionedSignedBeaconBlock
	var err error

	err = backoff.Retry(func() error {
		logger.Warnf("could not find slot: %d. Retrying in %vs.", slot, b.NextBackOff().Seconds())
		block, err = c.FetchBlockRequest(ctx, slot)
		if err != nil {
			return err
		}
		return nil
	}, b)

	if err == nil {
		return block, nil
	}

	// Try 3 previous slots
	for i := 1; i < 4; i++ {
		targetSlot := slot - types.Slot(i)
		logger.Warnf("could not find slot: %d. Retrying with previous slot %d", slot, targetSlot)
		block, err = c.FetchBlockRequest(ctx, targetSlot)
		if err == nil {
			return block, nil
		}
	}
	logger.Errorf("all block requests have failed starting at slot %d", slot)
	return nil, errors.New("all block requests have failed")
}

func (c *Client) FetchBlock(ctx context.Context, slot types.Slot) error {
	// TODO handle reorgs, etc.
	var signedBeaconBlock *spec.VersionedSignedBeaconBlock
	var err error

	signedBeaconBlock, err = c.FetchBlockRequest(ctx, slot)
	if err != nil {
		signedBeaconBlock, err = c.RetryBlockRequest(ctx, slot)
		if err != nil {
			return err
		}
	}

	block := &types.VersionedSignedBeaconBlock{
		VersionedSignedBeaconBlock: *signedBeaconBlock,
	}

	c.blockCache.Add(slot, block)

	blockNumber, err := signedBeaconBlock.ExecutionBlockNumber()
	if err != nil {
		return err
	}

	c.blockNumberToSlotIndex.Add(blockNumber, slot)
	return nil
}

type headEvent struct {
	Slot  string     `json:"slot"`
	Block types.Root `json:"block"`
}

func (c *Client) StreamHeads(ctx context.Context) <-chan types.Coordinate {
	logger := c.logger.Sugar()

	sseClient := sse.NewClient(c.client.Address() + "/eth/v1/events?topics=head")
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

	// TODO support branches w/ proposer public key
	// TODO pipe in context
	// TODO or consider getting for each head and caching locally...

	apiRsp, err := c.client.BeaconStateRandao(context.Background(), &eth2Api.BeaconStateRandaoOpts{State: fmt.Sprintf("%d", uint64(slot))})
	if err != nil {
		return types.Hash{}, nil
	}

	return phase0.Hash32(*apiRsp.Data), nil
}

func (c *Client) GetBlockNumberForProposal(slot types.Slot /*, proposerPublicKey *types.PublicKey */) (uint64, error) {
	// TODO support branches w/ proposer public key
	parentBlock, err := c.GetBlock(slot - 1)
	if err != nil {
		return 0, err
	}

	blockNumber, err := parentBlock.ExecutionBlockNumber()
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

	parentGasLimit, err := parentBlock.GasLimit()
	if err != nil {
		return nil, err
	}

	parentGasUsed, err := parentBlock.GasUsed()
	if err != nil {
		return nil, err
	}

	parentBaseFee, err := parentBlock.BaseFeePerGas()
	if err != nil {
		return nil, err
	}

	parentGasTarget := parentGasLimit / GasElasticityMultiplier
	return computeBaseFee(parentGasTarget, parentGasUsed, parentBaseFee), nil
}

func (c *Client) GetParentGasLimit(ctx context.Context, blockNumber uint64) (uint64, error) {
	// TODO support branches w/ proposer public key
	slotValue, ok := c.blockNumberToSlotIndex.Get(blockNumber)
	if !ok {
		return 0, fmt.Errorf("missing block for block number %d", blockNumber)
	}
	slot, ok := slotValue.(phase0.Slot)
	if !ok {
		return 0, fmt.Errorf("internal: unexpected type %T in block number to slot index", slotValue)
	}
	parentBlock, err := c.GetBlock(slot - types.Slot(1))
	if err != nil {
		return 0, err
	}
	return parentBlock.GasLimit()
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
