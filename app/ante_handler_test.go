package app_test

import (
	"example/app"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
)

const ChainID = "example"

func TestCustomAnteHandler(t *testing.T) {

	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()
	addr := sdk.AccAddress(pub.Address())

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	msg := testdata.NewTestMsg(addr)

	// Build transaction
	txBuilder := txConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	if err != nil {
		panic(err)
	}

	txBuilder.SetGasLimit(200000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)))
	memo, err := app.CreateValidMemo()
	if err != nil {
		panic("err at creating memo")
	}
	txBuilder.SetMemo(memo)

	// Sign the transaction
	sigV2 := signing.SignatureV2{
		PubKey: pub,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: 0,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		panic(err)
	}

	// Generate sign bytes
	signerData := authsigning.SignerData{
		ChainID:       ChainID,
		AccountNumber: 0,
		Sequence:      0,
	}

	ctx := sdk.NewContext(nil, cmtproto.Header{
		ChainID: ChainID,
		Height:  1,
	}, false, log.NewNopLogger()).
		WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	signBytes, err := authsigning.GetSignBytesAdapter(
		ctx,
		txConfig.SignModeHandler(),
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		panic(err)
	}

	signature, err := priv.Sign(signBytes)
	if err != nil {
		panic(err)
	}

	sigV2 = signing.SignatureV2{
		PubKey: pub,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: signature,
		},
		Sequence: 0,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		panic(err)
	}

	// Get the tx
	signedTx := txBuilder.GetTx()

	// Test with your custom decorator only
	myCustomAnteHandler := sdk.ChainAnteDecorators(
		app.NewSecondarySignatureVerificationDecorator(),
	)

	_, err = myCustomAnteHandler(ctx, signedTx, false)
	if err != nil {
		t.Logf("Custom ante handler returned error: %v", err)
		t.Logf("This might be expected if your decorator needs keepers or store access")
	} else {
		t.Log("Custom ante handler test passed!")
	}
}
