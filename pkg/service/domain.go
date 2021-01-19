package service

import (
	"context"

	v1 "github.com/majst01/metal-dns/api/v1"
	"go.uber.org/zap"
)

type DomainService struct {
	log *zap.Logger
}

func NewDomainService(l *zap.Logger) *DomainService {
	return &DomainService{
		log: l,
	}
}
func (d *DomainService) List(ctx context.Context, req *v1.DomainsListRequest) (*v1.DomainsResponse, error) {
	return nil, nil
}

func (d *DomainService) Create(ctx context.Context, req *v1.DomainCreateRequest) (*v1.DomainResponse, error) {
	return nil, nil
}
func (d *DomainService) Delete(ctx context.Context, req *v1.DomainDeleteRequest) (*v1.DomainResponse, error) {
	return nil, nil
}
func (d *DomainService) Get(ctx context.Context, req *v1.DomainGetRequest) (*v1.DomainResponse, error) {
	return nil, nil
}
