package service

import (
	"context"
	"net/http"

	"github.com/joeig/go-powerdns/v2"
	v1 "github.com/majst01/metal-dns/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DomainService struct {
	pdns *powerdns.Client
	log  *zap.Logger
}

func NewDomainService(l *zap.Logger, baseURL string, vHost string, apikey string, httpClient *http.Client) *DomainService {
	pdns := powerdns.NewClient(baseURL, vHost, map[string]string{"X-API-Key": apikey}, httpClient)
	return &DomainService{
		pdns: pdns,
		log:  l,
	}
}
func (d *DomainService) List(ctx context.Context, req *v1.DomainsListRequest) (*v1.DomainsResponse, error) {
	zones, err := d.pdns.Zones.List()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	domains := []*v1.Domain{}
	for _, zone := range zones {
		domain := &v1.Domain{
			Name: *zone.Name,
		}
		domains = append(domains, domain)
	}
	return &v1.DomainsResponse{Domains: domains}, nil
}

func (d *DomainService) Get(ctx context.Context, req *v1.DomainGetRequest) (*v1.DomainResponse, error) {
	zone, err := d.pdns.Zones.Get(req.Name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	domain := &v1.Domain{
		Name: *zone.Name,
	}
	return &v1.DomainResponse{Domain: domain}, nil
}

func (d *DomainService) Create(ctx context.Context, req *v1.DomainCreateRequest) (*v1.DomainResponse, error) {
	// TODO add parameters to DomainCreateRequest
	zone := &powerdns.Zone{
		Name:        &req.Name,
		Kind:        powerdns.ZoneKindPtr(powerdns.MasterZoneKind),
		DNSsec:      powerdns.Bool(false),
		Nsec3Param:  nil,
		Nsec3Narrow: powerdns.Bool(false),
		SOAEdit:     nil,
		SOAEditAPI:  nil,
		APIRectify:  powerdns.Bool(false),
		Nameservers: req.Nameservers,
	}
	zone, err := d.pdns.Zones.Add(zone)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	domain := &v1.Domain{
		Name: *zone.Name,
	}
	return &v1.DomainResponse{Domain: domain}, nil
}
func (d *DomainService) Delete(ctx context.Context, req *v1.DomainDeleteRequest) (*v1.DomainResponse, error) {
	err := d.pdns.Zones.Delete(req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	domain := &v1.Domain{
		Name: req.Name,
	}
	return &v1.DomainResponse{Domain: domain}, nil
}
