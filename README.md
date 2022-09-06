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

### GET `/eth/v1/relay-monitor/faults`

Exposes a summary of faults per relay.

This response contains a map of relay public key to a mapping of observed fault counts.

The types of faults and their meaning can be found here: https://hackmd.io/A2uex3QFSfiaJJ9BKxw-XA?view#behavior-faults

Example response:

```json
{
  "0x845bd072b7cd566f02faeb0a4033ce9399e42839ced64e8b2adcfc859ed1e8e1a5a293336a49feac6d9a5edb779be53a": {
    "valid_bids": 0,
    "malformed_bids": 0,
    "consensus_invalid_bids": 0,
    "payment_invalid_bids": 0,
    "nonconforming_bids": 0,
    "malformed_payloads": 0,
    "consensus_invalid_payloads": 0,
    "unavailable_payloads": 0
  }
}
```
