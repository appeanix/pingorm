package pingorm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseModelToPtr(t *testing.T) {
	tests := []struct {
		model  interface{}
		expGot interface{}
		expErr error
	}{
		{
			model: Author{
				Name: "Vichheka",
			},
			expGot: &Author{
				Name: "Vichheka",
			},
			expErr: nil,
		},
		{
			model: &Author{
				Name: "Vichheka",
			},
			expGot: &Author{
				Name: "Vichheka",
			},
			expErr: nil,
		},
		{
			model:  "hello",
			expGot: nil,
			expErr: errors.New("model must be a kind of struct or pointer to struct type"),
		},
	}

	for _, tc := range tests {
		req := require.New(t)

		got, err := parseModelToPtr(tc.model)

		req.Equal(tc.expErr, err)
		req.Equal(tc.expGot, got)
	}
}
