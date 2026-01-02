package benchmark

import (
	"encoding/json"
	"fmt"

	"time"

	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	CosmosK1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
)

const NumberOfAccounts int = 10 // Funding each account is around 2 seconds.
// Hence, numberofaccounts should not be too big
const NumberOfTransactionsPerAccount int = 10000

type Account struct {
	PrivKey  secp256k1.PrivKey
	PubKey   secp256k1.PubKey
	Address  sdk.AccAddress
	AccNum   uint64
	Sequence uint64
}

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
