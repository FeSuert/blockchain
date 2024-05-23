package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
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

func loadConfig(configPath string, config *Config) error {
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return toml.NewDecoder(file).Decode(config)
}

func saveConfig(config *Config, configPath string) {
	data, err := toml.Marshal(config)
	if err != nil {
		log.Fatalf("Error marshalling config: %s", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		log.Fatalf("Error writing config to file: %s", err)
	}
}

func extractNodeName(configPath string) string {
	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	return "node" + numbers
}

func createPrivateKey(privateKeyHex string) (crypto.PrivKey, error) {
	privateKeyBytes, err := hex.DecodeString(strings.TrimLeft(privateKeyHex, "0x"))
	if err != nil {
		return nil, err
	}

	priv := ed25519.NewKeyFromSeed(privateKeyBytes)
	return crypto.UnmarshalEd25519PrivateKey(priv)
}

func getPeerIDFromPublicKey(pubKeyHex string) (peer.ID, error) {
	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("error decoding public key: %v", err)
	}
	libp2pPubKey, err := crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling public key: %v", err)
	}
	return peer.IDFromPublicKey(libp2pPubKey)
}
