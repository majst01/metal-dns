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
	"fmt"
	"strings"

	"github.com/majst01/metal-dns/pkg/policies"
	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// OpaAuther is a gRPC server authorizer using OPA as backend
type OpaAuther struct {
	credentialHeader string
	qDecision        *rego.PreparedEvalQuery
	log              *zap.SugaredLogger
	secret           string
}

// NewOpaAuther creates an OPA authorizer
func NewOpaAuther(log *zap.Logger, secret string) (*OpaAuther, error) {
	authz := &OpaAuther{
		credentialHeader: "authorization",
		log:              log.Sugar(),
		secret:           secret,
	}
	files, err := policies.RegoPolicies.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var moduleLoads []func(r *rego.Rego)
	for _, f := range files {
		data, err := policies.RegoPolicies.ReadFile(f.Name())
		if err != nil {
			return nil, err
		}
		moduleLoads = append(moduleLoads, rego.Module(f.Name(), string(data)))
	}

	qDecision, err := rego.New(
		append(moduleLoads, rego.Query("x = data.api.v1.metalstack.io.authz.allow"))...,
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}

	authz.qDecision = &qDecision
	return authz, nil
}

// OpaStreamInterceptor is OpaAuther StreamServerInterceptor for the
// server. Only one stream interceptor can be installed.
// If you want to add extra functionality you might decorate this function.
func (o *OpaAuther) OpaStreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := o.authorize(stream.Context(), info.FullMethod, nil); err != nil {
		return err
	}

	return handler(srv, stream)
}

// OpaUnaryInterceptor is OpaAuther UnaryServerInterceptor for the
// server. Only one unary interceptor can be installed.
// If you want to add extra functionality you might decorate this function.
func (o *OpaAuther) OpaUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := o.authorize(ctx, info.FullMethod, req); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func (o *OpaAuther) authorize(ctx context.Context, methodName string, req interface{}) error {
	o.log.Infow("authorize")
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		o.log.Infow("authorize", "metadata", md)
		if token, exists := md[o.credentialHeader]; exists {
			parts := strings.Split(token[0], " ")
			jwt := parts[1]
			ok, err := o.decide(ctx, newOpaRequest(methodName, req, jwt, o.secret))
			if err != nil {
				o.log.Errorw("rego", "decision error", err)
			}
			if ok {
				return nil
			}
			o.log.Errorw("rego decision was false")
			return nil
		}
		return errors.New("unauthorized")
	}
	return errors.New("empty metadata")
}

func (o *OpaAuther) decide(ctx context.Context, input map[string]interface{}) (bool, error) {
	o.log.Infow("rego evaluation", "input", input)
	results, err := o.qDecision.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("error evaluating rego result set %w", err)
	}
	if len(results) == 0 {
		return false, fmt.Errorf("error evaluating rego result set: results have no length")
	}
	allow, ok := results[0].Bindings["x"].(bool)
	if !ok {
		return false, fmt.Errorf("error evaluating rego result set: unexpected response type")
	}
	// TODO remove, only for devel:
	o.log.Infow("made auth decision", "results", results)

	if !allow {
		return false, fmt.Errorf("access denied")
	}

	return allow, nil
}

func newOpaRequest(method string, req interface{}, token, secret string) map[string]interface{} {
	return map[string]interface{}{
		"method":  method,
		"request": req,
		"token":   token,
		"secret":  secret,
	}
}
