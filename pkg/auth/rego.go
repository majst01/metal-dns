package auth

import (
	_ "embed"

	"context"
	"fmt"

	"github.com/majst01/metal-dns/pkg/policies"
	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
)

// ideas are taken from: https://www.openpolicyagent.org/docs/latest/integration/#integrating-with-the-go-api

type regoDecider struct {
	log          *zap.SugaredLogger
	qDecision    *rego.PreparedEvalQuery
	qPermissions *rego.PreparedEvalQuery
}

func newRegoDecider(log *zap.SugaredLogger) (*regoDecider, error) {
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

	return &regoDecider{
		qDecision: &qDecision,
		log:       log,
	}, nil
}

func (r *regoDecider) Decide(ctx context.Context, input map[string]interface{}) (bool, error) {

	r.log.Infow("rego evaluation", "input", input)

	results, err := r.qDecision.Eval(ctx, rego.EvalInput(input))
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
	r.log.Infow("made auth decision", "results", results)

	if !allow {
		return false, fmt.Errorf("access denied")
	}

	return allow, nil
}

func (r *regoDecider) ListPermissions(ctx context.Context) ([]string, error) {
	results, err := r.qPermissions.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("error evaluating rego result set %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("error evaluating rego result set: results have no length")
	}

	set, ok := results[0].Bindings["x"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("error evaluating rego result set: unexpected response type")
	}

	var ps []string
	for _, p := range set {
		p := p.(string)
		ps = append(ps, p)
	}

	return ps, nil
}
