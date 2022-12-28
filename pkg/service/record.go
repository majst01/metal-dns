package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	connect "github.com/bufbuild/connect-go"
	"github.com/joeig/go-powerdns/v3"
	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

type RecordService struct {
	pdns *powerdns.Client
	log  *zap.SugaredLogger
}

func NewRecordService(l *zap.SugaredLogger, baseURL string, vHost string, apikey string, httpClient *http.Client) *RecordService {
	pdns := powerdns.NewClient(baseURL, vHost, map[string]string{"X-API-Key": apikey}, httpClient)
	return &RecordService{
		pdns: pdns,
		log:  l.Named("record"),
	}
}

type recordSearchType int

const (
	byAny recordSearchType = iota
	byName
	byType
	byNameAndType
)

func (r *RecordService) List(ctx context.Context, rq *connect.Request[v1.RecordServiceListRequest]) (*connect.Response[v1.RecordServiceListResponse], error) {
	r.log.Debugw("list", "req", rq)
	req := rq.Msg
	zone, err := r.pdns.Zones.Get(ctx, req.Domain)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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
				if *req.Name == *rset.Name {
					record = toV1Record(r, rset)
				}
			case byType:
				if req.Type.String() == string(*rset.Type) {
					record = toV1Record(r, rset)
				}
			case byNameAndType:
				if *req.Name == *rset.Name && req.Type.String() == string(*rset.Type) {
					record = toV1Record(r, rset)
				}
			}
			if record != nil {
				records = append(records, record)
			}
		}
	}
	return connect.NewResponse(&v1.RecordServiceListResponse{Records: records}), nil
}

func (r *RecordService) Create(ctx context.Context, rq *connect.Request[v1.RecordServiceCreateRequest]) (*connect.Response[v1.RecordServiceCreateResponse], error) {
	r.log.Debugw("create", "req", rq)
	req := rq.Msg
	domain, err := domainFromFQDN(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rrtype := powerdns.RRType(req.Type.String())
	r.log.Infow("create record", "domain", domain, "name", req.Name, "type", rrtype)
	err = r.pdns.Records.Add(ctx, domain, req.Name, rrtype, req.Ttl, []string{req.Data})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	record := &v1.Record{
		Name: req.Name,
		Data: req.Data,
		Type: req.Type,
		Ttl:  req.Ttl,
	}
	return connect.NewResponse(&v1.RecordServiceCreateResponse{Record: record}), nil
}

func (r *RecordService) Update(ctx context.Context, rq *connect.Request[v1.RecordServiceUpdateRequest]) (*connect.Response[v1.RecordServiceUpdateResponse], error) {
	r.log.Debugw("update", "req", rq)
	req := rq.Msg
	domain, err := domainFromFQDN(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rrtype := powerdns.RRType(req.Type.String())
	err = r.pdns.Records.Change(ctx, domain, req.Name, rrtype, req.Ttl, []string{req.Data})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	record := &v1.Record{
		Name: req.Name,
		Data: req.Data,
		Type: req.Type,
		Ttl:  req.Ttl,
	}
	return connect.NewResponse(&v1.RecordServiceUpdateResponse{Record: record}), nil
}

func (r *RecordService) Delete(ctx context.Context, rq *connect.Request[v1.RecordServiceDeleteRequest]) (*connect.Response[v1.RecordServiceDeleteResponse], error) {
	r.log.Debugw("delete", "req", rq)
	req := rq.Msg
	domain, err := domainFromFQDN(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rrtype := powerdns.RRType(req.Type.String())
	err = r.pdns.Records.Delete(ctx, domain, req.Name, rrtype)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	record := &v1.Record{
		Name: req.Name,
		Data: req.Data,
		Type: req.Type,
	}
	return connect.NewResponse(&v1.RecordServiceDeleteResponse{Record: record}), nil
}

// Helper

func domainFromFQDN(fqdn string) (string, error) {
	_, ok := dns.IsDomainName(fqdn)
	if !ok {
		return "", fmt.Errorf("%s is not a domain", fqdn)
	}
	_, domain, found := strings.Cut(fqdn, ".")
	if !found {
		return "", fmt.Errorf("fqdn must contain at least one dot")
	}
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

var (
	pndsmap = map[powerdns.RRType]v1.RecordType{
		powerdns.RRTypeA:          v1.RecordType_A,
		powerdns.RRTypeAAAA:       v1.RecordType_AAAA,
		powerdns.RRTypeA6:         v1.RecordType_A6,
		powerdns.RRTypeAFSDB:      v1.RecordType_AFSDB,
		powerdns.RRTypeALIAS:      v1.RecordType_ALIAS,
		powerdns.RRTypeDHCID:      v1.RecordType_DHCID,
		powerdns.RRTypeDLV:        v1.RecordType_DLV,
		powerdns.RRTypeCAA:        v1.RecordType_CAA,
		powerdns.RRTypeCERT:       v1.RecordType_CERT,
		powerdns.RRTypeCDNSKEY:    v1.RecordType_CDNSKEY,
		powerdns.RRTypeCDS:        v1.RecordType_CDS,
		powerdns.RRTypeCNAME:      v1.RecordType_CNAME,
		powerdns.RRTypeDNSKEY:     v1.RecordType_DNSKEY,
		powerdns.RRTypeDNAME:      v1.RecordType_DNAME,
		powerdns.RRTypeDS:         v1.RecordType_DS,
		powerdns.RRTypeEUI48:      v1.RecordType_EUI48,
		powerdns.RRTypeEUI64:      v1.RecordType_EUI64,
		powerdns.RRTypeHINFO:      v1.RecordType_HINFO,
		powerdns.RRTypeIPSECKEY:   v1.RecordType_IPSECKEY,
		powerdns.RRTypeKEY:        v1.RecordType_KEY,
		powerdns.RRTypeKX:         v1.RecordType_KX,
		powerdns.RRTypeLOC:        v1.RecordType_LOC,
		powerdns.RRTypeLUA:        v1.RecordType_LUA,
		powerdns.RRTypeMAILA:      v1.RecordType_MAILA,
		powerdns.RRTypeMAILB:      v1.RecordType_MAILB,
		powerdns.RRTypeMINFO:      v1.RecordType_MINFO,
		powerdns.RRTypeMR:         v1.RecordType_MR,
		powerdns.RRTypeMX:         v1.RecordType_MX,
		powerdns.RRTypeNAPTR:      v1.RecordType_NAPTR,
		powerdns.RRTypeNS:         v1.RecordType_NS,
		powerdns.RRTypeNSEC:       v1.RecordType_NSEC,
		powerdns.RRTypeNSEC3:      v1.RecordType_NSEC3,
		powerdns.RRTypeNSEC3PARAM: v1.RecordType_NSEC3PARAM,
		powerdns.RRTypeOPENPGPKEY: v1.RecordType_OPENPGPKEY,
		powerdns.RRTypePTR:        v1.RecordType_PTR,
		powerdns.RRTypeRKEY:       v1.RecordType_RKEY,
		powerdns.RRTypeRP:         v1.RecordType_RP,
		powerdns.RRTypeRRSIG:      v1.RecordType_RRSIG,
		powerdns.RRTypeSIG:        v1.RecordType_SIG,
		powerdns.RRTypeSOA:        v1.RecordType_SOA,
		powerdns.RRTypeSPF:        v1.RecordType_SPF,
		powerdns.RRTypeSSHFP:      v1.RecordType_SSHFP,
		powerdns.RRTypeSRV:        v1.RecordType_SRV,
		powerdns.RRTypeTKEY:       v1.RecordType_TKEY,
		powerdns.RRTypeTSIG:       v1.RecordType_TSIG,
		powerdns.RRTypeTLSA:       v1.RecordType_TLSA,
		powerdns.RRTypeSMIMEA:     v1.RecordType_SMIMEA,
		powerdns.RRTypeTXT:        v1.RecordType_TXT,
		powerdns.RRTypeURI:        v1.RecordType_URI,
		powerdns.RRTypeWKS:        v1.RecordType_WKS,
	}
)

func toV1RecordType(t *powerdns.RRType) v1.RecordType {
	v1type, ok := pndsmap[*t]
	if ok {
		return v1type
	}
	return v1.RecordType_UNKNOWN
}
