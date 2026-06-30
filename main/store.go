package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	MACAddress string `json:"mac_address"`
	Model      string `json:"model"`
}

const configFile = "config.json"

func saveConfig(config Config) error {
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(config)
}

func loadConfig() (Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	var config Config
	err = json.NewDecoder(file).Decode(&config)
	return config, err
}

func writeJSON(path string, v any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func readJSON(path string, v any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
}
