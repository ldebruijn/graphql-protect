package protect

import (
	"encoding/json"
	"errors"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/business/aliases"
	"github.com/ldebruijn/graphql-protect/internal/business/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/enforce_post"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/ldebruijn/graphql-protect/internal/business/max_depth"
	"github.com/ldebruijn/graphql-protect/internal/business/persisted_operations"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/business/tokens"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
	"log/slog"
	"net/http"
)

var (
	errRedacted = errors.New("error(s) redacted")
)

type GraphQLProtect struct {
	log      *slog.Logger
	cfg      *config.Config
	po       *persisted_operations.PersistedOperationsHandler
	schema   *schema.Provider
	tokens   *tokens.MaxTokensRule
	maxBatch *batch.MaxBatchRule
	next     http.Handler
}

func NewGraphQLProtect(log *slog.Logger, cfg *config.Config, po *persisted_operations.PersistedOperationsHandler, schema *schema.Provider, upstreamProxy http.Handler) (*GraphQLProtect, error) {
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
		// TODO Make sure middleware gets executed before validateRules
		next: disableMethod(po.Execute(upstreamProxy)),
	}, nil
}

func (p *GraphQLProtect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	errs := p.validateRequest(r)

	if len(errs) > 0 {
		if p.cfg.ObfuscateValidationErrors {
			errs = gqlerror.List{gqlerror.Wrap(errRedacted)}
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
	// Review question: Before when an error occurred we proceeded without validating, I don't think we desire that behaviour?
	if err != nil { // Why do we forward requests that are not parsable? Seems like a security risk?
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

	// only process the rest if no error yet
	if err == nil {
		for _, data := range payload {
			validationErrors := p.validateQuery(data)
			if len(validationErrors) > 0 {
				errs = append(errs, validationErrors...)
			}
		}
	}

	return errs
}

func (p *GraphQLProtect) validateQuery(data gql.RequestData) gqlerror.List {
	operationSource := &ast.Source{
		Input: data.Query,
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
