package orm

import (
	"reflect"
	"testing"

	"github.com/atomone-hub/atomone/codec"
	"github.com/atomone-hub/atomone/codec/types"
	"github.com/atomone-hub/atomone/testutil/testdata"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/group/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeSafeRowGetter(t *testing.T) {
	storeKey := sdk.NewKVStoreKey("test")
	ctx := NewMockContext()
	prefixKey := [2]byte{0x2}
	store := prefix.NewStore(ctx.KVStore(storeKey), prefixKey[:])
	md := testdata.TableModel{
		Id:   1,
		Name: "some name",
	}
	bz, err := md.Marshal()
	require.NoError(t, err)
	store.Set(EncodeSequence(1), bz)

	specs := map[string]struct {
		srcRowID     RowID
		srcModelType reflect.Type
		expObj       interface{}
		expErr       *sdkerrors.Error
	}{
		"happy path": {
			srcRowID:     EncodeSequence(1),
			srcModelType: reflect.TypeOf(testdata.TableModel{}),
			expObj:       md,
		},
		"unknown rowID should return sdkerrors.ErrNotFound": {
			srcRowID:     EncodeSequence(2),
			srcModelType: reflect.TypeOf(testdata.TableModel{}),
			expErr:       sdkerrors.ErrNotFound,
		},
		"wrong type should cause sdkerrors.ErrInvalidType": {
			srcRowID:     EncodeSequence(1),
			srcModelType: reflect.TypeOf(testdata.Cat{}),
			expErr:       sdkerrors.ErrInvalidType,
		},
		"empty rowID not allowed": {
			srcRowID:     []byte{},
			srcModelType: reflect.TypeOf(testdata.TableModel{}),
			expErr:       errors.ErrORMEmptyKey,
		},
		"nil rowID not allowed": {
			srcModelType: reflect.TypeOf(testdata.TableModel{}),
			expErr:       errors.ErrORMEmptyKey,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			interfaceRegistry := types.NewInterfaceRegistry()
			cdc := codec.NewProtoCodec(interfaceRegistry)

			getter := NewTypeSafeRowGetter(prefixKey, spec.srcModelType, cdc)
			var loadedObj testdata.TableModel

			err := getter(ctx.KVStore(storeKey), spec.srcRowID, &loadedObj)
			if spec.expErr != nil {
				require.True(t, spec.expErr.Is(err), err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, spec.expObj, loadedObj)
		})
	}
}
