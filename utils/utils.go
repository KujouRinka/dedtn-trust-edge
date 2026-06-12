package utils

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func KeyStore2PrivateKey(keystorePath, passwd string) (*keystore.Key, error) {
	keyjson, err := os.ReadFile(keystorePath)
	if err != nil {
		return nil, err
	}

	key, err := keystore.DecryptKey(keyjson, passwd)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func SendLegacyTransaction(client *ethclient.Client, keystore, passwd, toAddr string) error {
	// node1Key, err := KeyStore2PrivateKey("./keystore/node1_priv_key", "1")
	node1Key, err := KeyStore2PrivateKey(keystore, passwd)
	if err != nil {
		return fmt.Errorf("cannot decrypt private key: %v", err)
	}
	privateKey, err := crypto.HexToECDSA(node1Key.PrivateKey.D.Text(16))
	if err != nil {
		return fmt.Errorf("cannot convert private key: %v %v", err, node1Key.PrivateKey.D.String())
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("cannot assert type")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(
		context.Background(),
		fromAddress,
	)
	if err != nil {
		return fmt.Errorf("cannot generate nonce: %v", err)
	}
	// value
	value := big.NewInt(114514) // 1 ETH
	// gas limit
	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	toAddress := common.HexToAddress(
		// "0x2976a6FA14c098Bb9F883B033E2Cd845E05c6453",
		toAddr,
	)
	tx := types.NewTransaction(
		nonce,
		toAddress,
		value,
		gasLimit,
		gasPrice,
		nil,
	)

	// chainID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("cannot get chainID: %v", err)
	}
	// sign
	signedTx, err := types.SignTx(
		tx,
		types.NewEIP155Signer(chainID),
		privateKey,
	)
	if err != nil {
		return fmt.Errorf("cannot sign transaction: %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("SendTransaction failed: %v", err)
	}

	return nil
}

func MergeChan[T any](chs ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	out := make(chan T)

	wg.Add(len(chs))
	for _, ch := range chs {
		go func(ch <-chan T) {
			defer wg.Done()
			for v := range ch {
				out <- v
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func RandomIntsButV2(n, a, b, but int) []int {
	result := make([]int, n)
	for i := 0; i < n; {
		r := a + rand.IntN(b-a)
		if r == but {
			continue
		}
		result[i] = r
		i++
	}
	return result
}

type Pair[T any, U any] struct {
	First  T
	Second U
}
