package protect

import (
	"encoding/json"
	"errors"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/accesslogging"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/aliases"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/enforce_post"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/max_depth"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/tokens"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/business/trusteddocuments"
	"github.com/ldebruijn/graphql-protect/internal/business/validation"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
	validatorrules "github.com/vektah/gqlparser/v2/validator/rules"
	"go.opentelemetry.io/otel"
	"log/slog"
	"net/http"
)

var (
	ErrRedacted = errors.New("error(s) redacted")

	tracer = otel.Tracer("github.com/ldebruijn/graphql-protect/internal/business/protect")
)

type GraphQLProtect struct {
	log            *slog.Logger
	cfg            *config.Config
	schema         *schema.Provider
	tokens         *tokens.MaxTokensRule
	maxBatch       *batch.MaxBatchRule
	accessLogging  *accesslogging.AccessLogging
	next           http.Handler
	preFilterChain func(handler http.Handler) http.Handler
	rules          *validatorrules.Rules
}

func NewGraphQLProtect(log *slog.Logger, cfg *config.Config, po *trusteddocuments.Handler, schema *schema.Provider, upstreamHandler http.Handler) (*GraphQLProtect, error) {
	rules := validatorrules.NewDefaultRules()

	aliases.NewMaxAliasesRule(cfg.MaxAliases, rules)
	max_depth.NewMaxDepthRule(log, cfg.MaxDepth, rules)
	maxBatch, err := batch.NewMaxBatch(cfg.MaxBatch)
	if err != nil {
		log.Warn("Error initializing maximum batch protection", "err", err)
	}

	accessLogging := accesslogging.NewAccessLogging(cfg.AccessLogging, log)

	enforcePostMethod := enforce_post.EnforcePostMethod(cfg.EnforcePost)

	return &GraphQLProtect{
		log:           log,
		cfg:           cfg,
		schema:        schema,
		tokens:        tokens.MaxTokens(cfg.MaxTokens),
		maxBatch:      maxBatch,
		accessLogging: accessLogging,
		preFilterChain: func(next http.Handler) http.Handler {
			return enforcePostMethod(po.SwapHashForQuery(next))
		},
		next:  upstreamHandler,
		rules: rules,
	}, nil
}

func (p *GraphQLProtect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "Handle Request")
	defer span.End()
	p.preFilterChain(http.HandlerFunc(p.handle)).ServeHTTP(w, r.WithContext(ctx))
}

func (p *GraphQLProtect) handle(w http.ResponseWriter, r *http.Request) {
	if p.cfg.Web.RequestBodyMaxBytes != 0 {
		r.Body = http.MaxBytesReader(w, r.Body, int64(p.cfg.Web.RequestBodyMaxBytes))
	}

	payloads, validationErrors := p.validateRequest(r)

	p.accessLogging.Log(payloads, r.Header)

	if len(validationErrors) > 0 {
		if p.cfg.ObfuscateValidationErrors {
			validationErrors = gqlerror.List{gqlerror.Wrap(ErrRedacted)}
		}

		response := map[string]interface{}{
			"data":   nil,
			"errors": validationErrors,
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			p.log.Error("could not encode error", "err", err)
		}
		return
	}

	p.next.ServeHTTP(w, r)
}

func (p *GraphQLProtect) validateRequest(r *http.Request) ([]gql.RequestData, gqlerror.List) {
	payload, err := gql.ParseRequestPayload(r)
	if err != nil {
		return nil, gqlerror.List{gqlerror.Wrap(err)}
	}

	var errs gqlerror.List

	err = p.maxBatch.Validate(payload)
	if err != nil {
		errs = append(errs, gqlerror.Wrap(err))
	}

	if err != nil {
		return nil, errs
	}

	for _, data := range payload {
		validationErrors := p.ValidateQuery(data)
		if len(validationErrors) > 0 {
			errs = append(errs, validationErrors...)
		}
	}

	return payload, filterRejected(errs)
}

func filterRejected(errs gqlerror.List) gqlerror.List {
	var filtered gqlerror.List
	for _, err := range errs {
		var ruleResult validation.RuleValidationResult
		if errors.As(err, &ruleResult) {
			if ruleResult.Result == ("REJECTED") {
				filtered = append(filtered, err)
			}
			continue
		}
		// if error is not a validation error, it should be returned
		filtered = append(filtered, err)
	}

	return filtered
}

func (p *GraphQLProtect) ValidateQuery(data gql.RequestData) gqlerror.List {
	operationSource := &ast.Source{
		Input: data.Query,
	}
	err := p.tokens.Validate(operationSource, data.OperationName)
	if err != nil {
		return gqlerror.List{gqlerror.Wrap(err)}
	}

	query, err := parser.ParseQuery(operationSource)
	if err != nil {
		return gqlerror.List{gqlerror.Wrap(err)}
	}

	return validator.ValidateWithRules(p.schema.Get(), query, p.rules)
}
