package benchmark

import (
	"encoding/hex"
	"encoding/json"
	"example/common"
	"fmt"
	"log"
	"net/http"
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
	startBlockHeight, startTime, _ := GetLatestBlockInfo()
	var wg sync.WaitGroup
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
	endBlockHeight, endTime, _ := GetLatestBlockInfo()

	numberOfTxs, err := CountTxsInHeightRange(startBlockHeight, endBlockHeight)
	if err != nil {
		panic(err)
	}
	seconds := endTime.Sub(startTime)
	log.Println("TPS:", numberOfTxs/int(seconds.Seconds()))
}

var rpcURL = "http://localhost:26657"

type Status struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight int64     `json:"latest_block_height,string"`
			LatestBlockTime   time.Time `json:"latest_block_time"`
		} `json:"sync_info"`
	} `json:"result"`
}

func GetLatestBlockInfo() (height int64, t time.Time, err error) {
	resp, err := http.Get(rpcURL + "/status")
	if err != nil {
		return 0, time.Time{}, err
	}
	defer resp.Body.Close()

	var s Status
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return 0, time.Time{}, err
	}

	return s.Result.SyncInfo.LatestBlockHeight,
		s.Result.SyncInfo.LatestBlockTime,
		nil
}
func CountTxsInHeightRange(startHeight, endHeight int64) (int, error) {
	totalTxs := 0

	for height := startHeight; height <= endHeight; height++ {
		// Query block at specific height
		resp, err := http.Get(fmt.Sprintf("http://localhost:26657/block?height=%d", height))
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		var result struct {
			Result struct {
				Block struct {
					Data struct {
						Txs []string `json:"txs"`
					} `json:"data"`
				} `json:"block"`
			} `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return 0, err
		}

		totalTxs += len(result.Result.Block.Data.Txs)
	}

	return totalTxs, nil
}
