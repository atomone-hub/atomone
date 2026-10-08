package spectre

import (
	"github.com/ethereum/go-ethereum/crypto"

	errorsmod "cosmossdk.io/errors"
)

// evmCommitmentPath computes the storage key under which the EVM IBC router
// stores its commitments: keccak256(keccak256(path) || slot_be_bytes). It
// mirrors evm_ics26_commitment_path in the spectre ethereum light client.
func evmCommitmentPath(path []byte, slot [32]byte) [32]byte {
	pathHash := crypto.Keccak256(path)
	preimage := make([]byte, 0, 64)
	preimage = append(preimage, pathHash...)
	preimage = append(preimage, slot[:]...)
	var out [32]byte
	copy(out[:], crypto.Keccak256(preimage))
	return out
}

// checkCommitmentPath validates that the IBC merkle path addresses the
// commitment key proven by the storage proof.
func checkCommitmentPath(path [][]byte, slot [32]byte, key [32]byte) error {
	if len(path) != 1 {
		return errorsmod.Wrapf(ErrInvalidCommitmentPath, "expected path length 1, got %d", len(path))
	}
	expected := evmCommitmentPath(path[0], slot)
	if expected != key {
		return errorsmod.Wrapf(ErrInvalidCommitmentPath,
			"path derives commitment key %X but proof covers key %X", expected, key)
	}
	return nil
}

// verifyStorageMembership verifies a membership proof of value at path in
// the IBC router contract storage, against the consensus state at the proof
// height.
func verifyStorageMembership(cs *ClientState, consState *ConsensusState, proof *MembershipProof, path [][]byte, value []byte) error {
	if err := validateProofBounds(proof.AccountProof.Proof, proof.StorageProof.Proof); err != nil {
		return err
	}

	stateRoot := bytes32(consState.StateRoot)
	if err := verifyAccountStorageRoot(
		stateRoot,
		cs.IbcContractAddress,
		proof.AccountProof.Proof,
		bytes32(proof.AccountProof.StorageRoot),
	); err != nil {
		return errorsmod.Wrap(err, "account proof verification failed")
	}

	storageKey := bytes32(proof.StorageProof.Key)
	if err := checkCommitmentPath(path, bytes32(cs.IbcCommitmentSlot), storageKey); err != nil {
		return err
	}

	if !bytesEqual32(bytes32(proof.StorageProof.Value), bytes32(value)) {
		return errorsmod.Wrapf(ErrValueMismatch,
			"proven storage value %X does not match expected value %X", proof.StorageProof.Value, value)
	}

	return verifyStorageInclusionProof(
		bytes32(proof.AccountProof.StorageRoot),
		storageKey,
		bytes32(proof.StorageProof.Value),
		proof.StorageProof.Proof,
	)
}

// verifyStorageNonMembership verifies a non-membership proof: the value at
// path must be absent (zero) in the IBC router contract storage.
func verifyStorageNonMembership(cs *ClientState, consState *ConsensusState, proof *MembershipProof, path [][]byte) error {
	if err := validateProofBounds(proof.AccountProof.Proof, proof.StorageProof.Proof); err != nil {
		return err
	}

	stateRoot := bytes32(consState.StateRoot)
	if err := verifyAccountStorageRoot(
		stateRoot,
		cs.IbcContractAddress,
		proof.AccountProof.Proof,
		bytes32(proof.AccountProof.StorageRoot),
	); err != nil {
		return errorsmod.Wrap(err, "account proof verification failed")
	}

	storageKey := bytes32(proof.StorageProof.Key)
	if err := checkCommitmentPath(path, bytes32(cs.IbcCommitmentSlot), storageKey); err != nil {
		return err
	}

	var zero [32]byte
	if !bytesEqual32(bytes32(proof.StorageProof.Value), zero) {
		return errorsmod.Wrapf(ErrValueMismatch, "claimed value %X must be zero for non-membership", proof.StorageProof.Value)
	}

	return verifyStorageExclusionProof(
		bytes32(proof.AccountProof.StorageRoot),
		storageKey,
		proof.StorageProof.Proof,
	)
}

// bytes32 converts a byte slice to a fixed 32-byte array. Inputs of the
// wrong length yield the zero array, which then fails comparison against
// well-formed counterparties.
func bytes32(b []byte) [32]byte {
	var out [32]byte
	if len(b) == 32 {
		copy(out[:], b)
	}
	return out
}

func bytesEqual32(a, b [32]byte) bool {
	return a == b
}
