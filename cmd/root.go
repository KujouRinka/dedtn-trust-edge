package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kujourinka/dedtn-trust-edge/semantic"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(watchCmd)
	initGenerator()
}

func initGenerator() {
	config := &semantic.ImageSemanticConfig{
		RpcHost:      "localhost",
		RpcPort:      23334,
		TimeInterval: 1,
		Source:       "./assets/video.mp4",
	}

	gen, err := semantic.NewImgSemanticGen(config, context.Background())
	if err != nil {
		logger.Fatal("cannot create generator", zap.Error(err))
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
	chans := make([]<-chan interface{}, 0, len(generators))
	for _, generator := range generators {
		chans = append(chans, generator.ReadChan())
		go func(generator semantic.Generator) {
			if err := generator.Run(); err != nil {
				logger.Error("generator returns error", zap.Error(err))
			}
		}(generator)
	}

	ch := mergeChan(chans...)
	// todo: send semantic data to consensus
	for box := range ch {
		fmt.Println(box)
	}
}

func mergeChan[T any](chs ...<-chan T) <-chan T {
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
