package benchmark

import (
	"encoding/hex"
	"example/app"
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

	var keys []Account

	for i := 0; i < NumberOfAccounts; i++ {
		// Generate new PrivateKeys
		priv := secp256k1.GenPrivKey()
		pub := priv.PubKey()
		// Derive the address for the new Private Key
		fromAddr := sdk.AccAddress(pub.Address())
		// Fund the new account
		err := FundFromFaucet(fromAddr.String())
		if err != nil {
			t.Fatal(err.Error())
		}
		accNum, seq, err := GetAccountNumbers(fromAddr.String())
		if err != nil {
			panic("err at get account number")
		}
		keys = append(keys, Account{
			PrivKey:  priv,
			PubKey:   pub.Bytes(),
			Address:  fromAddr,
			AccNum:   accNum,
			Sequence: seq,
		})
	}
	memo, err := app.CreateValidMemo()
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Printf("tx creation started")
	txHexs := make([][]string, NumberOfAccounts)
	for i := 0; i < NumberOfAccounts; i++ {
		for j := 0; j < NumberOfTransactionsPerAccount; j++ {
			//Create the transactsion with the given info
			signDoc, err := CreateTX(keys[i].Address, memo, &CosmosK1.PubKey{
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
	for i := 0; i < NumberOfAccounts; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			for j := 0; j < NumberOfTransactionsPerAccount; j++ {
				err := BroadcastTx(txHexs[i][j])
				if err != nil {
					continue
				}
			}
		}(i)
	}
	wg.Wait()
	seconds := time.Since(s).Seconds()
	numberOfTransactions := NumberOfAccounts * NumberOfTransactionsPerAccount

	log.Println("TPS:", float64(numberOfTransactions)/seconds)
}
