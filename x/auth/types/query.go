package types

import codectypes "github.com/atomone-hub/atomone/codec/types"

func (m *QueryAccountResponse) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var account AccountI
	return unpacker.UnpackAny(m.Account, &account)
}

var _ codectypes.UnpackInterfacesMessage = &QueryAccountResponse{}
