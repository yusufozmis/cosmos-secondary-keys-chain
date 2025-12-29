package benchmark

import (
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
