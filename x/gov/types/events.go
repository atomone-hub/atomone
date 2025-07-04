package types

// Governance module event types
const (
	EventTypeSubmitProposal          = "submit_proposal"
	EventTypeProposalDeposit         = "proposal_deposit"
	EventTypeProposalVote            = "proposal_vote"
	EventTypeInactiveProposal        = "inactive_proposal"
	EventTypeActiveProposal          = "active_proposal"
	EventTypeSignalProposal          = "signal_proposal"
	EventTypeQuorumCheck             = "quorum_check"
	EventTypeMinDepositChange        = "min_deposit_change"
	EventTypeMinInitialDepositChange = "min_initial_deposit_change"

	AttributeKeyVoter                        = "voter"
	AttributeKeyProposalResult               = "proposal_result"
	AttributeKeyOption                       = "option"
	AttributeKeyProposalID                   = "proposal_id"
	AttributeKeyProposalMessages             = "proposal_messages" // Msg type_urls in the proposal
	AttributeKeyVotingPeriodStart            = "voting_period_start"
	AttributeValueProposalDropped            = "proposal_dropped"  // didn't meet min deposit
	AttributeValueProposalPassed             = "proposal_passed"   // met vote quorum
	AttributeValueProposalRejected           = "proposal_rejected" // didn't meet vote quorum
	AttributeValueProposalFailed             = "proposal_failed"   // error on proposal handler
	AttributeKeyProposalType                 = "proposal_type"
	AttributeSignalTitle                     = "signal_title"
	AttributeSignalDescription               = "signal_description"
	AttributeValueProposalQuorumMet          = "proposal_quorum_met"           // met quorum
	AttributeValueProposalQuorumNotMet       = "proposal_quorum_not_met"       // didn't meet quorum
	AttributeValueProposalQuorumCheckSkipped = "proposal_quorum_check_skipped" // skipped quorum check
	AttributeKeyNewMinDeposit                = "new_min_deposit"               // new min deposit value
	AttributeKeyLastMinDeposit               = "last_min_deposit"              // last min deposit value
	AttributeKeyNewMinInitialDeposit         = "new_min_initial_deposit"       // new min initial deposit value
	AttributeKeyLastMinInitialDeposit        = "last_min_initial_deposit"      // last min initial deposit value
)
