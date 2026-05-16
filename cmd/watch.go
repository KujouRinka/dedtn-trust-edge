package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:  "watch",
	Args: cobra.ExactArgs(1),
	Run:  watchMain,
}

func watchMain(cmd *cobra.Command, args []string) {
	fmt.Println("In test")

	client, err := ethclient.Dial(args[0])
	if err != nil {
		log.Fatalf("Cannot connect to eth client: %v", err)
	}
	defer client.Close()

	headers := make(chan *types.Header, 1)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatalf("Subscribe error: %v", err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Println("Subscribe error:", err)

		case header := <-headers:
			now := time.Now().Unix()
			log.Printf(
				"block: %d - ts: %d - received at: %d - dif: %s\n",
				header.Number.Uint64(),
				header.Time,
				now,
				header.Difficulty.String())
		}
	}
}
