package client

import (
	v1 "github.com/majst01/metal-dns/api/v1"
)

func ToV1RecordType(t string) v1.RecordType {
	for rt := v1.RecordType_UNKNOWN; rt <= v1.RecordType_ZZZ; rt++ {
		if rt.String() == t {
			return rt
		}
	}
	return v1.RecordType_UNKNOWN
}
