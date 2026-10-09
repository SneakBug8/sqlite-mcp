package config

import (
	"github.com/spf13/cobra"
)

type Config struct {
	Debug bool
}

func NewConfig(cmd *cobra.Command) (*Config, error) {
	debug, _ := cmd.Flags().GetBool("debug")

	return &Config{
		Debug: debug,
	}, nil
}
