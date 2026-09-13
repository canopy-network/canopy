package rpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleCommitteeID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      uint64
		wantError string
	}{
		{
			name:  "empty defaults to root chain",
			input: "",
			want:  0,
		},
		{
			name:  "blank defaults to root chain",
			input: "   ",
			want:  0,
		},
		{
			name:  "single committee",
			input: "12",
			want:  12,
		},
		{
			name:      "malformed committee",
			input:     "nested-12",
			wantError: "invalid syntax",
		},
		{
			name:      "multiple committees rejected",
			input:     "1,2",
			wantError: "expected exactly one committee",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := singleCommitteeID(test.input)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}
