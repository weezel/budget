package confighandler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigHandler(t *testing.T) {
	data := []byte(`
[teleconfig]
ChannelID = -987654
APIKey = "abcdefg:1234"

[webserverconfig]
HTTPPort = ":8080"
Hostname = "localhost"
`)
	tomlConfig := LoadConfig(data)
	assert.Equal(t, int64(-987654), tomlConfig.TeleConfig.ChannelID)
	assert.Equal(t, "abcdefg:1234", tomlConfig.TeleConfig.APIKey)
	assert.Equal(t, ":8080", tomlConfig.WebserverConfig.HTTPPort)
	assert.Equal(t, "localhost", tomlConfig.WebserverConfig.Hostname)
}
