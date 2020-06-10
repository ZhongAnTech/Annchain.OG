package hexutil

import (
	"reflect"
	"testing"
)

func TestFromHex(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{name: "l1", args: args{"0x00"}, want: []byte{0}, wantErr: false},
		{name: "l1", args: args{"0xC1"}, want: []byte{0xc1}, wantErr: false},
		{name: "l1", args: args{"0x0"}, want: []byte{0}, wantErr: false},
		{name: "l1", args: args{"0x"}, want: []byte{}, wantErr: false},
		{name: "l1", args: args{"0x0003"}, want: []byte{0, 0x03}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromHex(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromHex() got = %v, want %v", got, tt.want)
			}
		})
	}
}
