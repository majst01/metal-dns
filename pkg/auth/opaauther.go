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
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/connect-go"
	"github.com/majst01/metal-dns/pkg/policies"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
	"github.com/open-policy-agent/opa/topdown"

	"go.uber.org/zap"
)

const (
	authorizationHeader = "authorization"
)

// FIXME This buffer need to be cleared after every call
var buf bytes.Buffer

// OpaAuther is a gRPC server authorizer using OPA as backend
type OpaAuther struct {
	qDecision *rego.PreparedEvalQuery
	log       *zap.SugaredLogger
	secret    string
}

// NewOpaAuther creates an OPA authorizer
func NewOpaAuther(log *zap.SugaredLogger, secret string) (*OpaAuther, error) {
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
	// will be accessible as data.secret/roles/methods in rego rules
	data := inmem.NewFromObject(map[string]any{
		"secret": secret,
	})

	moduleLoads = append(moduleLoads, rego.Query("x = data.api.v1.metalstack.io.authz.decision"))
	moduleLoads = append(moduleLoads, rego.EnablePrintStatements(true))
	moduleLoads = append(moduleLoads, rego.PrintHook(topdown.NewPrintHook(&buf)))
	moduleLoads = append(moduleLoads, rego.Store(data))

	qDecision, err := rego.New(
		moduleLoads...,
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}
	return &OpaAuther{
		log:       log,
		secret:    secret,
		qDecision: &qDecision,
	}, nil
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
		err := o.authorize(ctx, req.Spec().Procedure, req.Header().Get, req.Any())
		if err != nil {
			return nil, err
		}
		return next(ctx, req)
	})
}
func (o *OpaAuther) authorize(ctx context.Context, methodName string, jwtTokenfunc func(string) string, req any) error {
	o.log.Debugw("authorize", "method", methodName, "req", req)
	// FIXME put this into a central config map
	if methodName == "/grpc.health.v1.Health/Check" {
		return nil
	}

	jwtToken, err := ExtractJWT(jwtTokenfunc)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	ok, err := o.decide(ctx, newOpaRequest(methodName, req, jwtToken), methodName)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	if ok {
		return nil
	}
	return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not allowed to call: %s", methodName))

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

	o.log.Debugw("made auth decision", "decision", decision)

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
	o.log.Debugw("made auth decision", "results", results)

	return true, nil
}

func newOpaRequest(method string, req any, token string) map[string]any {
	return map[string]any{
		"method":  method,
		"request": req,
		"token":   token,
	}
}

func ExtractJWT(jwtTokenfunc func(string) string) (string, error) {
	bearer := jwtTokenfunc(authorizationHeader)
	if bearer == "" {
		return "", fmt.Errorf("no header:%s found", authorizationHeader)
	}
	// can be bearer or token
	_, jwtToken, found := strings.Cut(bearer, " ")
	if !found {
		return "", fmt.Errorf("no bearer token found")
	}
	return jwtToken, nil
}
