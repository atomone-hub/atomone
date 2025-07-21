package types

// Event types for the coredaos module
const (
	EventTypeAnnotateProposal   = "annotate_proposal"
	EventTypeEndorseProposal    = "endorse_proposal"
	EventTypeExtendVotingPeriod = "extend_voting_period"
	EventTypeVetoProposal       = "veto_propos al"

	AttributeKeyProposalID    = "proposal_id"
	AttributeKeySigner        = "signer"
	AttributeKeyNewEndTime    = "new_end_time"
	AttributeKeyTimesExtended = "times_extended"

	AttributeValueProposalVetoed = "proposal_vetoed"
)
