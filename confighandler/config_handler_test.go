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
`)
	tomlConfig := LoadConfig(data)
	assert.Equal(t, int64(-987654), tomlConfig.TeleConfig.ChannelId)
	assert.Equal(t, "abcdefg:1234", tomlConfig.TeleConfig.ApiKey)
}
