package spectre

import (
	"math"
)

// Gas metering of client message verification. The gas a MsgUpdateClient pays
// by message size does not cover the verification work: the dominant cost is
// BLS public key decoding, one parse with subgroup checks per committee key
// the verifier touches, plus a hash-to-G2 and a pairing per signature check.
// Unmetered, sustained update spam buys seconds of block time per block
// chain-wide. The cost is therefore charged upfront, before any BLS work
// runs.
//
// The verifier starts from the committee aggregate and subtracts the
// non-signers, so the keys it decodes are exactly the members whose bits are
// clear: the charge follows that count, which the message itself fixes. An
// honest update with participation above 95% pays the base plus a handful of
// keys; an attacker clearing every bit pays for the full committee. A
// Misbehaviour carries two updates and is charged for both.
const (
	// GasCostBlsCheck is the fixed part charged per signature check: SSZ
	// hashing of the headers, the merkle branch walks, hash-to-G2 and the
	// pairing check.
	GasCostBlsCheck uint64 = 100_000

	// GasCostPerPubkey is charged per committee public key the verifier
	// decodes: one G1 parse with subgroup checks and one addition.
	GasCostPerPubkey uint64 = 5_000
)

// nonSigners counts the committee members whose bits are clear, from the
// bitfield and the client's committee size. Padding bits beyond the size are
// not members; verification rejects them separately.
func nonSigners(bits []byte, committeeSize uint64) uint64 {
	return committeeSize - (&SyncAggregate{SyncCommitteeBits: bits}).numParticipants(committeeSize)
}

// GasCostHeaderVerification returns the gas charged before verifying a
// Header: one signature check plus one key decode per non-signer.
func (cs *ClientState) GasCostHeaderVerification(bits []byte) uint64 {
	return cs.gasCostVerification(1, nonSigners(bits, cs.SyncCommitteeSize))
}

// GasCostMisbehaviourVerification returns the gas charged before verifying a
// Misbehaviour: one signature check per update plus one key decode per
// non-signer of either update.
func (cs *ClientState) GasCostMisbehaviourVerification(bits1, bits2 []byte) uint64 {
	non := nonSigners(bits1, cs.SyncCommitteeSize) + nonSigners(bits2, cs.SyncCommitteeSize)
	return cs.gasCostVerification(2, non)
}

// gasCostVerification computes checks*GasCostBlsCheck +
// GasCostPerPubkey*keys, saturating at math.MaxUint64 instead of overflowing:
// an absurd committee size must charge unaffordable gas, not wrap around to
// cheap gas.
func (cs *ClientState) gasCostVerification(checks, keys uint64) uint64 {
	if checks > math.MaxUint64/GasCostBlsCheck {
		return math.MaxUint64
	}
	if keys > (math.MaxUint64-GasCostBlsCheck*checks)/GasCostPerPubkey {
		return math.MaxUint64
	}
	return GasCostBlsCheck*checks + GasCostPerPubkey*keys
}
