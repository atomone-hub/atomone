package v1

import (
	"bytes"
	"sort"
	"strings"

	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/x/gov/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	MaxMonikerLength         = 70
	MaxIdentityLength        = 3000
	MaxWebsiteLength         = 140
	MaxSecurityContactLength = 140
	MaxDetailsLength         = 280
)

var (
	GovernorStatusUnspecified = GovernorStatus_name[int32(Unspecified)]
	GovernorStatusActive      = GovernorStatus_name[int32(Active)]
	GovernorStatusInactive    = GovernorStatus_name[int32(Inactive)]
)

var _ GovernorI = Governor{}

// NewGovernor constructs a new Governor
func NewGovernor(address string, description GovernorDescription) (Governor, error) {
	return Governor{
		GovernorAddress: address,
		Description:     description,
		Status:          Active,
	}, nil
}

// Governors is a collection of Governor
type Governors struct {
	Governors     []Governor
	GovernorCodec address.Codec
}

func (g Governors) String() (out string) {
	for _, gov := range g.Governors {
		out += gov.String() + "\n"
	}

	return strings.TrimSpace(out)
}

// Sort Governors sorts governor array in ascending operator address order
func (g Governors) Sort() {
	sort.Sort(g)
}

// Implements sort interface
func (g Governors) Len() int {
	return len(g.Governors)
}

// Implements sort interface
func (g Governors) Less(i, j int) bool {
	gi, err := g.GovernorCodec.StringToBytes(g.Governors[i].GetAddress().String())
	if err != nil {
		panic(err)
	}
	gj, err := g.GovernorCodec.StringToBytes(g.Governors[j].GetAddress().String())
	if err != nil {
		panic(err)
	}

	return bytes.Compare(gi, gj) == -1
}

// Implements sort interface
func (g Governors) Swap(i, j int) {
	g.Governors[i], g.Governors[j] = g.Governors[j], g.Governors[i]
}

// GovernorsByVotingPower implements sort.Interface for []Governor based on
// the VotingPower and Address fields.
// The vovernors are sorted first by their voting power (descending). Secondary index - Address (ascending).
// Copied from tendermint/types/vovernor_set.go
type GovernorsByVotingPower []Governor

func (govs GovernorsByVotingPower) Len() int { return len(govs) }

func (govs GovernorsByVotingPower) Less(i, j int, r math.Int) bool {
	if valz[i].ConsensusPower(r) == valz[j].ConsensusPower(r) {
		addrI, errI := valz[i].GetConsAddr()
		addrJ, errJ := valz[j].GetConsAddr()
		// If either returns error, then return false
		if errI != nil || errJ != nil {
			return false
		}
		return bytes.Compare(addrI, addrJ) == -1
	}
	return valz[i].ConsensusPower(r) > valz[j].ConsensusPower(r)
}

func (govs GovernorsByVotingPower) Swap(i, j int) {
	govs[i], govs[j] = govs[j], govs[i]
}

func MustMarshalGovernor(cdc codec.BinaryCodec, governor *Governor) []byte {
	return cdc.MustMarshal(governor)
}

func MustUnmarshalGovernor(cdc codec.BinaryCodec, value []byte) Governor {
	governor, err := UnmarshalGovernor(cdc, value)
	if err != nil {
		panic(err)
	}

	return governor
}

// unmarshal a redelegation from a store value
func UnmarshalGovernor(cdc codec.BinaryCodec, value []byte) (g Governor, err error) {
	err = cdc.Unmarshal(value, &g)
	return g, err
}

// IsActive checks if the governor status equals Active
func (g Governor) IsActive() bool {
	return g.GetStatus() == Active
}

// IsInactive checks if the governor status equals Inactive
func (g Governor) IsInactive() bool {
	return g.GetStatus() == Inactive
}

// constant used in flags to indicate that description field should not be updated
const DoNotModifyDesc = "[do-not-modify]"

func NewGovernorDescription(moniker, identity, website, securityContact, details string) GovernorDescription {
	return GovernorDescription{
		Moniker:         moniker,
		Identity:        identity,
		Website:         website,
		SecurityContact: securityContact,
		Details:         details,
	}
}

// UpdateDescription updates the fields of a given description. An error is
// returned if the resulting description contains an invalid length.
func (d GovernorDescription) UpdateDescription(d2 GovernorDescription) (GovernorDescription, error) {
	if d2.Moniker == DoNotModifyDesc {
		d2.Moniker = d.Moniker
	}

	if d2.Identity == DoNotModifyDesc {
		d2.Identity = d.Identity
	}

	if d2.Website == DoNotModifyDesc {
		d2.Website = d.Website
	}

	if d2.SecurityContact == DoNotModifyDesc {
		d2.SecurityContact = d.SecurityContact
	}

	if d2.Details == DoNotModifyDesc {
		d2.Details = d.Details
	}

	return NewGovernorDescription(
		d2.Moniker,
		d2.Identity,
		d2.Website,
		d2.SecurityContact,
		d2.Details,
	).EnsureLength()
}

// EnsureLength ensures the length of a vovernor's description.
func (d GovernorDescription) EnsureLength() (GovernorDescription, error) {
	if len(d.Moniker) > MaxMonikerLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid moniker length; got: %d, max: %d", len(d.Moniker), MaxMonikerLength)
	}

	if len(d.Identity) > MaxIdentityLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid identity length; got: %d, max: %d", len(d.Identity), MaxIdentityLength)
	}

	if len(d.Website) > MaxWebsiteLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid website length; got: %d, max: %d", len(d.Website), MaxWebsiteLength)
	}

	if len(d.SecurityContact) > MaxSecurityContactLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid security contact length; got: %d, max: %d", len(d.SecurityContact), MaxSecurityContactLength)
	}

	if len(d.Details) > MaxDetailsLength {
		return d, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid details length; got: %d, max: %d", len(d.Details), MaxDetailsLength)
	}

	return d, nil
}

// MinEqual defines a more minimum set of equality conditions when comparing two
// governors.
func (g *Governor) MinEqual(other *Governor) bool {
	return g.GovernorAddress == other.GovernorAddress &&
		g.Status == other.Status &&
		g.Description.Equal(other.Description)
}

// Equal checks if the receiver equals the parameter
func (g *Governor) Equal(v2 *Governor) bool {
	return g.MinEqual(v2)
}

func (g Governor) GetMoniker() string                  { return g.Description.Moniker }
func (g Governor) GetStatus() GovernorStatus           { return g.Status }
func (g Governor) GetDescription() GovernorDescription { return g.Description }
func (g Governor) GetAddress() types.GovernorAddress {
	addr, err := types.GovernorAddressFromBech32(g.GovernorAddress)
	if err != nil {
		panic(err)
	}
	return addr
}
