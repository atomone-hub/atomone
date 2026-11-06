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

// fastAggregateVerifyExcluding verifies that sig is a valid aggregate
// signature by the committee members whose keys are NOT in nonSigners, using
// the proof-of-possession ciphersuite. It starts from committeeAgg, the
// aggregate over the whole committee, and subtracts the non-signers, so with
// participation usually above 95% it decodes ~25 keys instead of ~490; key
// decoding dominates the verification cost. With pubkeys and the message
// hash in G1/G2 respectively, it checks
// e(committeeAgg - sum(nonSigners), H(msg)) * e(-G1_gen, sig) == 1.
//
// committeeAgg must be the aggregate of the full committee key set bound to
// stored state (for spectre, by the committee summary comparison); anything
// else verifies a different statement.
//
// Accumulation runs in Jacobian coordinates: affine Add pays a field
// inversion per key, AddMixed does not, and one conversion at the end is
// enough. The result is rejected at the identity: keys summing to it would
// verify the identity signature over any message, which the standard's key
// validation forbids.
func fastAggregateVerifyExcluding(committeeAgg []byte, nonSigners [][]byte, msg [32]byte, sig []byte) error {
	start, err := parsePubkey(committeeAgg)
	if err != nil {
		return errorsmod.Wrap(err, "committee aggregate pubkey")
	}
	if start.IsInfinity() {
		return errorsmod.Wrap(ErrBLSVerificationFailed, "committee aggregate pubkey is the identity")
	}

	var agg bls12381.G1Jac
	agg.FromAffine(&start)
	for i, pk := range nonSigners {
		p, err := parsePubkey(pk)
		if err != nil {
			return errorsmod.Wrapf(err, "pubkey at index %d", i)
		}
		p.Neg(&p)
		agg.AddMixed(&p)
	}
	var aggPk bls12381.G1Affine
	aggPk.FromJacobian(&agg)
	if aggPk.IsInfinity() {
		return errorsmod.Wrap(ErrBLSVerificationFailed, "aggregate of the signers is the identity")
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
