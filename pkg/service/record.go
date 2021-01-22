package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogo/status"
	"github.com/joeig/go-powerdns/v2"
	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type RecordService struct {
	pdns *powerdns.Client
	log  *zap.Logger
}

func NewRecordService(l *zap.Logger, baseURL string, vHost string, apikey string, httpClient *http.Client) *RecordService {
	pdns := powerdns.NewClient(baseURL, vHost, map[string]string{"X-API-Key": apikey}, httpClient)
	return &RecordService{
		pdns: pdns,
		log:  l,
	}
}

type recordSearchType int

const (
	byAny recordSearchType = iota
	byName
	byType
	byNameAndType
)

func (r *RecordService) List(ctx context.Context, req *v1.RecordsListRequest) (*v1.RecordsResponse, error) {
	zone, err := r.pdns.Zones.Get(req.Domain)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	recordSearch := byAny
	if req.Name != nil && req.Type == v1.RecordType_ANY {
		recordSearch = byName
	}
	if req.Type != v1.RecordType_ANY && req.Name == nil {
		recordSearch = byType
	}
	if req.Name != nil && req.Type != v1.RecordType_ANY {
		recordSearch = byNameAndType
	}
	records := []*v1.Record{}
	for _, rset := range zone.RRsets {
		for _, r := range rset.Records {
			if r.Disabled != nil && *r.Disabled {
				continue
			}
			var record *v1.Record
			switch recordSearch {
			case byAny:
				record = toV1Record(r, rset)
			case byName:
				if req.Name.Value == *rset.Name {
					record = toV1Record(r, rset)
				}
			case byType:
				if req.Type.String() == string(*rset.Type) {
					record = toV1Record(r, rset)
				}
			case byNameAndType:
				if req.Name.Value == *rset.Name && req.Type.String() == string(*rset.Type) {
					record = toV1Record(r, rset)
				}
			}
			if record != nil {
				records = append(records, record)
			}
		}
	}
	return &v1.RecordsResponse{Records: records}, nil
}

func (r *RecordService) Get(ctx context.Context, req *v1.RecordGetRequest) (*v1.RecordResponse, error) {
	return nil, nil
}

func (r *RecordService) Create(ctx context.Context, req *v1.RecordCreateRequest) (*v1.RecordResponse, error) {
	domain, err := domainFromFQDN(req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	rrtype := powerdns.RRType(req.Type)
	r.log.Sugar().Infof("create record domain:%s name:%s type:%s", domain, req.Name, rrtype)
	err = r.pdns.Records.Add(domain, req.Name, rrtype, req.Ttl, []string{req.Data})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	record := &v1.Record{
		Name: req.Name,
		Data: req.Data,
		Type: req.Type,
		Ttl:  req.Ttl,
	}
	return &v1.RecordResponse{Record: record}, nil
}

func (r *RecordService) Update(ctx context.Context, req *v1.RecordUpdateRequest) (*v1.RecordResponse, error) {
	domain, err := domainFromFQDN(req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	rrtype := powerdns.RRType(req.Type)
	err = r.pdns.Records.Change(domain, req.Name, rrtype, req.Ttl, []string{req.Data})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	record := &v1.Record{
		Name: req.Name,
		Data: req.Data,
		Type: req.Type,
		Ttl:  req.Ttl,
	}
	return &v1.RecordResponse{Record: record}, nil
}

func (r *RecordService) Delete(ctx context.Context, req *v1.RecordDeleteRequest) (*v1.RecordResponse, error) {
	domain, err := domainFromFQDN(req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	rrtype := powerdns.RRType(req.Type)
	err = r.pdns.Records.Delete(domain, req.Name, rrtype)

	record := &v1.Record{
		Name: req.Name,
		Data: req.Data,
		Type: req.Type,
	}
	return &v1.RecordResponse{Record: record}, nil
}

// Helper

func domainFromFQDN(fqdn string) (string, error) {
	_, ok := dns.IsDomainName(fqdn)
	if !ok {
		return "", fmt.Errorf("%s is not a domain", fqdn)
	}
	fqdnparts := strings.SplitAfterN(fqdn, ".", 2)
	if len(fqdnparts) < 2 {
		return "", fmt.Errorf("fqdn must contain at least one dot")
	}
	domain := fqdnparts[1]
	return domain, nil
}

func toV1Record(r powerdns.Record, rset powerdns.RRset) *v1.Record {
	return &v1.Record{
		Name: *rset.Name,
		Data: *r.Content,
		Ttl:  *rset.TTL,
		Type: toV1RecordType(rset.Type),
	}
}

func toV1RecordType(t *powerdns.RRType) v1.RecordType {
	switch *t {
	case powerdns.RRTypeA:
		return v1.RecordType_A
	case powerdns.RRTypeAAAA:
		return v1.RecordType_AAAA
	case powerdns.RRTypeCAA:
		return v1.RecordType_CAA
	case powerdns.RRTypeCNAME:
		return v1.RecordType_CNAME
	case powerdns.RRTypeDNAME:
		return v1.RecordType_DNANE
	case powerdns.RRTypeDS:
		return v1.RecordType_DS
	case powerdns.RRTypeHINFO:
		return v1.RecordType_HINFO
	case powerdns.RRTypeMX:
		return v1.RecordType_MX
	case powerdns.RRTypeNS:
		return v1.RecordType_NS
	case powerdns.RRTypeRP:
		return v1.RecordType_RP
	case powerdns.RRTypeSOA:
		return v1.RecordType_SOA
	case powerdns.RRTypeSRV:
		return v1.RecordType_SRV
	case powerdns.RRTypeTLSA:
		return v1.RecordType_TLSA
	case powerdns.RRTypeTXT:
		return v1.RecordType_TXT
	}
	return v1.RecordType_ANY
}
