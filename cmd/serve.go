package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lukasbarti/musixx/internal/app"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the musixx web server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			application, err := app.New()
			if err != nil {
				return fmt.Errorf("init app: %w", err)
			}

			if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("run server: %w", err)
			}

			return nil
		},
	}

	return cmd
}
