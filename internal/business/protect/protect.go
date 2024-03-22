package protect

import (
	"encoding/json"
	"errors"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/ldebruijn/graphql-protect/internal/business/persisted_operations"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/aliases"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/enforce_post"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/max_depth"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/tokens"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
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
	po             *persisted_operations.PersistedOperationsHandler
	schema         *schema.Provider
	tokens         *tokens.MaxTokensRule
	maxBatch       *batch.MaxBatchRule
	next           http.Handler
	preFilterChain func(handler http.Handler) http.Handler
}

func NewGraphQLProtect(log *slog.Logger, cfg *config.Config, po *persisted_operations.PersistedOperationsHandler, schema *schema.Provider, upstreamHandler http.Handler) (*GraphQLProtect, error) {
	aliases.NewMaxAliasesRule(cfg.MaxAliases)
	max_depth.NewMaxDepthRule(cfg.MaxDepth)
	maxBatch, err := batch.NewMaxBatch(cfg.MaxBatch)
	if err != nil {
		log.Warn("Error initializing maximum batch protection", err)
	}

	disableMethod := enforce_post.EnforcePostMethod(cfg.EnforcePost)

	return &GraphQLProtect{
		log:      log,
		cfg:      cfg,
		po:       po,
		schema:   schema,
		tokens:   tokens.MaxTokens(cfg.MaxTokens),
		maxBatch: maxBatch,
		preFilterChain: func(next http.Handler) http.Handler {
			return disableMethod(po.Execute(next))
		},
		next: upstreamHandler,
	}, nil
}

func (p *GraphQLProtect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "Handle Request")
	defer span.End()
	p.preFilterChain(http.HandlerFunc(p.handle)).ServeHTTP(w, r.WithContext(ctx))
}

func (p *GraphQLProtect) handle(w http.ResponseWriter, r *http.Request) {
	errs := p.validateRequest(r)

	if len(errs) > 0 {
		if p.cfg.ObfuscateValidationErrors {
			errs = gqlerror.List{gqlerror.Wrap(ErrRedacted)}
		}

		response := map[string]interface{}{
			"data":   nil,
			"errors": errs,
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

func (p *GraphQLProtect) validateRequest(r *http.Request) gqlerror.List {
	payload, err := gql.ParseRequestPayload(r)
	if err != nil {
		return gqlerror.List{gqlerror.Wrap(err)}
	}

	var errs gqlerror.List

	err = p.maxBatch.Validate(payload)
	if err != nil {
		errs = append(errs, gqlerror.Wrap(err))
	}

	if err != nil {
		return errs
	}

	for _, data := range payload {
		validationErrors := p.ValidateQuery(data.Query)
		if len(validationErrors) > 0 {
			errs = append(errs, validationErrors...)
		}
	}

	return errs
}

func (p *GraphQLProtect) ValidateQuery(operation string) gqlerror.List {
	operationSource := &ast.Source{
		Input: operation,
	}

	err := p.tokens.Validate(operationSource)
	if err != nil {
		return gqlerror.List{gqlerror.Wrap(err)}
	}

	query, err := parser.ParseQuery(operationSource)
	if err != nil {
		return gqlerror.List{gqlerror.Wrap(err)}
	}

	return validator.Validate(p.schema.Get(), query)
}
