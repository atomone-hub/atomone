package spectre

import (
	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"

	errorsmod "cosmossdk.io/errors"
)

// blsPubkeySize is the size of a compressed BLS public key (G1).
const blsPubkeySize = 48

// blsSignatureSize is the size of a compressed BLS signature (G2).
const blsSignatureSize = 96

// dstSyncCommittee is the BLS ciphersuite domain separation tag used by the
// Ethereum beacon chain for sync committee signatures (hash-to-G2 with the
// proof-of-possession scheme, minimal pubkey size).
const dstSyncCommittee = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_"

// parsePubkey deserializes a compressed G1 public key with a subgroup check.
func parsePubkey(pk []byte) (bls12381.G1Affine, error) {
	if len(pk) != blsPubkeySize {
		return bls12381.G1Affine{}, errorsmod.Wrapf(ErrInvalidSyncCommittee,
			"invalid pubkey length %d, expected %d", len(pk), blsPubkeySize)
	}
	var p bls12381.G1Affine
	if _, err := p.SetBytes(pk); err != nil {
		return bls12381.G1Affine{}, errorsmod.Wrap(ErrInvalidSyncCommittee, "invalid pubkey encoding")
	}
	return p, nil
}

// parseSignature deserializes a compressed G2 signature with a subgroup check.
func parseSignature(sig []byte) (bls12381.G2Affine, error) {
	if len(sig) != blsSignatureSize {
		return bls12381.G2Affine{}, errorsmod.Wrapf(ErrBLSVerificationFailed,
			"invalid signature length %d, expected %d", len(sig), blsSignatureSize)
	}
	var s bls12381.G2Affine
	if _, err := s.SetBytes(sig); err != nil {
		return bls12381.G2Affine{}, errorsmod.Wrap(ErrBLSVerificationFailed, "invalid signature encoding")
	}
	return s, nil
}

// aggregatePubkeys sums the compressed public keys into their aggregate
// public key, returning the serialized aggregate.
func aggregatePubkeys(pubkeys [][]byte) ([blsPubkeySize]byte, error) {
	var agg bls12381.G1Affine
	for i, pk := range pubkeys {
		p, err := parsePubkey(pk)
		if err != nil {
			return [blsPubkeySize]byte{}, errorsmod.Wrapf(err, "pubkey at index %d", i)
		}
		var sum bls12381.G1Affine
		sum.Add(&agg, &p)
		agg.Set(&sum)
	}
	if len(pubkeys) == 0 {
		agg.SetInfinity()
	}
	return agg.Bytes(), nil
}

// fastAggregateVerify verifies that the signature is a valid aggregate
// signature by the public keys over the 32-byte message, using the
// proof-of-possession ciphersuite. With pubkeys and the message hash in G1/G2
// respectively, it checks e(pk_agg, H(msg)) * e(-G1_gen, sig) == 1.
func fastAggregateVerify(pubkeys [][]byte, msg [32]byte, sig []byte) error {
	if len(pubkeys) == 0 {
		return errorsmod.Wrap(ErrBLSVerificationFailed, "no public keys to verify")
	}

	var aggPk bls12381.G1Affine
	for i, pk := range pubkeys {
		p, err := parsePubkey(pk)
		if err != nil {
			return errorsmod.Wrapf(err, "pubkey at index %d", i)
		}
		var sum bls12381.G1Affine
		sum.Add(&aggPk, &p)
		aggPk.Set(&sum)
	}

	signature, err := parseSignature(sig)
	if err != nil {
		return err
	}

	hashedMsg, err := bls12381.HashToG2(msg[:], []byte(dstSyncCommittee))
	if err != nil {
		return errorsmod.Wrap(ErrBLSVerificationFailed, "hash to G2 failed")
	}

	_, _, g1GenAff, _ := bls12381.Generators()
	var negG1Gen bls12381.G1Affine
	negG1Gen.Neg(&g1GenAff)

	ok, err := bls12381.PairingCheck(
		[]bls12381.G1Affine{aggPk, negG1Gen},
		[]bls12381.G2Affine{hashedMsg, signature},
	)
	if err != nil {
		return errorsmod.Wrap(ErrBLSVerificationFailed, err.Error())
	}
	if !ok {
		return errorsmod.Wrap(ErrBLSVerificationFailed, "aggregate signature verification failed")
	}
	return nil
}
