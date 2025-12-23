package voteextension

import (
	"encoding/json"
	"example/x/example/keeper"
	secondarykeys "example/x/secondarykeys/module"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// VoteExtensionHandler handles vote extension creation and verification
type VoteExtensionHandler struct {
	keeper *keeper.Keeper
}

// NewVoteExtensionHandler creates a new vote extension handler
func NewVoteExtensionHandler(keeper *keeper.Keeper) *VoteExtensionHandler {
	return &VoteExtensionHandler{
		keeper: keeper,
	}
}

type SignatureVoteExtend struct {
	Signature []byte
}

func (h *VoteExtensionHandler) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		ctx.Logger().Info("EXTEND VOTE HANDLER CALLED",
			"height", req.GetHeight(),
		)

		signature, err := EthereumK1.Sign(ctx.HeaderHash(), secondarykeys.SecondaryPrivateKey.D.Bytes())
		if err != nil {
			ctx.Logger().Error("Failed to sign", "error", err)
			return nil, err
		}

		voteExt, err := json.Marshal(SignatureVoteExtend{
			Signature: signature,
		})
		if err != nil {
			ctx.Logger().Error("Failed to marshal vote extension", "error", err)
			return nil, err
		}
		ctx.Logger().Info("VOTE EXTENSION CREATED",
			"size", len(voteExt),
			"height", req.GetHeight(),
		)
		return &abci.ResponseExtendVote{VoteExtension: voteExt}, nil
	}
}
func (h *VoteExtensionHandler) VerifyVoteExtensionHandler() sdk.VerifyVoteExtensionHandler {

	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		ctx.Logger().Info("Verify Vote Extend CALLED",
			"height", req.Height,
		)
		var voteExtension SignatureVoteExtend
		err := json.Unmarshal(req.VoteExtension, &voteExtension)
		if err != nil {
			ctx.Logger().Info("Unmarshall ERR",
				"height", req.Height,
			)
			return &abci.ResponseVerifyVoteExtension{
				Status: abci.ResponseVerifyVoteExtension_REJECT,
			}, nil
		}
		publicKey := secondarykeys.SecondaryKeyMap[string(req.ValidatorAddress)]
		if !EthereumK1.VerifySignature(publicKey.Bytes(), ctx.HeaderHash(), voteExtension.Signature) {
			ctx.Logger().Info("Signature NOT verified, calling from verifyvoteextension",
				"height", req.Height,
			)
			return &abci.ResponseVerifyVoteExtension{
				Status: abci.ResponseVerifyVoteExtension_REJECT,
			}, nil
		}
		ctx.Logger().Info("Signature verified, calling from verifyvoteextension",
			"height", req.Height,
		)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}
