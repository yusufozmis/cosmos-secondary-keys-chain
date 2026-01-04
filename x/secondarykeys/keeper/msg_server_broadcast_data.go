package keeper

import (
	"context"

	"example/x/secondarykeys/types"

	errorsmod "cosmossdk.io/errors"
)

func (k msgServer) BroadcastData(ctx context.Context, msg *types.MsgBroadcastData) (*types.MsgBroadcastDataResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Sender); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	// TODO: Handle the message

	return &types.MsgBroadcastDataResponse{}, nil
}
