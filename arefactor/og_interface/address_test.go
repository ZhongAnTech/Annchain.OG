package og_interface

import (
	"fmt"
	"testing"
)

func TestAddress20_BytesToAddress(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		a    Address20
		args args
	}{
		{name: "c1", a: Address20([20]byte{}), args: args{b: []byte{0x03, 0x44}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.a.FromBytes(tt.args.b)
			tt.a[3] = 0x88
			fmt.Printf("%+v", tt.a)
		})
	}
}

func TestAddress20_HexToAddress(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		a       Address20
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.FromHex(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("HexToAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddress20_HexToAddressNoError(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		a    Address20
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
