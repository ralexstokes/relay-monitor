package types

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func TestSlotFromString(t *testing.T) {
	type args struct {
		slot string
	}
	tests := []struct {
		name    string
		args    args
		want    Slot
		wantErr bool
	}{
		{
			name: "test slot from string",
			args: args{
				slot: "123",
			},
			want:    Slot(123),
			wantErr: false,
		},
		{
			name: "test slot from string",
			args: args{
				slot: "123.5",
			},
			want:    Slot(0),
			wantErr: true,
		},
		{
			name: "test slot from string",
			args: args{
				slot: "abc",
			},
			want:    Slot(0),
			wantErr: true,
		},
		{
			name: "test slot from string",
			args: args{
				slot: "123456789012345678901234567890",
			},
			want:    Slot(0),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SlotFromString(tt.args.slot)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlotFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SlotFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLSPubKeyFromHexString(t *testing.T) {
	type args struct {
		hex string
	}
	tests := []struct {
		name    string
		args    args
		want    phase0.BLSPubKey
		wantErr bool
	}{
		{
			name: "test bls pub key from hex string",
			args: args{
				hex: "0x845bd072b7cd566f02faeb0a4033ce9399e42839ced64e8b2adcfc859ed1e8e1a5a293336a49feac6d9a5edb779be53a",
			},
			want: phase0.BLSPubKey{
				0x84, 0x5b, 0xd0, 0x72, 0xb7, 0xcd, 0x56, 0x6f, 0x02, 0xfa, 0xeb, 0x0a, 0x40, 0x33, 0xce, 0x93,
				0x99, 0xe4, 0x28, 0x39, 0xce, 0xd6, 0x4e, 0x8b, 0x2a, 0xdc, 0xfc, 0x85, 0x9e, 0xd1, 0xe8, 0xe1,
				0xa5, 0xa2, 0x93, 0x33, 0x6a, 0x49, 0xfe, 0xac, 0x6d, 0x9a, 0x5e, 0xdb, 0x77, 0x9b, 0xe5, 0x3a,
			},
			wantErr: false,
		},
		{
			name: "test bls pub key from hex string",
			args: args{
				hex: "1234567890abcdef",
			},
			want:    phase0.BLSPubKey{},
			wantErr: true,
		},
		{
			name: "test bls pub key from hex string",
			args: args{
				hex: "0x1234567890abcdef1234567890abcdef",
			},
			want:    phase0.BLSPubKey{},
			wantErr: true,
		},
		{
			name: "test bls pub key from hex string",
			args: args{
				hex: "0x1234567890abcdef1234567890abcde",
			},
			want:    phase0.BLSPubKey{},
			wantErr: true,
		},
		{
			name: "test bls pub key from hex string",
			args: args{
				hex: "0x1234567890abcdef1234567890abcdeg",
			},
			want:    phase0.BLSPubKey{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BLSPubKeyFromHexString(tt.args.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("BLSPubKeyFromHexString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BLSPubKeyFromHexString() = %v, want %v", got, tt.want)
			}
		})
	}
}
