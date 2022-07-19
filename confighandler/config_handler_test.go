package confighandler

import (
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	type args struct {
		filedata []byte
	}
	tests := []struct {
		name string
		args args
		want TomlConfig
	}{
		{
			name: "",
			args: args{
				filedata: []byte(`
				[teleconfig]
				ChannelID = -987654
				APIKey = "abcdefg:1234"
				WorkingDir = "/home/blaa/dingdong"

				[webserverconfig]
				HTTPPort = ":8080"
				Hostname = "localhost"
				`),
			},
			want: TomlConfig{
				WebserverConfig: WebserverConfig{
					HTTPPort: ":8080",
					Hostname: "localhost",
				},
				TeleConfig: TeleConfig{
					APIKey:     "abcdefg:1234",
					WorkingDir: "/home/blaa/dingdong",
					ChannelID:  -987654,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LoadConfig(tt.args.filedata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
