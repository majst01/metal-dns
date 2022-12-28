package service

import (
	"context"
	"fmt"
	"net/http"

	connect "github.com/bufbuild/connect-go"
	"github.com/joeig/go-powerdns/v3"
	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/token"
	"go.uber.org/zap"
)

type DomainService struct {
	pdns  *powerdns.Client
	log   *zap.SugaredLogger
	vhost string
}

func NewDomainService(l *zap.SugaredLogger, baseURL string, vHost string, apikey string, httpClient *http.Client) *DomainService {
	pdns := powerdns.NewClient(baseURL, vHost, map[string]string{"X-API-Key": apikey}, httpClient)
	return &DomainService{
		pdns:  pdns,
		log:   l.Named("domain"),
		vhost: vHost,
	}
}

func (d *DomainService) List(ctx context.Context, rq *connect.Request[v1.DomainServiceListRequest]) (*connect.Response[v1.DomainServiceListResponse], error) {
	d.log.Debugw("list", "req", rq)
	req := rq.Msg
	claims := token.ClaimsFromContext(ctx)
	allowedDomains := claims.Domains

	filtered, err := filterDomains(req.Domains, allowedDomains)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	zones, err := d.pdns.Zones.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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
	return connect.NewResponse(&v1.DomainServiceListResponse{Domains: domains}), nil
}

func (d *DomainService) Get(ctx context.Context, rq *connect.Request[v1.DomainServiceGetRequest]) (*connect.Response[v1.DomainServiceGetResponse], error) {
	d.log.Debugw("get", "req", rq)
	req := rq.Msg
	zone, err := d.pdns.Zones.Get(ctx, req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	domain := toV1Domain(zone)
	return connect.NewResponse(&v1.DomainServiceGetResponse{Domain: domain}), nil
}

func (d *DomainService) Create(ctx context.Context, rq *connect.Request[v1.DomainServiceCreateRequest]) (*connect.Response[v1.DomainServiceCreateResponse], error) {
	d.log.Debugw("create", "req", rq)
	req := rq.Msg
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
		zone.URL = req.Url
	}
	zone, err := d.pdns.Zones.Add(ctx, zone)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	domain := toV1Domain(zone)
	return connect.NewResponse(&v1.DomainServiceCreateResponse{Domain: domain}), nil
}

func (d *DomainService) Update(ctx context.Context, rq *connect.Request[v1.DomainServiceUpdateRequest]) (*connect.Response[v1.DomainServiceUpdateResponse], error) {
	d.log.Debugw("update", "req", rq)
	req := rq.Msg
	existingZone, err := d.pdns.Zones.Get(ctx, req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	existingZone.Nameservers = req.Nameservers
	if req.Url != nil {
		existingZone.URL = req.Url
	}

	err = d.pdns.Zones.Change(ctx, *existingZone.Name, existingZone)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	domain := toV1Domain(existingZone)
	return connect.NewResponse(&v1.DomainServiceUpdateResponse{Domain: domain}), nil
}

func (d *DomainService) Delete(ctx context.Context, rq *connect.Request[v1.DomainServiceDeleteRequest]) (*connect.Response[v1.DomainServiceDeleteResponse], error) {
	d.log.Debugw("delete", "req", rq)
	req := rq.Msg
	err := d.pdns.Zones.Delete(ctx, req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	domain := &v1.Domain{
		Name: req.Name,
	}
	return connect.NewResponse(&v1.DomainServiceDeleteResponse{Domain: domain}), nil
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
