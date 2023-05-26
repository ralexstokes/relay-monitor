package crypto_test

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	boostTypes "github.com/flashbots/go-boost-utils/types"
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
	bidMessage string = `{"message":{"header":{"parent_hash":"0x18979f800e1b67ee1d6d29f08bb316badb2bb301f02f6611c241bcd4665e0a9f","fee_recipient":"0x690b9a9e9aa1c9db991c7721a92d351db4fac990","state_root":"0x5e267303781b42d6b7d35c7259db3d73e339408c2b165bbf822d099b277f8361","receipts_root":"0x43b3e500905209cddb14484d5e2acd070206e6a2e4064c4d36e284a1ff208028","logs_bloom":"0x0c773e8341e580eed462188abb72c3208316c69000aa607aa2a39652e480021036bff705003a41a805901b22db84c5571e115654da707dce6f1017156a78a734ca03483af3e548ef7eabe81bc3d0a9faac820e90b9c11af81994cc14dc819b219bc201039f0e8f4c24407cb06c42acd15928b4540cfaefa5eeeb0eba589b5bc8e7a70ff11224d8697c7adfc109820837ea8082a37b9c461c7bad554339735db19278090f33b9648756135cc0f052c60ed6620f9911ce88135327aab6177a6942c188a70254a46ef20562de322d492b4f41296a54a4e8cb7b0f87b5120d99a25cc1f1a8cf818cb1322f351e8e9e29ba472e9ca1b6d8f208494096fa48ba13d780","prev_randao":"0xfe30a5fabf7b11dc38827260ab9817be2f211e705178e71cae54657706f79fb0","block_number":"17344122","gas_limit":"30000000","gas_used":"14353928","timestamp":"1685114483","extra_data":"0x627920406275696c64657230783639","base_fee_per_gas":"32970824166","block_hash":"0x5564410419c9f38d53a2e2b08411c31b951b502352fdf44a44abce6469f186d5","transactions_root":"0x4df13964c141d866463a14ccbc992771e14650714a747700f2f720dcfb72c9c9"},"value": "458425265650592382","pubkey": "0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae"}}`
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
	genesisForkVersion := [4]byte{}
	binary.BigEndian.PutUint32(genesisForkVersion[0:4], capellaForkVersionMainnetAsNumber)
	domain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, types.Root{})
	valid, err := crypto.VerifySignature(bid.Message, domain, bid.Message.Pubkey[:], bid.Signature[:])
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("signature did not verify")
	}
}
