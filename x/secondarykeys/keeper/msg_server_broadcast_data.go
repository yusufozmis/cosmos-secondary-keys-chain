package keeper

import (
	"context"
	"strings"

	"example/common"
	"example/x/secondarykeys/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
)

func (k msgServer) BroadcastData(ctx context.Context, msg *types.MsgBroadcastData) (*types.MsgBroadcastDataResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Sender); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}
	msg.Data = strings.Trim(msg.Data, "SECONDARY")
	secondSig, err := common.DecodeSecondSigFromMemo([]byte(msg.Data))
	if err != nil {
		panic(err)
	}
	err = secondSig.Validate()
	if err != nil {
		panic(err)
	}

	hsh := crypto.Keccak256([]byte(secondSig.PublicKey))
	if EthereumK1.VerifySignature(secondSig.PublicKey, hsh, secondSig.Signature) {
		f, err := sdk.AccAddressFromBech32(msg.Sender)
		if err != nil {
			panic(err)
		}
		err = k.SetSecondaryPubKeyAnteHandler(ctx, f, secondSig.PublicKey)
		if err != nil {
			panic(err)
		}
	}
	return &types.MsgBroadcastDataResponse{}, nil
}
