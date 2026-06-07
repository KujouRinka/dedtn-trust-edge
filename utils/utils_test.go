package utils

import (
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestSendLegacyTransaction(t *testing.T) {
	keystore, passwd := "./keystore/node1_priv_key", "1"

	client, err := ethclient.Dial("ws://192.168.0.203:23333")
	if err != nil {
		log.Fatalf("cannot connect to eth client: %v", err)
	}
	defer client.Close()

	if err := SendLegacyTransaction(client, keystore, passwd, "0x2976a6FA14c098Bb9F883B033E2Cd845E05c6453"); err != nil {
		t.Fatal("send transaction failed:", err)
	}
}
