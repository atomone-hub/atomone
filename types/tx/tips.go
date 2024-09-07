package tx

import (
	sdk "github.com/atomone-hub/atomone/types"
)

// TipTx defines the interface to be implemented by Txs that handle Tips.
type TipTx interface {
	sdk.FeeTx
	GetTip() *Tip
}
