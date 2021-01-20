package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/gogo/status"
	"github.com/joeig/go-powerdns/v2"
	v1 "github.com/majst01/metal-dns/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type RecordService struct {
	pdns *powerdns.Client

	log *zap.Logger
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
	if req.Name != nil && req.Type == nil {
		recordSearch = byName
	}
	if req.Type != nil && req.Name == nil {
		recordSearch = byType
	}
	if req.Name != nil && req.Type != nil {
		recordSearch = byNameAndType
	}
	records := []*v1.Record{}
	for _, rset := range zone.RRsets {
		for _, r := range rset.Records {
			var record *v1.Record
			switch recordSearch {
			case byAny:
				record = toV1Record(r, rset)
			case byName:
				if req.Name.Value == *rset.Name {
					record = toV1Record(r, rset)
				}
			case byType:
				if req.Type.Value == string(*rset.Type) {
					record = toV1Record(r, rset)
				}
			case byNameAndType:
				if req.Name.Value == *rset.Name && req.Type.Value == string(*rset.Type) {
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

func toV1Record(r powerdns.Record, rset powerdns.RRset) *v1.Record {
	return &v1.Record{
		Name: *rset.Name,
		Data: *r.Content,
		Ttl:  int32(*rset.TTL),
		Type: string(*rset.Type),
	}
}

func (r *RecordService) Get(ctx context.Context, req *v1.RecordGetRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
func (r *RecordService) Create(ctx context.Context, req *v1.RecordCreateRequest) (*v1.RecordResponse, error) {
	domainparts := strings.SplitAfterN(req.Name, ".", 2)
	domain := domainparts[1]
	rrtype := powerdns.RRType(req.Type)
	r.log.Sugar().Infof("create record domain:%s name:%s type:%s", domain, req.Name, rrtype)
	err := r.pdns.Records.Add(domain, req.Name, rrtype, uint32(req.Ttl), []string{req.Data})
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
	return nil, nil
}
func (r *RecordService) Delete(ctx context.Context, req *v1.RecordDeleteRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
