package service

import (
	"context"

	v1 "github.com/majst01/metal-dns/api/v1"
	"go.uber.org/zap"
)

type RecordService struct {
	log *zap.Logger
}

func NewRecordService(l *zap.Logger) *RecordService {
	return &RecordService{
		log: l,
	}
}

func (r *RecordService) Create(ctx context.Context, req *v1.RecordCreateRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
func (r *RecordService) Delete(ctx context.Context, req *v1.RecordDeleteRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
func (r *RecordService) Get(ctx context.Context, req *v1.RecordGetRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
func (r *RecordService) List(ctx context.Context, req *v1.RecordsListRequest) (*v1.RecordsResponse, error) {
	return nil, nil
}
func (r *RecordService) Update(ctx context.Context, req *v1.RecordUpdateRequest) (*v1.RecordResponse, error) {
	return nil, nil
}
