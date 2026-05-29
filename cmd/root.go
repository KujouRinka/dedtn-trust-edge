package cmd

import (
	"os"
	"sync"

	"github.com/kujourinka/dedtn-trust-edge/semantic"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(watchCmd)
}

var appDesc string = "none"

var generators = []semantic.Generator{
	//
}

var rootCmd = &cobra.Command{
	Use:   "None",
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
			generator.Run()
		}(generator)
	}

	_ = mergeChan(chans...)
	// todo: send semantic data to consensus
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
