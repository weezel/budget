package utils

import "testing"

func TestGetCategory(t *testing.T) {
	type args struct {
		tokens []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Shopping without category",
			args{[]string{"osto", "lidl", "6.66"}},
			"",
		},
		{
			"Shopping with category",
			args{[]string{"osto", "#oma", "lidl", "6.66"}},
			"oma",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCategory(tt.args.tokens)
			if got != tt.want {
				t.Errorf("%s: GetCategory() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}
