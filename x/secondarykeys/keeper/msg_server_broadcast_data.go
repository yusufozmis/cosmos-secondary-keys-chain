package keeper

import (
	"context"

	"example/common"
	"example/x/secondarykeys/types"

	errorsmod "cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
)

func (k msgServer) BroadcastData(ctx context.Context, msg *types.MsgBroadcastData) (*types.MsgBroadcastDataResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Sender); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}
	secondSig, err := common.DecodeSecondSigFromMemo([]byte(msg.Data))
	if err != nil {
		panic(err)
	}
	err = secondSig.Validate()
	if err != nil {
		panic(err)
	}
	hsh := crypto.Keccak256([]byte(msg.Sender))
	if EthereumK1.VerifySignature(secondSig.PublicKey, hsh, secondSig.Signature) {
		err = k.SetSecondaryPubKeyAnteHandler(ctx, []byte(msg.Sender), secondSig.PublicKey)
		if err != nil {
			panic(err)
		}
	}
	return &types.MsgBroadcastDataResponse{}, nil
}
