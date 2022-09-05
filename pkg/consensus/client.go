package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/protolambda/eth2api"
	"github.com/protolambda/eth2api/client/beaconapi"
	"github.com/protolambda/eth2api/client/validatorapi"
	"github.com/protolambda/zrnt/eth2/beacon/bellatrix"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	consensustypes "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/r3labs/sse/v2"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const clientTimeoutSec = 5

type ValidatorInfo struct {
	publicKey types.PublicKey
	index     types.ValidatorIndex
}

type Client struct {
	logger      *zap.Logger
	client      *eth2api.Eth2HttpClient
	genesisTime time.Time
	clock       *slots.SlotTicker
	epochClock  *helpers.EpochTicker

	proposerCache      map[consensustypes.Slot]ValidatorInfo
	proposerCacheMutex sync.RWMutex

	executionCache      map[consensustypes.Slot]types.Hash
	executionCacheMutex sync.RWMutex
}

func NewClient(endpoint string, clock *slots.SlotTicker, epochClock *helpers.EpochTicker, genesisTime time.Time, logger *zap.Logger) *Client {
	client := &eth2api.Eth2HttpClient{
		Addr: endpoint,
		Cli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 128,
			},
			Timeout: clientTimeoutSec * time.Second,
		},
		Codec: eth2api.JSONCodec{},
	}
	return &Client{
		logger:         logger,
		client:         client,
		genesisTime:    genesisTime,
		clock:          clock,
		epochClock:     epochClock,
		proposerCache:  make(map[consensustypes.Slot]ValidatorInfo),
		executionCache: make(map[consensustypes.Slot]types.Hash),
	}
}

func (c *Client) GetParentHash(slot consensustypes.Slot) (types.Hash, error) {
	targetSlot := slot - 1
	c.executionCacheMutex.RLock()
	parentHash, ok := c.executionCache[targetSlot]
	c.executionCacheMutex.RUnlock()
	if !ok {
		return c.fetchExecutionHash(targetSlot)
	}
	return parentHash, nil
}

func (c *Client) GetProposerPublicKey(slot consensustypes.Slot) (*types.PublicKey, error) {
	c.proposerCacheMutex.RLock()
	defer c.proposerCacheMutex.RUnlock()

	validator, ok := c.proposerCache[slot]
	if !ok {
		return nil, fmt.Errorf("missing proposal for slot %d", slot)
	}

	return &validator.publicKey, nil
}

func (c *Client) fetchProposers(epoch consensustypes.Epoch) error {
	ctx := context.Background()

	var proposerDuties eth2api.DependentProposerDuty
	syncing, err := validatorapi.ProposerDuties(ctx, c.client, common.Epoch(epoch), &proposerDuties)
	if syncing {
		return fmt.Errorf("could not fetch proposal duties in epoch %d because node is syncing", epoch)
	} else if err != nil {
		return err
	}

	// TODO handle reorgs, etc.
	c.proposerCacheMutex.Lock()
	for _, duty := range proposerDuties.Data {
		c.proposerCache[consensustypes.Slot(duty.Slot)] = ValidatorInfo{
			publicKey: types.PublicKey(duty.Pubkey),
			index:     uint64(duty.ValidatorIndex),
		}
	}
	c.proposerCacheMutex.Unlock()

	return nil
}

func (c *Client) loadData(epoch consensustypes.Epoch) error {
	err := c.fetchProposers(epoch)
	if err != nil {
		return err
	}

	return nil
}

type headEvent struct {
	Slot  string     `json:"slot"`
	Block types.Root `json:"block"`
}

func (c *Client) streamHeads() <-chan types.Coordinate {
	logger := c.logger.Sugar()

	sseClient := sse.NewClient(c.client.Addr + "/eth/v1/events?topics=head")
	ch := make(chan types.Coordinate, 1)
	go func() {
		err := sseClient.SubscribeRaw(func(msg *sse.Event) {
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
				Slot: consensustypes.Slot(slot),
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

func (c *Client) fetchExecutionHash(slot consensustypes.Slot) (types.Hash, error) {
	ctx := context.Background()

	blockID := eth2api.BlockIdSlot(slot)

	var signedBeaconBlock eth2api.VersionedSignedBeaconBlock
	exists, err := beaconapi.BlockV2(ctx, c.client, blockID, &signedBeaconBlock)
	if !exists {
		// assume missing slot, use execution hash from previous slot(s)
		for i := slot; i > 0; i-- {
			targetSlot := i - 1
			c.executionCacheMutex.RLock()
			executionHash, ok := c.executionCache[targetSlot]
			c.executionCacheMutex.RUnlock()
			if !ok {
				continue
			}
			return executionHash, nil
		}
		return types.Hash{}, fmt.Errorf("block at slot %d is missing", slot)
	} else if err != nil {
		return types.Hash{}, err
	}

	bellatrixBlock, ok := signedBeaconBlock.Data.(*bellatrix.SignedBeaconBlock)
	if !ok {
		return types.Hash{}, fmt.Errorf("could not parse block %s", signedBeaconBlock)
	}
	executionHash := types.Hash(bellatrixBlock.Message.Body.ExecutionPayload.BlockHash)

	// TODO handle reorgs, etc.
	c.executionCacheMutex.Lock()
	c.executionCache[slot] = executionHash
	c.executionCacheMutex.Unlock()

	return executionHash, nil
}

func (c *Client) runSlotTasks(wg *sync.WaitGroup) {
	logger := c.logger.Sugar()

	// load data for the previous slot
	currentSlot := slots.SinceGenesis(c.genesisTime)
	_, err := c.fetchExecutionHash(currentSlot - 1)
	if err != nil {
		logger.Warnf("could not fetch latest execution hash for slot %d: %v", currentSlot, err)
	}

	// load data for the current slot
	_, err = c.fetchExecutionHash(currentSlot)
	if err != nil {
		logger.Warnf("could not fetch latest execution hash for slot %d: %v", currentSlot, err)
	}
	// done with init...
	wg.Done()

	for head := range c.streamHeads() {
		_, err := c.fetchExecutionHash(head.Slot)
		if err != nil {
			logger.Warnf("could not fetch latest execution hash for slot %d: %v", head.Slot, err)
		}
	}
}

func (c *Client) runEpochTasks(wg *sync.WaitGroup) {
	logger := c.logger.Sugar()

	for epoch := range c.epochClock.C() {
		// load data for the current epoch
		err := c.loadData(consensustypes.Epoch(epoch))
		if err != nil {
			logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
		}

		// load data for the next epoch, as we will typically do
		err = c.loadData(consensustypes.Epoch(epoch) + 1)
		if err != nil {
			logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
		}
		// signal to caller that we have done the initialization...
		wg.Done()
	}
}

func (c *Client) Run(wg *sync.WaitGroup) {
	wg.Add(1)
	go c.runSlotTasks(wg)

	wg.Add(1)
	go c.runEpochTasks(wg)

	wg.Done()
}
