package keeper

import (
	"context"
	"example/x/secondarykeys/types"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema         collections.Schema
	Params         collections.Item[types.Params]
	AnteHandlerMap collections.Map[sdk.AccAddress, []byte]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		AnteHandlerMap: collections.NewMap(
			sb,
			collections.NewPrefix(0), // or 1, 2, etc if you have multiple maps
			"ante_handler_map",
			sdk.AccAddressKey,
			collections.BytesValue,
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}

func (k Keeper) SetSecondaryPubKeyAnteHandler(ctx context.Context, addr sdk.AccAddress, pubKey []byte) error {
	return k.AnteHandlerMap.Set(ctx, addr, pubKey)
}

func (k Keeper) GetSecondaryPubKeyAnteHandler(ctx context.Context, addr sdk.AccAddress) ([]byte, error) {
	bz, err := k.AnteHandlerMap.Get(ctx, addr)
	if err != nil {
		return nil, err
	}
	return bz, err
}
