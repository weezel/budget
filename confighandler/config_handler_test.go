package confighandler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	type args struct {
		filedata []byte
	}
	tests := []struct {
		name    string
		args    args
		want    TomlConfig
		wantErr bool
	}{
		{
			name: "",
			args: args{
				filedata: []byte(`
				[general]
				WorkingDir = "/home/blaa/dingdong"

				[telegram]
				ChannelID = -987654
				APIKey = "abcdefg:1234"

				[webserver]
				HTTPPort = ":8080"
				Hostname = "localhost"

				[postgres]
				Hostname = "localhost"
				Port = "5432"
				Database = "dingdong"
				Username = "tester"
				Password = "you wouldn'T have gues$ed"
				`),
			},
			want: TomlConfig{
				General: General{
					WorkingDir: "/home/blaa/dingdong",
				},
				Webserver: Webserver{
					HTTPPort: ":8080",
					Hostname: "localhost",
				},
				Telegram: Telegram{
					APIKey:    "abcdefg:1234",
					ChannelID: -987654,
				},
				Postgres: Postgres{
					Hostname: "localhost",
					Port:     "5432",
					Database: "dingdong",
					Username: "tester",
					Password: "you wouldn'T have gues$ed",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadConfig(tt.args.filedata)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: LoadConfig() error = %v, wantErr %v",
					tt.name, err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("%s: LoadConfig() mismatch:\n%s", tt.name, diff)
			}
		})
	}
}
