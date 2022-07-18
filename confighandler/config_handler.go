package confighandler

import (
	"weezel/budget/logger"

	toml "github.com/pelletier/go-toml"
)

type TeleConfig struct {
	APIKey     string
	WorkingDir string
	ChannelID  int64
}

type WebserverConfig struct {
	HTTPPort string
	Hostname string
}

type TomlConfig struct {
	WebserverConfig WebserverConfig
	TeleConfig      TeleConfig
}

func LoadConfig(filedata []byte) TomlConfig {
	config := TomlConfig{}
	if err := toml.Unmarshal(filedata, &config); err != nil {
		logger.Panicf("Error parsing config: %s", err)
	}
	return config
}
