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

func TestCanComputeDomain(t *testing.T) {
	genesisForkVersion := [4]byte{}
	binary.BigEndian.PutUint32(genesisForkVersion[0:4], sepoliaGenesisForkVersionAsNumber)
	domain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, boostTypes.Root{})
	correctDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, sepoliaGenesisForkVersion, boostTypes.Root{})
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
	domain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, boostTypes.Root{})

	valid, err := crypto.VerifySignature(registration.Message, domain, registration.Message.Pubkey[:], registration.Signature[:])
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("signature did not verify")
	}
}
