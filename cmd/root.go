package cmd

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/kujourinka/dedtn-trust-edge/consensus"
	"github.com/kujourinka/dedtn-trust-edge/consensus/smart_bft"
	"github.com/kujourinka/dedtn-trust-edge/logger"
	"github.com/kujourinka/dedtn-trust-edge/semantic"
	"github.com/kujourinka/dedtn-trust-edge/utils"
)

func init() {
	rootCmd.AddCommand(watchCmd)
	initGenerator()
}

func initGenerator() {
	config := &semantic.ImageSemanticConfig{
		RpcHost:      "localhost",
		RpcPort:      23334,
		TimeInterval: 1 * time.Second,
		Source:       "./assets/video.mp4",
	}

	gen, err := semantic.NewImgSemanticGen(config, context.Background())
	if err != nil {
		logger.Logger.Fatal("cannot create generator", zap.Error(err))
		os.Exit(1)
	}
	generators = append(generators, gen)
}

var appDesc string = "none"

var generators []semantic.Generator

var rootCmd = &cobra.Command{
	Use:   "dedtn",
	Short: appDesc,
	Args:  cobra.ExactArgs(0),
	Run:   runMain,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runMain(cmd *cobra.Command, args []string) {
	chans := make([]<-chan semantic.SemanticClaim, 0, len(generators))
	for _, generator := range generators {
		chans = append(chans, generator.ReadChan())
		go func(generator semantic.Generator) {
			if err := generator.Run(); err != nil {
				logger.Logger.Error("generator returns error", zap.Error(err))
			}
		}(generator)
	}

	ch := utils.MergeChan(chans...)
	// todo: send semantic data to consensus

	servers := make([]consensus.Node, 0)

	var cfg smart_bft.Config
	SmartBFTServer, err := smart_bft.NewSmartPBFServer(&cfg)
	if err != nil {
		logger.Logger.Error("cannot create Smart BFT server", zap.Error(err))
		os.Exit(1)
	}
	servers = append(servers, SmartBFTServer)
	for _, server := range servers {
		go func(server consensus.Node) {
			if err := server.Run(); err != nil {
				logger.Logger.Error("consensus server running error", zap.Error(err))
			}
		}(server)
	}

	for msg := range ch {
		logger.Logger.Infof("%v", msg)
		// for _, server := range servers {
		// 	server.SubmitSemantic(&msg)
		// }
	}
}
