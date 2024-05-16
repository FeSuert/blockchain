package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pelletier/go-toml"
)

type Config struct {
	Peers             []string `toml:"peers"`
	RPCPort           int64    `toml:"rpc_port"`
	SendPort          int64    `toml:"send_port"`
	Miners            []string `toml:"miners"`
	PrivateKey        string   `toml:"private_key"`
	MinedBlockSize    int      `toml:"mined_block_size"`
	LeaderProbability float64  `toml:"leader_probability"`
}

func SaveConfig(config *Config, configPath string) {

	data, err := toml.Marshal(config)
	if err != nil {
		log.Fatalf("Error marshalling config: %s", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		log.Fatalf("Error writing config to file: %s", err)
	}
}

func DecodeConfig(configPath string, config Config) (Config, error) {

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Println("Error opening config file:", err)
		return config, err
	}
	defer file.Close()

	if err := toml.NewDecoder(file).Decode(&config); err != nil {
		fmt.Println("Error reading config file:", err)
		return config, err
	}

	return config, nil
}
