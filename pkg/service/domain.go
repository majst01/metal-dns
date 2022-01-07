package service

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/joeig/go-powerdns/v3"
	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc/status"
)

type DomainService struct {
	pdns  *powerdns.Client
	log   *zap.Logger
	mu    sync.RWMutex
	vhost string
}

func NewDomainService(l *zap.Logger, baseURL string, vHost string, apikey string, httpClient *http.Client) *DomainService {
	pdns := powerdns.NewClient(baseURL, vHost, map[string]string{"X-API-Key": apikey}, httpClient)
	return &DomainService{
		pdns:  pdns,
		log:   l,
		vhost: vHost,
	}
}
func (d *DomainService) List(ctx context.Context, req *v1.DomainsListRequest) (*v1.DomainsResponse, error) {
	_, jwt, err := auth.JWTFromContext(ctx)
	if err != nil {
		return nil, err
	}
	claims, err := parseJWTToken(jwt)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	allowedDomains := claims.Domains

	filtered, err := filterDomains(req.Domains, allowedDomains)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	zones, err := d.pdns.Zones.List(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	domains := []*v1.Domain{}
	for i := range zones {
		z := zones[i]
		_, exists := filtered[*z.Name]
		if !exists {
			continue
		}
		domain := toV1Domain(&z)
		domains = append(domains, domain)
	}
	return &v1.DomainsResponse{Domains: domains}, nil
}

func (d *DomainService) Get(ctx context.Context, req *v1.DomainGetRequest) (*v1.DomainResponse, error) {
	zone, err := d.pdns.Zones.Get(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	domain := toV1Domain(zone)
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
	if req.Url != nil {
		zone.URL = &req.Url.Value
	}
	zone, err := d.pdns.Zones.Add(ctx, zone)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	domain := toV1Domain(zone)
	return &v1.DomainResponse{Domain: domain}, nil
}

func (d *DomainService) Update(ctx context.Context, req *v1.DomainUpdateRequest) (*v1.DomainResponse, error) {
	existingZone, err := d.pdns.Zones.Get(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	existingZone.Nameservers = req.Nameservers
	if req.Url != nil {
		existingZone.URL = &req.Url.Value
	}

	err = d.pdns.Zones.Change(ctx, *existingZone.Name, existingZone)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	domain := toV1Domain(existingZone)
	return &v1.DomainResponse{Domain: domain}, nil
}

func (d *DomainService) Delete(ctx context.Context, req *v1.DomainDeleteRequest) (*v1.DomainResponse, error) {
	err := d.pdns.Zones.Delete(ctx, req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	domain := &v1.Domain{
		Name: req.Name,
	}
	return &v1.DomainResponse{Domain: domain}, nil
}

func toV1Domain(zone *powerdns.Zone) *v1.Domain {
	return &v1.Domain{
		Id:          *zone.ID,
		Name:        *zone.Name,
		Url:         *zone.URL,
		Nameservers: zone.Nameservers,
	}
}

func filterDomains(requested, allowed []string) (map[string]bool, error) {
	if len(allowed) == 0 {
		return nil, fmt.Errorf("no domains allowed")
	}
	allowedMap := toMap(allowed)

	if len(requested) == 0 {
		return allowedMap, nil
	}
	requestedMap := toMap(requested)

	for k := range requestedMap {
		_, exists := allowedMap[k]
		if !exists {
			return nil, fmt.Errorf("domain:%s is not allowed to list, only %s", k, allowed)
		}
	}

	return requestedMap, nil
}

func toMap(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for i := range in {
		out[in[i]] = true
	}
	return out
}

func (d *DomainService) Check(ctx context.Context, in *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s, err := d.pdns.Servers.Get(ctx, d.vhost)
	if err != nil {
		return &healthpb.HealthCheckResponse{
			Status: healthpb.HealthCheckResponse_NOT_SERVING,
		}, err
	}
	if s.Version != nil {
		return &healthpb.HealthCheckResponse{
			Status: healthpb.HealthCheckResponse_SERVING,
		}, nil
	}
	return nil, status.Error(codes.NotFound, "unknown service")
}

// Watch implements `service Health`.
func (d *DomainService) Watch(in *healthpb.HealthCheckRequest, stream healthgrpc.Health_WatchServer) error {
	// TODO not implemented
	return nil
}
