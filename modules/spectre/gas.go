package spectre

import (
	"math"
)

// Gas metering of client message verification. The gas a MsgUpdateClient
// pays by message size does not cover the verification work: one committee
// verification is SyncCommitteeSize BLS public key parses with subgroup
// checks, an aggregation, a hash-to-G2 and a pairing check, paid on every
// CheckTx, DeliverTx and mempool recheck. Unmetered, sustained update spam
// buys seconds of block time per block chain-wide. The cost is therefore
// charged upfront, before any BLS work runs, proportional to the client's
// committee size.
//
// The charge counts full passes over the committee: a Header performs one
// committee aggregation and one signature verification, a Misbehaviour one
// aggregation and two signature verifications. The aggregation is cached by
// pubkeys hash, but the charge must not depend on node-local cache state:
// an attacker with self-created clients can sustain cache misses, and
// charging the uncached work is what bounds that.
const (
	// GasCostBlsPass is the fixed part charged per full pass over the sync
	// committee: SSZ hashing of the headers, the merkle branch walks,
	// hash-to-G2 and the pairing check.
	GasCostBlsPass uint64 = 100_000

	// GasCostPerPubkey is charged per committee public key of a pass: one G1
	// parse with subgroup checks and one aggregation step.
	GasCostPerPubkey uint64 = 5_000
)

// GasCostHeaderVerification returns the gas charged before verifying a
// Header: one committee aggregation pass and one signature verification
// pass.
func (cs *ClientState) GasCostHeaderVerification() uint64 {
	return cs.gasCostVerification(2)
}

// GasCostMisbehaviourVerification returns the gas charged before verifying a
// Misbehaviour: one committee aggregation pass and two signature
// verification passes.
func (cs *ClientState) GasCostMisbehaviourVerification() uint64 {
	return cs.gasCostVerification(3)
}

// gasCostVerification computes passes*(GasCostBlsPass +
// GasCostPerPubkey*SyncCommitteeSize), saturating at math.MaxUint64 instead
// of overflowing: an absurd committee size must charge unaffordable gas, not
// wrap around to cheap gas.
func (cs *ClientState) gasCostVerification(passes uint64) uint64 {
	if cs.SyncCommitteeSize > (math.MaxUint64-GasCostBlsPass)/GasCostPerPubkey {
		return math.MaxUint64
	}
	perPass := GasCostBlsPass + GasCostPerPubkey*cs.SyncCommitteeSize
	total := perPass * passes
	if total/perPass != passes {
		return math.MaxUint64
	}
	return total
}
