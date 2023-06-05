package crypto_test

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	boostTypes "github.com/flashbots/go-boost-utils/ssz"
	"github.com/ralexstokes/relay-monitor/pkg/crypto"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

const (
	sepoliaSignedValidatorRegistration string = `{"message":{"fee_recipient":"0x1268ad189526ac0b386faf06effc46779c340ee6","gas_limit":"30000000","timestamp":"1667839752","pubkey":"0xb01a30d439def99e676c097e5f4b2aa249aa4d184eaace81819a698cb37d33f5a24089339916ee0acb539f0e62936d83"},"signature":"0xa7000ac532da1f072964c11a058857e3aea87ba29b54ecd743a6e2a46bb05912463cfceace236c37d63b01b7094a0fcc01fbedede160068bceea9c72a77914635c46e810e0e161ebc84f98267c31efab7f920af96442a3cf216935577a77f5b2"}`
)

var (
	sepoliaGenesisForkVersion         [4]byte = [4]byte{0x90, 0x00, 0x00, 0x69}
	sepoliaGenesisForkVersionAsNumber uint32  = 2415919209
)

const (
	bidMessage string = `{"message": {"header": {"parent_hash": "0xa1acb27555a9b11da48fa774c05bcc305ea4fb3519adc30647cb1740fa26dabd", "fee_recipient": "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5", "state_root": "0xefa75ce3912f721a17c51f2e09e5cdc6cff9280988d459ec1578c59fa5660791", "receipts_root": "0x80d84eb225cafc7784fb4c8c5331c4e2524781e8475bc05a5857556ed2c2ead7", "logs_bloom": "0xd021d86a4192d109878b4ae08250714d1258014ea6110d0802d96680279c9d10e487988a819ab3e702613b11d42b19237eb985608f13b98370160e4541ff4b6140cd83841cbc983de819714da2b451f66d930a4151da9e344c05ba0690741a384203ab818ea0d98b1dc9f00bb20b7c525eed45d53d431d2cd2f67bd58c0e0650963d7df94140955c23c904628d9fc656202c40a9434139c8a66188486b549c62266d3976deb8211b155254c0829617f84504e4045894414d4366af6fd6898037261816024a2b244d0996a48c1c3d5c747140a00a84a7ae78cc83890b180161600e1a6cd04c935106ba0612889605b8b40103ea38c183004202e83631821314df", "prev_randao": "0x9536bcc1b72cb4d3e889c669a06aea0ee5aabd2c33d24fc30bf638662d8ac3a6", "block_number": "17415288", "gas_limit": "30000000", "gas_used": "23810195", "timestamp": "1685979947", "extra_data": "0x6265617665726275696c642e6f7267", "base_fee_per_gas": "65067320965", "block_hash": "0x2368b0283ef2cf084571086a8d8bf819a23e17602f3df4c2a34b0508bccf5789", "transactions_root": "0x7a78af7f6fb3b18a41090bec358cdbc9838093a6405b8560d885e621f04b3a62", "withdrawals_root": "0xcf4a07f18729ffdb784e9b7e8425b21addb7642be2dc4a1e6b8238081753e27c"}, "value": "193637766565243815", "pubkey": "0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae"}, "signature": "0x995f29ae4c8045ecd6810bf24138297acc157c26185eb729f69655780e6c09a93e3bfea1ed2b12bc93ff4f693f7180ec0b9470c9acd2525c0b61cc3518033921268400326780d48679fe0ca9012f5eb33b91633a430ccca7e58ceb02677051fe"}`
)

var (
	capellaForkVersionMainnet         [4]byte = [4]byte{0x03, 0x00, 0x00, 0x00}
	capellaForkVersionMainnetAsNumber uint32  = 50331648
)

func TestCanComputeDomain(t *testing.T) {
	genesisForkVersion := [4]byte{}
	binary.BigEndian.PutUint32(genesisForkVersion[0:4], sepoliaGenesisForkVersionAsNumber)
	domain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, types.Root{})
	correctDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, sepoliaGenesisForkVersion, types.Root{})
	if domain != correctDomain {
		t.Fatal("could not compute correct domain")
	}
}

func TestSignatureVerification(t *testing.T) {
	var registration types.SignedValidatorRegistration
	err := json.Unmarshal([]byte(sepoliaSignedValidatorRegistration), &registration)
	if err != nil {
		t.Fatal(err)
	}

	genesisForkVersion := [4]byte{}
	binary.BigEndian.PutUint32(genesisForkVersion[0:4], sepoliaGenesisForkVersionAsNumber)
	domain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, types.Root{})

	valid, err := crypto.VerifySignature(registration.Message, domain, registration.Message.Pubkey[:], registration.Signature[:])
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("signature did not verify")
	}
}

func TestCanComputeDomainCapella(t *testing.T) {
	genesisForkVersion := [4]byte{}
	binary.BigEndian.PutUint32(genesisForkVersion[0:4], capellaForkVersionMainnetAsNumber)
	domain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, types.Root{})
	correctDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, capellaForkVersionMainnet, types.Root{})
	if domain != correctDomain {
		t.Fatal("could not compute correct domain")
	}
}

func TestBidSignature(t *testing.T) {
	var bid types.Bid
	err := json.Unmarshal([]byte(bidMessage), &bid)
	if err != nil {
		t.Fatal(err)
	}
	genesisForkVersion := [4]byte{0x00, 0x00, 0x00, 0x00}
	domain := crypto.Domain(boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, types.Root{}))
	valid, err := crypto.VerifySignature(bid.Message, domain, bid.Message.Pubkey[:], bid.Signature[:])
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("signature did not verify")
	}
}
