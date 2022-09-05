# relay-monitor

A service to monitor a set of relayers in the external builder network on Ethereum.

## NOTE

Work in progress. Not ready for use yet.

## Implementation

The monitor is structured as a series of components that ingest data and produce a live stream of fault data for each configured relay.

### `collector`

The `collector` gathers data from either connected validators or relays it is configured to scan.

- validator registrations
- bids from relays
- auction transcript (bid + signed blinded beacon block) from connected proposers
- execution payload (conditional on auction transcript)
- signed beacon block (collected from a consensus client)

### `analyzer`

The `analyzer` component derives faults from the collected data.

Each slot of the consensus protocol has a unique analysis which tries to derive any configured faults based on the observed data at a given time.

It is defined as set of rules (defined in Go for now) that should be re-evaluated when any dependent piece of data is collected.

The current fault count for each relay summarizing their behavior is maintained by the monitor and exposed via API.

## API docs

### POST `/eth/v1/builder/validators`

Expose the `registerValidator` endpoint from the `builder-specs` APIs to accept `SignedValidatorRegistrationsV1` from connected proposers.

### POST `/eth/v1/relay-monitor/transcript`

Accept complete transcripts from proposers to verify the proposer's leg of the auction was performed correctly.

Definition of an auction "transcript":

```python
class AuctionTranscript(Container):
    bid: SignedBuilderBid
    acceptance: SignedBlindedBeaconBlock
```

and transcripts are signed by a given proposer to verify their authenticity.

```python
class SignedAuctionTranscript(Container):
    transcript: AuctionTranscript
    signature: BLSSignature
```

### GET `/eth/v1/relay-monitor/faults`

Exposes a summary of faults per relay.
