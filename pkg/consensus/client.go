package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/protolambda/eth2api"
	"github.com/protolambda/eth2api/client/beaconapi"
	"github.com/protolambda/eth2api/client/validatorapi"
	"github.com/protolambda/zrnt/eth2/beacon/bellatrix"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/r3labs/sse/v2"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const (
	clientTimeoutSec = 5
	cacheSize        = 128
)

type ValidatorInfo struct {
	publicKey types.PublicKey
	index     types.ValidatorIndex
}

type Client struct {
	logger *zap.Logger
	client *eth2api.Eth2HttpClient
	clock  *Clock

	proposerCache  lru.Cache
	executionCache lru.Cache
}

func NewClient(endpoint string, clock *Clock, logger *zap.Logger) (*Client, error) {
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
	proposerCache, err := lru.New(cacheSize)
	if err != nil {
		return &Client{}, err
	}
	executionHash, err := lru.New(cacheSize)
	if err != nil {
		return &Client{}, err
	}
	return &Client{
		logger:         logger,
		client:         client,
		clock:          clock,
		proposerCache:  *proposerCache,
		executionCache: *executionHash,
	}, nil
}

func (c *Client) GetParentHash(slot types.Slot) (types.Hash, error) {
	targetSlot := slot - 1
	parentHash, ok := c.executionCache.Get(targetSlot)
	if !ok {
		return c.fetchExecutionHash(targetSlot)
	}
	if hash, ok := parentHash.(types.Hash); ok {
		return hash, nil
	} else {
		return types.Hash{}, fmt.Errorf("invalid value stored in cache")
	}
}

func (c *Client) GetProposerPublicKey(slot types.Slot) (*types.PublicKey, error) {
	validator, ok := c.proposerCache.Get(slot)
	if !ok {
		return nil, fmt.Errorf("missing proposal for slot %d", slot)
	}

	if v, ok := validator.(ValidatorInfo); ok {
		return &v.publicKey, nil
	} else {
		return &types.PublicKey{}, fmt.Errorf("invalid value stored in cache")
	}
}

func (c *Client) fetchProposers(epoch types.Epoch) error {
	ctx := context.Background()

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

func (c *Client) loadData(epoch types.Epoch) error {
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

func (c *Client) fetchExecutionHash(slot types.Slot) (types.Hash, error) {
	ctx := context.Background()

	blockID := eth2api.BlockIdSlot(slot)

	var signedBeaconBlock eth2api.VersionedSignedBeaconBlock
	exists, err := beaconapi.BlockV2(ctx, c.client, blockID, &signedBeaconBlock)
	if !exists {
		// assume missing slot, use execution hash from previous slot(s)
		for i := slot; i > 0; i-- {
			targetSlot := i - 1
			executionHash, ok := c.executionCache.Get(targetSlot)
			if !ok {
				continue
			}
			if hash, ok := executionHash.(types.Hash); ok {
				return hash, nil
			} else {
				return types.Hash{}, fmt.Errorf("invalid value stored in cache")
			}
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
	c.executionCache.Add(slot, executionHash)

	return executionHash, nil
}

func (c *Client) runSlotTasks(wg *sync.WaitGroup) {
	logger := c.logger.Sugar()

	// load data for the previous slot
	now := time.Now().Unix()
	currentSlot := c.clock.currentSlot(now)
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

	epochs := c.clock.TickEpochs()

	// load data for the current epoch
	epoch := <-epochs
	err := c.loadData(epoch)
	if err != nil {
		logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
	}

	// load data for the next epoch, as we will typically do
	err = c.loadData(epoch + 1)
	if err != nil {
		logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
	}
	// signal to caller that we have done the initialization...
	wg.Done()

	for epoch := range epochs {
		err := c.loadData(epoch + 1)
		if err != nil {
			logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
		}
	}
}

func (c *Client) Run(wg *sync.WaitGroup) {
	wg.Add(1)
	go c.runSlotTasks(wg)

	wg.Add(1)
	go c.runEpochTasks(wg)

	wg.Done()
}
