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
	"fmt"
	"strings"

	"github.com/majst01/metal-dns/pkg/policies"
	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const credentialHeader = "authorization"

// OpaAuther is a gRPC server authorizer using OPA as backend
type OpaAuther struct {
	qDecision *rego.PreparedEvalQuery
	log       *zap.SugaredLogger
	secret    string
}

// NewOpaAuther creates an OPA authorizer
func NewOpaAuther(log *zap.Logger, secret string) (*OpaAuther, error) {
	authz := &OpaAuther{
		log:    log.Sugar(),
		secret: secret,
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
		append(moduleLoads, rego.Query("x = data.api.v1.metalstack.io.authz.decision"))...,
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
	md, jwt, err := JWTFromContext(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	o.log.Infow("authorize", "metadata", md)
	ok, err := o.decide(ctx, newOpaRequest(methodName, req, jwt, o.secret), methodName)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if ok {
		return nil
	}
	return status.Error(codes.Unauthenticated, "not allowed to call: "+methodName)

}

func (o *OpaAuther) decide(ctx context.Context, input map[string]interface{}, method string) (bool, error) {
	o.log.Infow("rego evaluation", "input", input)
	results, err := o.qDecision.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("error evaluating rego result set %w", err)
	}
	if len(results) == 0 {
		return false, fmt.Errorf("error evaluating rego result set: results have no length")
	}

	decision, ok := results[0].Bindings["x"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("error evaluating rego result set: unexpected response type")
	}
	allow, ok := decision["allow"].(bool)
	if !ok {
		return false, fmt.Errorf("error evaluating rego result set: unexpected response type")
	}

	if !allow {
		reason, ok := decision["reason"].(string)
		if ok {
			return false, fmt.Errorf("access denied: %s", reason)
		}
		return false, fmt.Errorf("access denied to:%s", method)
	}

	// TODO remove, only for devel:
	// o.log.Infow("made auth decision", "results", results)

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

func JWTFromContext(ctx context.Context) (metadata.MD, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, "", fmt.Errorf("no metadata found")
	}
	token, exists := md[credentialHeader]
	if !exists {
		return nil, "", fmt.Errorf("no token found")
	}
	parts := strings.Split(token[0], " ")
	if len(parts) < 2 {
		return nil, "", fmt.Errorf("token format error")
	}
	jwt := parts[1]
	return md, jwt, nil
}
