package app

import (
	"encoding/json"
	"fmt"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	"time"

	"bytes"
	secondarykeys "example/x/secondarykeys/module"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	CosmosK1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
)

type SecondarySignature struct {
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
}

const (
	ErrInvalidSecondaryPublicKey = "INVALID_SECONDARY_PUBLIC_KEY"
)

const chainID string = "example"
const rpcURL string = "http://localhost:26657"
const toAddrStr string = "cosmos1yguqe63ekq7e3vnkmpc3gr0n9h43zf53cemrrt"
const sendAmount int64 = 1
const txFee int64 = 0
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

var KeyMap = make(map[string]secp256k1.PrivKey)

const MapKeyWord string = "test"

func GetAddr(tx sdk.Tx) ([]byte, error) {

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, sdkerrors.ErrTxDecode
	}

	signers, err := sigTx.GetSignaturesV2()
	if err != nil {
		log.Println(err)
		return nil, sdkerrors.ErrPanic
	}
	if len(signers) == 0 {
		return nil, sdkerrors.ErrNoSignatures
	}
	pubKey := signers[0].PubKey
	if pubKey == nil {
		return nil, sdkerrors.ErrInvalidPubKey.Wrap("signature has no public key")
	}
	return signers[0].PubKey.Bytes(), nil

}

// Requests funds from faucet
func FundFromFaucet(addr string) error {
	t := time.Now()
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
	log.Printf("faucet tx took %s", time.Since(t).String())
	return nil
}

func GetAccountNumbers(addr string) (uint64, uint64, error) {
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

	base, err := ExtractBaseAccount(accResp.Account)
	if err != nil {
		return 0, 0, err
	}

	accNum, _ := strconv.ParseUint(base.AccountNumber, 10, 64)
	seq, _ := strconv.ParseUint(base.Sequence, 10, 64)

	return accNum, seq, nil
}

func ExtractBaseAccount(raw json.RawMessage) (*BaseAccount, error) {
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

func CreateTX(fromAddr sdk.AccAddress, memo string, sdkPub *CosmosK1.PubKey, accNum uint64, seq uint64) (*txtypes.SignDoc, error) {

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
	url := fmt.Sprintf("%s/broadcast_tx_async?tx=0x%s", rpcURL, txHex)

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

func CreateValidMemo() (string, error) {
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
func (s *SecondarySignature) Validate() error {
	if len(s.PublicKey) == 0 {
		return fmt.Errorf("missing public key")
	}
	if len(s.Signature) == 0 {
		return fmt.Errorf("missing signature")
	}
	return nil
}

// EncodeMemoWithSecondSig - just encode the signature
func EncodeMemoWithSecondSig(secondSig SecondarySignature) ([]byte, error) {

	memoBytes, err := json.Marshal(secondSig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal memo: %w", err)
	}

	return memoBytes, nil
}

// DecodeSecondSigFromMemo - extract the signature and remove recovery byte from signature
func DecodeSecondSigFromMemo(memo []byte) (*SecondarySignature, error) {
	if memo == nil {
		return nil, fmt.Errorf("empty memo")
	}

	var memoData SecondarySignature
	if err := json.Unmarshal(memo, &memoData); err != nil {
		return nil, fmt.Errorf("invalid memo format: %w", err)
	}

	sig := memoData.Signature

	// remove the recovery byte from the signature
	if len(sig) == 65 {
		sig = sig[:64]
	}

	return &SecondarySignature{
		PublicKey: memoData.PublicKey,
		Signature: sig,
	}, nil
}
