package common

import (
	"encoding/json"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type SecondarySignature struct {
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
}

const (
	ErrInvalidSecondaryPublicKey = "INVALID_SECONDARY_PUBLIC_KEY"
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
