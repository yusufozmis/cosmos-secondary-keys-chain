package benchmark

import (
	"encoding/hex"
	"example/common"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	CosmosK1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/gogoproto/proto"
)

func TestBenchmark(t *testing.T) {

	var keys []common.Account

	for i := 0; i < common.NumberOfAccounts; i++ {
		// Generate new PrivateKeys
		priv := secp256k1.GenPrivKey()
		pub := priv.PubKey()
		// Derive the address for the new Private Key
		fromAddr := sdk.AccAddress(pub.Address())
		// Fund the new account
		err := common.FundFromFaucet(fromAddr.String())
		if err != nil {
			t.Fatal(err.Error())
		}
		accNum, seq, err := common.GetAccountNumbers(fromAddr.String())
		if err != nil {
			panic("err at get account number")
		}
		keys = append(keys, common.Account{
			PrivKey:  priv,
			PubKey:   pub.Bytes(),
			Address:  fromAddr,
			AccNum:   accNum,
			Sequence: seq,
		})
	}
	log.Printf("tx creation started")
	txHexs := make([][]string, common.NumberOfAccounts)
	for i := 0; i < common.NumberOfAccounts; i++ {
		for j := 0; j < common.NumberOfTransactionsPerAccount; j++ {
			//Create the transactsion with the given info
			signDoc, err := common.CreateTX(keys[i].Address, "", &CosmosK1.PubKey{
				Key: keys[i].PubKey.Bytes(),
			}, keys[i].AccNum, keys[i].Sequence+uint64(j))

			signDocBytes, err := proto.Marshal(signDoc)
			if err != nil {
				t.Fatal(err.Error())
			}

			// Sign the transaction
			sig, err := keys[i].PrivKey.Sign(signDocBytes)
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
			txHexs[i] = append(txHexs[i], hex.EncodeToString(txBytes))
		}
	}
	log.Printf("broadcast started")
	var wg sync.WaitGroup
	s := time.Now()
	for i := 0; i < common.NumberOfAccounts; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			for j := 0; j < common.NumberOfTransactionsPerAccount; j++ {
				err := common.BroadcastTx(txHexs[i][j])
				if err != nil {
					continue
				}
			}
		}(i)
	}
	wg.Wait()
	seconds := time.Since(s).Seconds()
	numberOfTransactions := common.NumberOfAccounts * common.NumberOfTransactionsPerAccount

	log.Println("TPS:", float64(numberOfTransactions)/seconds)
}
