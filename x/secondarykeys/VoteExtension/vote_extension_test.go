package voteextension

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	CosmosK1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/stretchr/testify/require"
)

// paramStore is a simple implementation of ParamStore for testing
type paramStore struct {
	params cmtproto.ConsensusParams
}

var SecondaryPrivateKey ecdsa.PrivateKey
var SecondaryKeyMap = make(map[string]cryptotypes.PubKey)

const MockValidator = "validator-1"
const ChainID = "example"
const FakeHashVal = "some-hash"

func initSecondaryKeysForTest() error {
	prv, err := EthereumK1.GenerateKey()
	if err != nil {
		return err
	}
	SecondaryPrivateKey = *prv
	pubKey := EthereumK1.FromECDSAPub(&SecondaryPrivateKey.PublicKey)

	SecondaryKeyMap[MockValidator] = &CosmosK1.PubKey{
		Key: pubKey,
	}
	return nil
}

func TestVoteExtension(t *testing.T) {

	err := initSecondaryKeysForTest()
	if err != nil {
		t.Fatal(err)
	}
	name := t.Name()
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)
	app := baseapp.NewBaseApp(name, logger, db, nil)

	// Set up vote extension handlers
	app.SetExtendVoteHandler(func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		signature, err := EthereumK1.Sign(req.GetHash(), &SecondaryPrivateKey)
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
			"height", req.GetHeight(),
		)
		return &abci.ResponseExtendVote{VoteExtension: voteExt}, nil
	})
	// Set up vote extension verifier
	app.SetVerifyVoteExtensionHandler(func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
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
		publicKey, exists := SecondaryKeyMap[string(req.ValidatorAddress)]
		if !exists {
			pk, err := secp256k1.RecoverPubkey(req.Hash, voteExtension.Signature)
			if err != nil {
				return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
			}
			publicKey = &CosmosK1.PubKey{Key: pk}
			SecondaryKeyMap[string(req.ValidatorAddress)] = publicKey
		}
		if !EthereumK1.VerifySignature(publicKey.Bytes(), req.Hash, voteExtension.Signature[:64]) {
			ctx.Logger().Info("Invalid Signature",
				"height", req.Height,
			)
			return &abci.ResponseVerifyVoteExtension{
				Status: abci.ResponseVerifyVoteExtension_REJECT,
			}, nil
		}
		ctx.Logger().Info("Vote Extension Verified",
			"height", req.Height,
		)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	})

	// Set parameters to enable VoteExtension
	app.SetParamStore(&paramStore{
		params: cmtproto.ConsensusParams{
			Abci: &cmtproto.ABCIParams{
				VoteExtensionsEnableHeight: 1,
			},
		},
	})

	// Initialize chain with vote extensions enabled at height 1
	_, err = app.InitChain(
		&abci.RequestInitChain{
			InitialHeight: 1,
			ConsensusParams: &cmtproto.ConsensusParams{
				Abci: &cmtproto.ABCIParams{
					VoteExtensionsEnableHeight: 1,
				},
			},
		},
	)
	require.NoError(t, err)

	header := cmtproto.Header{
		ChainID: ChainID,
		Height:  1,
	}

	metricGatherer := metrics.NewNoOpMetrics()
	ms := rootmulti.NewStore(db, logger, metricGatherer)
	ctx := sdk.NewContext(ms, header, false, logger)

	fakeHash := make([]byte, 32)
	copy(fakeHash, []byte(FakeHashVal))

	res, err := app.ExtendVote(ctx, &abci.RequestExtendVote{Height: 5, Hash: fakeHash})
	require.NoError(t, err)

	verifyRes, err := app.VerifyVoteExtension(&abci.RequestVerifyVoteExtension{
		Height:           5,
		Hash:             fakeHash,
		VoteExtension:    res.VoteExtension,
		ValidatorAddress: []byte(MockValidator),
	})
	require.NoError(t, err)
	require.Equal(t,
		abci.ResponseVerifyVoteExtension_ACCEPT,
		verifyRes.Status,
	)
}

func (ps *paramStore) Get(ctx context.Context) (cmtproto.ConsensusParams, error) {
	return ps.params, nil
}

func (ps *paramStore) Has(ctx context.Context) (bool, error) {
	return true, nil
}

func (ps *paramStore) Set(ctx context.Context, cp cmtproto.ConsensusParams) error {
	ps.params = cp
	return nil
}
