package confighandler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigHandler(t *testing.T) {
	data := []byte(`
[teleconfig]
ChannelId = -987654
ApiKey = "abcdefg:1234"

[webserverconfig]
HttpPort = ":8080"
Hostname = "localhost"
`)
	tomlConfig := LoadConfig(data)
	assert.Equal(t, int64(-987654), tomlConfig.TeleConfig.ChannelId)
	assert.Equal(t, "abcdefg:1234", tomlConfig.TeleConfig.ApiKey)
	assert.Equal(t, ":8080", tomlConfig.WebserverConfig.HttpPort)
	assert.Equal(t, "localhost", tomlConfig.WebserverConfig.Hostname)
}
