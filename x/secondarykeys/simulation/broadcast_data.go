package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"example/x/secondarykeys/keeper"
	"example/x/secondarykeys/types"
)

func SimulateMsgBroadcastData(
	ak types.AuthKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
	txGen client.TxConfig,
	memo string,
) simtypes.Operation {
	newMsgServer := keeper.NewMsgServerImpl(k)
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		msg := &types.MsgBroadcastData{
			Sender: accs[0].Address.String(),
			Data:   memo,
		}
		_, err := newMsgServer.BroadcastData(ctx, msg)
		if err != nil {
			panic("err at broadcast data inside simulate")
		}
		return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "BroadcastData simulation not implemented"), nil, nil
	}
}
