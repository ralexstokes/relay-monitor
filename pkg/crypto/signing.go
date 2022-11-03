package crypto

import (
	"fmt"

	ssz "github.com/ferranbt/fastssz"
	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/bls"
)

var (
	BuilderDomain = [32]byte{}
)

func init() {
	var err error

	if BuilderDomain, err = consensus.ComputeDomain(consensus.Domain{0x00, 0x00, 0x00, 0x01}, [4]byte{}, [32]byte{}); err != nil {
		panic(fmt.Errorf("BUG: failed to compute domain: %v", err))
	}
}

type HashTreeRoot interface {
	HashTreeRoot() ([32]byte, error)
}

func VerifySignature(pub *bls.PublicKey, obj HashTreeRoot, domain [32]byte, sigBytes []byte) (bool, error) {
	root, err := obj.HashTreeRoot()
	if err != nil {
		return false, err
	}

	rootToSign, err := ssz.HashWithDefaultHasher(&consensus.SigningData{
		ObjectRoot: root,
		Domain:     domain,
	})
	if err != nil {
		return false, err
	}

	sig := &bls.Signature{}
	if err := sig.Deserialize(sigBytes); err != nil {
		return false, err
	}
	correct := sig.VerifyByte(pub, rootToSign[:])
	return correct, nil
}
