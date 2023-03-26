# relay-monitor

A service to monitor a set of relayers in the external builder network on Ethereum. Implements the [Relay Monitor API Spec](https://github.com/andriidski/relay-monitor-specs)

## NOTE

Work in progress. Not ready for use yet.

## Dependencies

Go v1.19+

Requires a consensus client that implements the **dev** version of the standard [beacon node APIs](https://ethereum.github.io/beacon-APIs). This includes the standard set and [the RANDAO endpoint](https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Beacon/getStateRandao).

Every other required information is given in the configuration, e.g. `config.example.yaml`.

## Operation

`$ go run ./cmd/relay-monitor/main.go -config config.example.yaml`

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

See the [definition from the builder specs](https://ethereum.github.io/builder-specs/#/Builder/registerValidator) for more information.

### POST `/monitor/v1/transcript`

Accept complete transcripts from proposers to verify the proposer's leg of the auction was performed correctly.

This endpoint accepts the JSON encoding of the signed builder bid under a top-level key `"bid"` and the signed blinded beacon block under a top-level key `"acceptance"`. Encodings follow the JSON definition given in the [builder-specs](https://github.com/ethereum/builder-specs).

This endpoint returns HTTP 200 OK upon success and HTTP 4XX otherwise.

Example request:

```json
{
  "bid": {
    "data": {
      "message": {
        "header": {
          "parent_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc09",
          "state_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "receipts_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "logs_bloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
          "prev_randao": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "block_number": "1",
          "gas_limit": "1",
          "gas_used": "1",
          "timestamp": "1",
          "extra_data": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "base_fee_per_gas": "1",
          "block_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "transactions_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
        },
        "value": "1",
        "pubkey": "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
      },
      "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
    }
  },
  "acceptance": {
    "message": {
      "slot": "1",
      "proposer_index": "1",
      "parent_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "state_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "body": {
        ... fields omitted ...
        "execution_payload_header": {
          "parent_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc09",
          "state_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "receipts_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "logs_bloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
          "prev_randao": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "block_number": "1",
          "gas_limit": "1",
          "gas_used": "1",
          "timestamp": "1",
          "extra_data": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "base_fee_per_gas": "1",
          "block_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "transactions_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
        }
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
}
```
