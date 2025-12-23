package voteextension

import (
	"encoding/json"
	"errors"
	"example/x/example/keeper"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ProposalHandler struct {
	Logger   log.Logger
	Keeper   keeper.Keeper
	valStore baseapp.ValidatorStore
}

type InjectedVoteExtTx struct {
	Signatures [][]byte
}

func (h *ProposalHandler) PrepareProposal() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {

		ctx.Logger().Info("PrepareProposal called",
			"height", req.Height,
			"num_votes", len(req.LocalLastCommit.Votes),
		)

		var signatures [][]byte

		for i, vote := range req.LocalLastCommit.Votes {
			ctx.Logger().Info("Examining vote",
				"index", i,
				"vote_extension_length", len(vote.VoteExtension),
				"signature_length", len(vote.ExtensionSignature),
			)

			if len(vote.VoteExtension) == 0 {
				ctx.Logger().Info("Vote has no extension", "index", i)
				continue
			}

			var voteExt SignatureVoteExtend
			if err := json.Unmarshal(vote.VoteExtension, &voteExt); err != nil {
				ctx.Logger().Error("Failed to unmarshal vote extension in prepare",
					"error", err,
					"index", i,
					"raw_data", string(vote.VoteExtension),
				)
				continue
			}

			signatures = append(signatures, voteExt.Signature)
			ctx.Logger().Info("Successfully extracted signature",
				"index", i,
				"signature_length", len(voteExt.Signature),
			)
		}

		ctx.Logger().Info("PrepareProposalHandler collected signatures",
			"count", len(signatures),
			"total_votes", len(req.LocalLastCommit.Votes),
			"height", req.Height,
		)

		if len(signatures) == 0 {
			ctx.Logger().Info("No vote extensions found, not injecting tx")
			return &abci.ResponsePrepareProposal{
				Txs: req.Txs,
			}, nil
		}

		injectedTx := InjectedVoteExtTx{
			Signatures: signatures,
		}
		tx, err := json.Marshal(injectedTx)
		if err != nil {
			return nil, errors.New("Error while marshalling json")
		}

		txs := make([][]byte, 0, len(req.Txs)+1)
		txs = append(txs, tx)
		txs = append(txs, req.Txs...)

		ctx.Logger().Info("Injected signature tx",
			"injected_tx_size", len(tx),
			"total_txs", len(txs),
		)

		return &abci.ResponsePrepareProposal{
			Txs: txs,
		}, nil
	}
}

// ProcessProposal validates the proposal including the injected signature transaction
func (h *ProposalHandler) ProcessProposal() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		ctx.Logger().Info("ProcessProposal started",
			"height", req.Height,
			"num_txs", len(req.Txs),
			"proposer", req.ProposerAddress,
		)
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

		// Verify that we have signatures in the injected tx
		if len(injectedTx.Signatures) == 0 {
			ctx.Logger().Error("Signature tx has no signatures")
			return &abci.ResponseProcessProposal{
				Status: abci.ResponseProcessProposal_REJECT,
			}, nil
		}

		ctx.Logger().Info("Signature transaction validated",
			"height", req.Height,
			"num_signatures", len(injectedTx.Signatures),
			"total_txs", len(req.Txs),
		)

		return &abci.ResponseProcessProposal{
			Status: abci.ResponseProcessProposal_ACCEPT,
		}, nil
	}
}
