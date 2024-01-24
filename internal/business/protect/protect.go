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
	log2 "log"
	"log/slog"
	"net/http"
	"net/http/httputil"
)

var (
	errRedacted = errors.New("error(s) redacted")
)

type GraphQLProtect struct {
	log    *slog.Logger
	cfg    *config.Config
	po     *persisted_operations.PersistedOperationsHandler
	schema *schema.Provider
	next   http.Handler
}

func NewGraphQLProtect(log *slog.Logger, cfg *config.Config, po *persisted_operations.PersistedOperationsHandler, schema *schema.Provider, upstreamProxy *httputil.ReverseProxy) (*GraphQLProtect, error) {
	aliases.NewMaxAliasesRule(cfg.MaxAliases)
	max_depth.NewMaxDepthRule(cfg.MaxDepth)
	tks := tokens.MaxTokens(cfg.MaxTokens)
	maxBatch, err := batch.NewMaxBatch(cfg.MaxBatch)
	if err != nil {
		log.Warn("Error initializing maximum batch protection", err)
	}

	vr := ValidationRules(schema, tks, maxBatch, cfg.ObfuscateValidationErrors)
	disableMethod := enforce_post.EnforcePostMethod(cfg.EnforcePost)

	return &GraphQLProtect{
		log:    log,
		cfg:    cfg,
		po:     po,
		schema: schema,
		next:   disableMethod(po.Execute(vr(upstreamProxy))),
	}, nil
}

func (p *GraphQLProtect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO Include validation rules here?
	p.next.ServeHTTP(w, r)
}

func ValidationRules(schema *schema.Provider, tks *tokens.MaxTokensRule, batch *batch.MaxBatchRule, obfuscateErrors bool) func(next http.Handler) http.Handler { // nolint:funlen,cyclop
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			payload, err := gql.ParseRequestPayload(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			var errs gqlerror.List

			err = batch.Validate(payload)
			if err != nil {
				errs = append(errs, gqlerror.Wrap(err))
			}

			// only process the rest if no error yet
			if err == nil {
				for _, data := range payload {
					operationSource := &ast.Source{
						Input: data.Query,
					}

					err = tks.Validate(operationSource)
					if err != nil {
						errs = append(errs, gqlerror.Wrap(err))
						continue // we could consider break-ing here. That would short-circuit on error, with the downside of not returning all potential errors
					}

					var query, err = parser.ParseQuery(operationSource)
					if err != nil {
						errs = append(errs, gqlerror.Wrap(err))
						continue
					}

					errList := validator.Validate(schema.Get(), query)
					if len(errList) > 0 {
						errs = append(errs, errList...)
						continue
					}
				}
			}

			if len(errs) > 0 {
				if obfuscateErrors {
					errs = gqlerror.List{gqlerror.Wrap(errRedacted)}
				}

				response := map[string]interface{}{
					"data":   nil,
					"errors": errs,
				}

				err = json.NewEncoder(w).Encode(response)
				if err != nil {
					log2.Println(err)
				}
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
