/*
 *
 * Copyright 2019 Jens Bieber
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package auth

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// OpaAuthorizer is a gRPC server authorizer using OPA as backend
type OpaAuthorizer struct {
	credentialHeader string
	regoDecider      *regoDecider
	log              *zap.Logger
	secret           string
}

// A AuthzOption sets options such as url, used headers etc.
type AuthzOption func(*OpaAuthorizer)

// CredentialHeader is the name used to extract client credentials.
// default: "authorization"
func CredentialHeader(headerName string) AuthzOption {
	return func(o *OpaAuthorizer) {
		o.credentialHeader = headerName
	}
}

// Logger set the logger
func Logger(log *zap.Logger) AuthzOption {
	return func(o *OpaAuthorizer) {
		o.log = log
	}
}

// JWTSecret set the jwt secret
func JWTSecret(secret string) AuthzOption {
	return func(o *OpaAuthorizer) {
		o.secret = secret
	}
}

// NewOpaAuthorizer creates an OPA authorizer
func NewOpaAuthorizer(options ...AuthzOption) (*OpaAuthorizer, error) {
	authz := &OpaAuthorizer{
		credentialHeader: "authorization",
	}
	for _, opt := range options {
		opt(authz)
	}
	decider, err := newRegoDecider(authz.log.Sugar())
	if err != nil {
		return nil, err
	}
	authz.regoDecider = decider
	return authz, nil
}

// OpaStreamInterceptor is OpaAuthorizers StreamServerInterceptor for the
// server. Only one stream interceptor can be installed.
// If you want to add extra functionality you might decorate this function.
func (authz *OpaAuthorizer) OpaStreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := authz.authorize(stream.Context(), info.FullMethod, nil); err != nil {
		return err
	}

	return handler(srv, stream)
}

// OpaUnaryInterceptor is OpaAuthorizers UnaryServerInterceptor for the
// server. Only one unary interceptor can be installed.
// If you want to add extra functionality you might decorate this function.
func (authz *OpaAuthorizer) OpaUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := authz.authorize(ctx, info.FullMethod, req); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func (authz *OpaAuthorizer) authorize(ctx context.Context, methodName string, req interface{}) error {
	authz.log.Sugar().Infow("authorize")
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		authz.log.Sugar().Infow("authorize", "metadata", md)
		if token, exists := md[authz.credentialHeader]; exists {
			parts := strings.Split(token[0], " ")
			jwt := parts[1]
			ok, err := authz.regoDecider.Decide(ctx, newOpaRequest(methodName, req, jwt, authz.secret))
			if err != nil {
				authz.log.Sugar().Errorw("rego", "decision error", err)
			}
			if ok {
				return nil
			}
			authz.log.Sugar().Errorw("rego decision was false")
			return nil
		}
		return errors.New("unauthorized")
	}
	return errors.New("empty metadata")
}
