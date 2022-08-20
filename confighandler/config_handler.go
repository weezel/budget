package confighandler

import (
	toml "github.com/pelletier/go-toml"
)

type General struct {
	WorkingDir string
}

type Telegram struct {
	APIKey    string
	ChannelID int64
}

type Webserver struct {
	HTTPPort string
	Hostname string
}

type Postgres struct {
	Hostname string
	Port     string
	Database string
	Username string
	Password string
}

type TomlConfig struct {
	General   General
	Telegram  Telegram
	Webserver Webserver
	Postgres  Postgres
}

func LoadConfig(filedata []byte) (TomlConfig, error) {
	config := TomlConfig{}
	if err := toml.Unmarshal(filedata, &config); err != nil {
		return TomlConfig{}, err
	}
	return config, nil
}
