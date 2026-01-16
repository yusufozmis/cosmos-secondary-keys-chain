package voteextension

import (
	"encoding/json"
	"errors"
	"example/x/secondarykeys/keeper"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type ProposalHandler struct {
	Logger   log.Logger
	Keeper   keeper.Keeper
	valStore baseapp.ValidatorStore
}

type ValidatorSignature struct {
	ValidatorAddress []byte `json:"validator_address"`
	Signature        []byte `json:"signature"`
}

type InjectedVoteExtTx struct {
	ValidatorSignatures []ValidatorSignature `json:"validator_signatures"`
}

func (h *ProposalHandler) PrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {

		ctx.Logger().Info("PrepareProposal called")

		var validatorSignatures []ValidatorSignature

		for i, vote := range req.LocalLastCommit.Votes {
			if len(vote.VoteExtension) == 0 {
				ctx.Logger().Info("Vote has no extension", "index", i)
				continue
			}
			var voteExt SignatureVoteExtend
			if err := json.Unmarshal(vote.VoteExtension, &voteExt); err != nil {
				ctx.Logger().Error("unmarshall err")
				continue
			}
			validatorSignatures = append(validatorSignatures, ValidatorSignature{
				ValidatorAddress: vote.Validator.Address,
				Signature:        voteExt.Signature,
			})
		}
		if len(validatorSignatures) == 0 {
			ctx.Logger().Info("No vote extensions found, not injecting tx")
			return &abci.ResponsePrepareProposal{
				Txs: req.Txs,
			}, nil
		}
		injectedTx := InjectedVoteExtTx{
			ValidatorSignatures: validatorSignatures,
		}
		tx, err := json.Marshal(injectedTx)
		if err != nil {
			return nil, errors.New("Error while marshalling json")
		}

		txs := make([][]byte, 0, len(req.Txs)+1)
		txs = append(txs, tx)
		txs = append(txs, req.Txs...)
		return &abci.ResponsePrepareProposal{
			Txs: txs,
		}, nil
	}
}

// ProcessProposal validates the proposal including the injected signature transaction
func (h *ProposalHandler) ProcessProposal() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {

		if len(req.Txs) == 0 {
			ctx.Logger().Info("Empty block proposal (no vote extensions from previous block), accepting")
			return &abci.ResponseProcessProposal{
				Status: abci.ResponseProcessProposal_ACCEPT,
			}, nil
		}

		// First tx should be the signature transaction injected by PrepareProposal
		firstTx := req.Txs[0]

		var injectedTx InjectedVoteExtTx
		if err := json.Unmarshal(firstTx, &injectedTx); err != nil {
			ctx.Logger().Error("Failed to unmarshal first tx as signature tx", "error", err)
			return &abci.ResponseProcessProposal{
				Status: abci.ResponseProcessProposal_REJECT,
			}, nil
		}

		if len(injectedTx.ValidatorSignatures) == 0 {
			ctx.Logger().Error("Signature tx has no signatures")
			return &abci.ResponseProcessProposal{
				Status: abci.ResponseProcessProposal_REJECT,
			}, nil
		}

		blockHash := ctx.HeaderHash()

		for _, valSig := range injectedTx.ValidatorSignatures {
			pk, err := crypto.SigToPub(blockHash, valSig.Signature)
			publicKey := crypto.FromECDSAPub(pk)
			if err != nil {
				ctx.Logger().Error("Failed to recover public key",
					"error", err,
					"validator", valSig.ValidatorAddress,
				)
				return &abci.ResponseProcessProposal{
					Status: abci.ResponseProcessProposal_REJECT,
				}, nil
			}
			exists, err := h.Keeper.VoteExtensionMap.Has(ctx, valSig.ValidatorAddress)
			if err != nil {
				ctx.Logger().Error("vote extension map err")
				return &abci.ResponseProcessProposal{
					Status: abci.ResponseProcessProposal_REJECT,
				}, nil
			}
			if !exists {
				if err := h.Keeper.VoteExtensionMap.Set(ctx, valSig.ValidatorAddress, publicKey); err != nil {
					ctx.Logger().Error("vote extension map err")
					return &abci.ResponseProcessProposal{
						Status: abci.ResponseProcessProposal_REJECT,
					}, nil
				}
			}
		}
		ctx.Logger().Info("vote extension valid")
		return &abci.ResponseProcessProposal{
			Status: abci.ResponseProcessProposal_ACCEPT,
		}, nil
	}
}
