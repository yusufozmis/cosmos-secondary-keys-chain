package app

import (
	"encoding/hex"
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	CosmosK1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/gogoproto/proto"
)

func TestSendTx(t *testing.T) {
	// Generate new PrivateKey
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()
	KeyMap[MapKeyWord] = priv
	// Derive the address for the new Private Key
	fromAddr := sdk.AccAddress(pub.Address())

	// Fund the new account
	err := FundFromFaucet(fromAddr.String())
	if err != nil {
		t.Fatal(err.Error())
	}

	memo, err := CreateValidMemo()
	if err != nil {
		t.Fatal(err.Error())
	}
	accNum, seq, err := GetAccountNumbers(fromAddr.String())
	if err != nil {
		panic("err at get account number")
	}
	//Create the transaction with the given info
	signDoc, err := CreateTX(fromAddr, memo, &CosmosK1.PubKey{
		Key: pub.Bytes(),
	}, accNum, seq)

	signDocBytes, err := proto.Marshal(signDoc)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Sign the transaction
	sig, err := priv.Sign(signDocBytes)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Raw Tx Data
	txRaw := &txtypes.TxRaw{
		BodyBytes:     signDoc.BodyBytes,
		AuthInfoBytes: signDoc.AuthInfoBytes,
		Signatures:    [][]byte{sig},
	}
	txBytes, err := proto.Marshal(txRaw)
	if err != nil {
		t.Log(err.Error())
	}
	// Broadcast the transaction
	err = BroadcastTx(hex.EncodeToString(txBytes))
	if err != nil {
		t.Fatal(err.Error())
	}
}

// Send Tx with same priv key to check antehandler mapping
func TestAnteHandlerMapping(t *testing.T) {
	// Generate new PrivateKey
	priv := KeyMap[MapKeyWord]
	pub := priv.PubKey()
	// Derive the address for the new Private Key
	fromAddr := sdk.AccAddress(pub.Address())

	// Fund the new account
	err := FundFromFaucet(fromAddr.String())
	if err != nil {
		t.Fatal(err.Error())
	}

	memo, err := CreateValidMemo()
	if err != nil {
		t.Fatal(err.Error())
	}
	accNum, seq, err := GetAccountNumbers(fromAddr.String())
	if err != nil {
		panic("err at get account number")
	}
	//Create the transaction with the given info
	signDoc, err := CreateTX(fromAddr, memo, &CosmosK1.PubKey{
		Key: pub.Bytes(),
	}, accNum, seq)

	signDocBytes, err := proto.Marshal(signDoc)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Sign the transaction
	sig, err := priv.Sign(signDocBytes)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Raw Tx Data
	txRaw := &txtypes.TxRaw{
		BodyBytes:     signDoc.BodyBytes,
		AuthInfoBytes: signDoc.AuthInfoBytes,
		Signatures:    [][]byte{sig},
	}
	txBytes, err := proto.Marshal(txRaw)
	if err != nil {
		t.Log(err.Error())
	}
	// Broadcast the transaction
	err = BroadcastTx(hex.EncodeToString(txBytes))
	if err != nil {
		t.Fatal(err.Error())
	}
}
