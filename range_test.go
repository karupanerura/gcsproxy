package gcsproxy

import (
	"reflect"
	"testing"
)

func Test_parseRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    bodyRange
		wantErr bool
	}{
		{
			name: "first 500 bytes",
			raw:  "bytes=0-499",
			want: bodyRange{
				offset: 0,
				length: 500,
			},
			wantErr: false,
		},
		{
			name: "second 500 bytes",
			raw:  "bytes=500-999",
			want: bodyRange{
				offset: 500,
				length: 500,
			},
			wantErr: false,
		},
		{
			name: "same byte pos",
			raw:  "bytes=500-500",
			want: bodyRange{
				offset: 500,
				length: 1,
			},
			wantErr: false,
		},
		{
			name:    "invalid last byte pos",
			raw:     "bytes=500-499",
			want:    bodyRange{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRange(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRange() got = %v, want %v", got, tt.want)
			}
		})
	}
}
