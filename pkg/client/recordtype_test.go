package client

import (
	"reflect"
	"testing"

	v1 "github.com/majst01/metal-dns/api/v1"
)

func TestToV1RecordType(t *testing.T) {
	tests := []struct {
		t    string
		want v1.RecordType
	}{
		{t: "A", want: v1.RecordType_A},
		{t: "AAAA", want: v1.RecordType_AAAA},
		{t: "A6", want: v1.RecordType_A6},
		{t: "MX", want: v1.RecordType_MX},
		{t: "B", want: v1.RecordType_UNKNOWN},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.t, func(t *testing.T) {
			if got := ToV1RecordType(tt.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToV1RecordType() = %v, want %v", got, tt.want)
			}
		})
	}
}
