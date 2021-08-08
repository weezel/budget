package confighandler

import (
	"weezel/budget/logger"

	toml "github.com/pelletier/go-toml"
)

type TeleConfig struct {
	ChannelId  int64
	ApiKey     string
	WorkingDir string
}

type WebserverConfig struct {
	HttpPort string
	Hostname string
}

type TomlConfig struct {
	TeleConfig      TeleConfig
	WebserverConfig WebserverConfig
}

func LoadConfig(filedata []byte) TomlConfig {
	config := TomlConfig{}
	if err := toml.Unmarshal(filedata, &config); err != nil {
		logger.Panicf("Error parsing config: %s", err)
	}
	return config
}
