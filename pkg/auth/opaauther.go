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

	"github.com/bufbuild/connect-go"
	"github.com/majst01/metal-dns/pkg/policies"
	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
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

func (o *OpaAuther) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		o.log.Warnw("streamclient called", "procedure", spec.Procedure)
		return next(ctx, spec)
	})
}

// WrapStreamingHandler is a Opa StreamServerInterceptor for the
// server. Only one stream interceptor can be installed.
// If you want to add extra functionality you might decorate this function.
func (o *OpaAuther) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	})
}

// WrapUnary is a Opa UnaryServerInterceptor for the
// server. Only one unary interceptor can be installed.
// If you want to add extra functionality you might decorate this function.
func (o *OpaAuther) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	// Same as previous UnaryInterceptorFunc.
	return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// FIXME implement
		return next(ctx, req)
	})
}

func (o *OpaAuther) authorize(ctx context.Context, methodName string, req any) error {
	// FIXME put this into a central config map
	if methodName == "/grpc.health.v1.Health/Check" {
		return nil
	}
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

func (o *OpaAuther) decide(ctx context.Context, input map[string]any, method string) (bool, error) {
	o.log.Infow("rego evaluation", "input", input)
	results, err := o.qDecision.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("error evaluating rego result set %w", err)
	}
	if len(results) == 0 {
		return false, fmt.Errorf("error evaluating rego result set: results have no length")
	}

	decision, ok := results[0].Bindings["x"].(map[string]any)
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

func newOpaRequest(method string, req any, token, secret string) map[string]any {
	return map[string]any{
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
	_, jwt, found := strings.Cut(token[0], " ")
	if !found {
		return nil, "", fmt.Errorf("token format error")
	}
	return md, jwt, nil
}
