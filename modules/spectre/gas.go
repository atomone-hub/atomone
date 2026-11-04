package spectre

import (
	"math"
)

// Gas metering of client message verification. The gas a MsgUpdateClient pays
// by message size does not cover the verification work: one committee
// verification is up to SyncCommitteeSize BLS public key parses with subgroup
// checks, an aggregation, a hash-to-G2 and a pairing check, paid on every
// CheckTx, DeliverTx and mempool recheck. Unmetered, sustained update spam
// buys seconds of block time per block chain-wide. The cost is therefore
// charged upfront, before any BLS work runs, proportional to the client's
// committee size.
//
// The charge counts full passes over the committee: a Header performs one
// signature verification pass, a Misbehaviour two (one per update).
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
// Header: one signature verification pass.
func (cs *ClientState) GasCostHeaderVerification() uint64 {
	return cs.gasCostVerification(1)
}

// GasCostMisbehaviourVerification returns the gas charged before verifying a
// Misbehaviour: one signature verification pass per update.
func (cs *ClientState) GasCostMisbehaviourVerification() uint64 {
	return cs.gasCostVerification(2)
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
