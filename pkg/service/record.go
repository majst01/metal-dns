package service

import (
	"context"
	"net/http"

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
func (r *RecordService) List(ctx context.Context, req *v1.RecordsListRequest) (*v1.RecordsResponse, error) {
	zone, err := r.pdns.Zones.Get(req.Domain)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	records := []*v1.Record{}
	for _, rset := range zone.RRsets {
		record := &v1.Record{
			Name: *rset.Name,
			Ttl:  int32(*rset.TTL),
			Type: string(*rset.Type),
		}
		records = append(records, record)
	}
	return &v1.RecordsResponse{Records: records}, nil
}
func (r *RecordService) Get(ctx context.Context, req *v1.RecordGetRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
func (r *RecordService) Create(ctx context.Context, req *v1.RecordCreateRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
func (r *RecordService) Update(ctx context.Context, req *v1.RecordUpdateRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
func (r *RecordService) Delete(ctx context.Context, req *v1.RecordDeleteRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
