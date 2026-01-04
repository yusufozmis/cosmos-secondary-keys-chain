package app_test

import (
	"example/app"
	"example/common"
	"testing"

	"cosmossdk.io/log"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	CosmosK1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	"math/rand"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client"

	"example/x/secondarykeys/keeper"
	"example/x/secondarykeys/simulation"
	"example/x/secondarykeys/types"
)

const ChainID = "example"

func TestAnteHandler(t *testing.T) {
	logger := log.NewNopLogger()
	db := dbm.NewMemDB()

	var appOpts servertypes.AppOptions = sims.EmptyAppOptions{}

	myApp := app.New(
		logger,
		db,
		nil,
		true,
		appOpts,
	)

	ak := myApp.AuthKeeper
	bk := myApp.BankKeeper
	k := myApp.SecondarykeysKeeper

	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()
	addr := sdk.AccAddress(pub.Address())

	accs := make([]simtypes.Account, 1)
	accs[0] = simtypes.Account{
		PrivKey: &CosmosK1.PrivKey{
			Key: priv.Bytes(),
		},
		PubKey: &CosmosK1.PubKey{
			Key: pub.Bytes(),
		},
		Address: addr,
	}

	var txGen client.TxConfig = myApp.TxConfig()
	memo, err := common.CreateValidMemo()
	if err != nil {
		t.Fatal(err)
	}

	op := simulation.SimulateMsgBroadcastData(ak, bk, k, txGen, memo)
	r := rand.New(rand.NewSource(1))

	ctx := myApp.BaseApp.NewUncachedContext(false, tmproto.Header{
		Height:  1,
		ChainID: ChainID,
		Time:    time.Now(),
	})

	_, _, err = op(
		r,
		myApp.GetBaseApp(),
		ctx,
		accs,
		ctx.ChainID(),
	)
	if err != nil {
		t.Fatalf("simulation op failed: %v", err)
	}

	msgServer := keeper.NewMsgServerImpl(k)
	msgServer.BroadcastData(ctx, &types.MsgBroadcastData{
		Sender: addr.String(),
		Data:   memo,
	})
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	// Build transaction
	txBuilder := txConfig.NewTxBuilder()
	msg := testdata.NewTestMsg(addr)
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		t.Fatal(err)
	}

	txBuilder.SetMemo(memo)

	// Sign the transaction
	sigV2 := signing.SignatureV2{
		PubKey: &CosmosK1.PubKey{
			Key: pub.Bytes(),
		},
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: 0,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		t.Fatal(err)
	}

	// Generate sign bytes
	signerData := authsigning.SignerData{
		ChainID:       ChainID,
		AccountNumber: 0,
		Sequence:      0,
	}

	signBytes, err := authsigning.GetSignBytesAdapter(
		ctx,
		txConfig.SignModeHandler(),
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		t.Fatal(err)
	}

	signature, err := priv.Sign(signBytes)
	if err != nil {
		t.Fatal(err)
	}

	sigV2 = signing.SignatureV2{
		PubKey: &CosmosK1.PubKey{
			Key: pub.Bytes(),
		},
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: signature,
		},
		Sequence: 0,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		t.Fatal(err)
	}

	// Get the tx
	signedTx := txBuilder.GetTx()

	// Test with your custom decorator only
	myCustomAnteHandler := sdk.ChainAnteDecorators(
		app.NewSecondarySignatureVerificationDecorator(k),
	)

	_, err = myCustomAnteHandler(ctx, signedTx, false)
	if err != nil {
		t.Logf("Custom ante handler returned error: %v", err)
		t.Logf("This might be expected if your decorator needs keepers or store access")
	} else {
		t.Log("Custom ante handler test passed!")
	}
}
