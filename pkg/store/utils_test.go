package store

import (
	"testing"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

func TestBuildSlotBoundsFilterClause(t *testing.T) {
	type args struct {
		query      string
		slotBounds *types.SlotBounds
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "nil",
			args: args{
				query:      "SELECT * FROM analysis",
				slotBounds: nil,
			},
			want: "SELECT * FROM analysis",
		},
		{
			name: "start",
			args: args{
				query: "SELECT * FROM analysis",
				slotBounds: &types.SlotBounds{
					StartSlot: types.SlotPtr(123),
				},
			},
			want: "SELECT * FROM analysis AND slot >= 123",
		},
		{
			name: "end",
			args: args{
				query: "SELECT * FROM analysis",
				slotBounds: &types.SlotBounds{
					EndSlot: types.SlotPtr(123),
				},
			},
			want: "SELECT * FROM analysis AND slot <= 123",
		},
		{
			name: "both",
			args: args{
				query: "SELECT * FROM analysis",
				slotBounds: &types.SlotBounds{
					StartSlot: types.SlotPtr(123),
					EndSlot:   types.SlotPtr(456),
				},
			},
			want: "SELECT * FROM analysis AND slot >= 123 AND slot <= 456",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildSlotBoundsFilterClause(tt.args.query, tt.args.slotBounds); got != tt.want {
				t.Errorf("BuildSlotBoundsFilterClause() = %v, want %v", got, tt.want)
			}
		})
	}
}
