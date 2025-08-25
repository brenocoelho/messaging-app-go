package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func LoadConfig[T any](config *T) error {
	viper.SetConfigFile(".env") // Look for a .env file
	viper.SetConfigType("env")  // Treat it as an env file
	viper.AutomaticEnv()        // Read from environment variables

	// Read config file (if exists)
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Config file loaded:", viper.ConfigFileUsed())
	} else {
		fmt.Println("No config file found, using only environment variables.")
	}

	// var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}
