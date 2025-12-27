package app

import (
	"encoding/hex"
	"testing"

	"bytes"
	"encoding/json"
	secondarykeys "example/x/secondarykeys/module"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	CosmosK1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const chainID string = "example"
const rpcURL string = "http://localhost:26657"
const toAddrStr string = "cosmos1yguqe63ekq7e3vnkmpc3gr0n9h43zf53cemrrt"
const sendAmount int64 = 1
const txFee int64 = 1000
const faucetRequestAmount string = "100000stake"

type FaucetReq struct {
	Address string   `json:"address"`
	Coins   []string `json:"coins"`
}

type AccountResp struct {
	Account json.RawMessage `json:"account"`
}

type BaseAccount struct {
	AccountNumber string `json:"account_number"`
	Sequence      string `json:"sequence"`
}

type VestingAccount struct {
	BaseAccount BaseAccount `json:"base_account"`
}

func TestSendTxViaHTTP(t *testing.T) {
	// Generate new PrivateKey
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()

	// Derive the address for the new Private Key
	fromAddr := sdk.AccAddress(pub.Address())

	// Fund the new account
	err := fundFromFaucet(fromAddr.String())
	if err != nil {
		t.Fatal(err.Error())
	}

	memo, err := CreateMemo()
	if err != nil {
		t.Fatal(err.Error())
	}
	//Create the transaction with the given info
	signDoc, err := CreateTX(fromAddr, memo, &CosmosK1.PubKey{
		Key: pub.Bytes(),
	})

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

// Requests funds from faucet
func fundFromFaucet(addr string) error {
	req := FaucetReq{
		Address: addr,
		Coins:   []string{faucetRequestAmount},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		"http://0.0.0.0:4500/",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("faucet failed")
	}
	return nil
}

func getAccountNumbers(addr string) (uint64, uint64, error) {
	url := fmt.Sprintf(
		"http://0.0.0.0:1317/cosmos/auth/v1beta1/accounts/%s",
		addr,
	)

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var accResp AccountResp
	if err := json.NewDecoder(resp.Body).Decode(&accResp); err != nil {
		return 0, 0, err
	}

	base, err := extractBaseAccount(accResp.Account)
	if err != nil {
		return 0, 0, err
	}

	accNum, _ := strconv.ParseUint(base.AccountNumber, 10, 64)
	seq, _ := strconv.ParseUint(base.Sequence, 10, 64)

	return accNum, seq, nil
}

func extractBaseAccount(raw json.RawMessage) (*BaseAccount, error) {
	// Try direct BaseAccount
	var base BaseAccount
	if err := json.Unmarshal(raw, &base); err == nil && base.AccountNumber != "" {
		return &base, nil
	}

	// Try nested in base_account
	var nested struct {
		BaseAccount BaseAccount `json:"base_account"`
	}
	if err := json.Unmarshal(raw, &nested); err == nil && nested.BaseAccount.AccountNumber != "" {
		return &nested.BaseAccount, nil
	}

	// Try vesting account
	var vesting struct {
		BaseVestingAccount VestingAccount `json:"base_vesting_account"`
	}
	if err := json.Unmarshal(raw, &vesting); err == nil && vesting.BaseVestingAccount.BaseAccount.AccountNumber != "" {
		return &vesting.BaseVestingAccount.BaseAccount, nil
	}

	return nil, fmt.Errorf("failed to extract base account")
}

func CreateTX(fromAddr sdk.AccAddress, memo string, sdkPub *CosmosK1.PubKey) (*txtypes.SignDoc, error) {

	toAddr, err := sdk.AccAddressFromBech32(toAddrStr)
	if err != nil {
		return nil, err
	}

	amount := sdk.NewCoins(sdk.NewInt64Coin("stake", sendAmount))

	msg := &banktypes.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      amount,
	}

	msgAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(err)
	}

	// Create Transaction's body
	body := &txtypes.TxBody{
		Messages: []*codectypes.Any{msgAny},
		Memo:     memo,
	}

	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		return nil, err
	}

	pubAny, err := codectypes.NewAnyWithValue(sdkPub)
	if err != nil {
		return nil, err
	}

	fee := sdk.NewCoins(sdk.NewInt64Coin("stake", txFee))

	// Get the account number and sequence to create tx
	accNum, seq, err := getAccountNumbers(fromAddr.String())
	if err != nil {
		return nil, err
	}

	signerInfo := &txtypes.SignerInfo{
		PublicKey: pubAny,
		ModeInfo: &txtypes.ModeInfo{
			Sum: &txtypes.ModeInfo_Single_{
				Single: &txtypes.ModeInfo_Single{
					Mode: signing.SignMode_SIGN_MODE_DIRECT,
				},
			},
		},
		Sequence: seq,
	}

	authInfo := &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{signerInfo},
		Fee: &txtypes.Fee{
			Amount:   fee,
			GasLimit: 200000,
		},
	}

	authBytes, _ := proto.Marshal(authInfo)

	return &txtypes.SignDoc{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authBytes,
		ChainId:       chainID,
		AccountNumber: accNum,
	}, nil
}

func BroadcastTx(txHex string) error {
	url := fmt.Sprintf("%s/broadcast_tx_commit?tx=0x%s", rpcURL, txHex)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func CreateMemo() (string, error) {
	// Generate a random Ethereum private key
	secondaryPrivKey, err := EthereumK1.GenerateKey()
	if err != nil {
		return "", err
	}

	// Get the public key (uncompressed format, 65 bytes)
	secondaryPubKey := crypto.FromECDSAPub(&secondaryPrivKey.PublicKey)

	// Sign the predefined messagea
	messageToSign := []byte(secondaryPubKey)
	hsh := crypto.Keccak256(messageToSign)

	signature, err := EthereumK1.Sign(hsh, secondaryPrivKey)
	if err != nil {
		panic(err)
	}
	// Remove the recovery byte of the signature
	sigNoV := signature[:64]

	// Create the secondary signature struct
	secondSig := SecondarySignature{
		PublicKey: secondaryPubKey,
		Signature: sigNoV,
	}

	// Encode it into memo format
	memoBytes, err := EncodeMemoWithSecondSig(secondSig)
	if err != nil {
		return "", err
	}
	log.Println("memo string", string(memoBytes))

	// Verify the signature
	if !EthereumK1.VerifySignature(secondSig.PublicKey, hsh, secondSig.Signature) {
		return "", err
	}

	memo := secondarykeys.AnteHandlerPrefix + string(memoBytes)

	return memo, nil
}
