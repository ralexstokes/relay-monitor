package consensus

import (
	"context"
	"errors"
	"fmt"
	"strings"

	lru "github.com/hashicorp/golang-lru"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"

	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/http"
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
	client *http.Client

	// slot -> ValidatorInfo
	proposerCache *lru.Cache
	// slot -> Hash
	executionCache *lru.Cache
	// publicKey -> Validator
	validatorCache *lru.Cache
}

func NewClient(ctx context.Context, endpoint string, logger *zap.Logger, currentSlot types.Slot, currentEpoch types.Epoch, slotsPerEpoch uint64) (*Client, error) {
	proposerCache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	executionCache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	validatorCache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	client := &Client{
		logger:         logger,
		proposerCache:  proposerCache,
		executionCache: executionCache,
		validatorCache: validatorCache,
	}

	err = client.loadCurrentContext(ctx, currentSlot, currentEpoch, slotsPerEpoch)
	if err != nil {
		logger := logger.Sugar()
		logger.Warn("could not load the current context from the consensus client")
	}

	return client, nil
}

func (c *Client) loadCurrentContext(ctx context.Context, currentSlot types.Slot, currentEpoch types.Epoch, slotsPerEpoch uint64) error {
	logger := c.logger.Sugar()

	var baseSlot uint64
	if currentSlot > slotsPerEpoch {
		baseSlot = currentSlot - slotsPerEpoch
	}

	for i := baseSlot; i < slotsPerEpoch; i++ {
		_, err := c.FetchExecutionHash(ctx, i)
		if err != nil {
			logger.Warnf("could not fetch latest execution hash for slot %d: %v", currentSlot, err)
		}
	}

	err := c.FetchProposers(ctx, currentEpoch)
	if err != nil {
		logger.Warnf("could not load consensus state for epoch %d: %v", currentEpoch, err)
	}

	nextEpoch := currentEpoch + 1
	err = c.FetchProposers(ctx, nextEpoch)
	if err != nil {
		logger.Warnf("could not load consensus state for epoch %d: %v", nextEpoch, err)
	}

	err = c.FetchValidators(ctx)
	if err != nil {
		logger.Warnf("could not load validators: %v", err)
	}

	return nil
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

func (c *Client) GetExecutionHash(slot types.Slot) (types.Hash, error) {
	val, ok := c.executionCache.Get(slot)
	if !ok {
		return types.Hash{}, fmt.Errorf("could not find execution hash for slot %d", slot)
	}
	hash, ok := val.(types.Hash)
	if !ok {
		return types.Hash{}, fmt.Errorf("internal: execution cache contains an unexpected type %T", val)
	}
	return hash, nil
}

func (c *Client) GetValidator(publicKey *types.PublicKey) (*http.Validator, error) {
	val, ok := c.validatorCache.Get(publicKey)
	if !ok {
		return nil, fmt.Errorf("missing validator entry for public key %s", publicKey)
	}
	validator, ok := val.(*http.Validator)
	if !ok {
		return nil, fmt.Errorf("internal: validator cache contains an unexpected type %T", val)
	}
	return validator, nil
}

func (c *Client) GetParentHash(ctx context.Context, slot types.Slot) (types.Hash, error) {
	targetSlot := slot - 1
	parentHash, err := c.GetExecutionHash(targetSlot)
	if err != nil {
		return c.FetchExecutionHash(ctx, targetSlot)
	}
	return parentHash, nil
}

func (c *Client) GetProposerPublicKey(ctx context.Context, slot types.Slot) (*types.PublicKey, error) {
	validator, err := c.GetProposer(slot)
	if err != nil {
		// TODO consider fallback to grab the assignments for the missing epoch...
		return nil, fmt.Errorf("missing proposer for slot %d: %v", slot, err)
	}
	return &validator.publicKey, nil
}

func (c *Client) FetchProposers(ctx context.Context, epoch types.Epoch) error {
	duties, err := c.client.Validator().GetProposerDuties(epoch)
	if err != nil {
		return err
	}

	// TODO handle reorgs, etc.
	for _, duty := range duties {
		c.proposerCache.Add(uint64(duty.Slot), ValidatorInfo{
			// publicKey: types.PublicKey(duty.Pubkey), TODO
			index: uint64(duty.ValidatorIndex),
		})
	}

	return nil
}

func (c *Client) backFillExecutionHash(slot types.Slot) (types.Hash, error) {
	for i := slot; i > 0; i-- {
		targetSlot := i - 1
		executionHash, err := c.GetExecutionHash(targetSlot)
		if err == nil {
			for i := targetSlot; i < slot; i++ {
				c.executionCache.Add(i+1, executionHash)
			}
			return executionHash, nil
		}
	}
	return types.Hash{}, fmt.Errorf("no execution hashes present before %d (inclusive)", slot)
}

func (c *Client) FetchExecutionHash(ctx context.Context, slot types.Slot) (types.Hash, error) {
	// TODO handle reorgs, etc.
	executionHash, err := c.GetExecutionHash(slot)
	if err == nil {
		return executionHash, nil
	}

	var bellatrixBlock consensus.BeaconBlockBellatrix
	if _, err := c.client.Beacon().GetBlock(http.Slot(slot), &bellatrixBlock); err != nil {
		if errors.Is(err, http.ErrorNotFound) {
			// TODO move search to `GetParentHash`
			// TODO also instantiate with first execution hash...
			return c.backFillExecutionHash(slot) // TODO
		}
		return types.Hash{}, nil
	}

	executionHash = types.Hash(bellatrixBlock.Body.ExecutionPayload.BlockHash)

	// TODO handle reorgs, etc.
	c.executionCache.Add(slot, executionHash)

	return executionHash, nil
}

func (c *Client) StreamHeads(ctx context.Context) <-chan types.Coordinate {
	ch := make(chan types.Coordinate, 1)
	c.client.Events(ctx, []string{"head"}, func(obj interface{}) {
		event := obj.(*http.HeadEvent)

		head := types.Coordinate{
			Slot: event.Slot,
			Root: event.Block,
		}
		ch <- head
	})
	return ch
}

// TODO handle reorgs
func (c *Client) FetchValidators(ctx context.Context) error {
	validators, err := c.client.Beacon().GetValidators(http.Head)
	if err != nil {
		return err
	}
	for _, validator := range validators {
		publicKey := validator.Validator.PubKey
		c.validatorCache.Add(publicKey, validator)
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
