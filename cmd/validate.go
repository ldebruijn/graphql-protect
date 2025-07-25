package main

import (
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/business/protect"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/business/trusteddocuments"
	"github.com/ldebruijn/graphql-protect/internal/business/validation"
	"io"
	"log/slog"
	"os"
)

var ErrValidationErrorsFound = errors.New("errors found during validation")

func validate(log *slog.Logger, cfg *config.Config, _ chan os.Signal) error {
	loader, err := trusteddocuments.NewLoaderFromConfig(cfg.PersistedOperations, log)
	if err != nil {
		err := fmt.Errorf("store must be defined to have files to validate")
		log.Error("Error running validations", "err", err)
		return err
	}

	// Load the persisted operations from the local dir into memory
	persistedOperations, err := trusteddocuments.NewPersistedOperations(log, cfg.PersistedOperations, loader)
	if err != nil {
		log.Error("Error initializing Persisted Operations", "err", err)
		return nil
	}

	// Build up the schema
	schemaProvider, err := schema.NewSchema(cfg.Schema, log)
	if err != nil {
		log.Error("Error initializing schema", "err", err)
		return nil
	}

	// Validate if the operations in the manifests adhere to our 'rules' (e.g. max depth/aliases/..)
	protectChain, err := protect.NewGraphQLProtect(log, cfg, persistedOperations, schemaProvider, nil)
	if err != nil {
		log.Error("Error initializing GraphQL Protect", "err", err)
		return err
	}

	// Validate if the fields that are defined in the operation exist in our schema (this protects us from clients moving to pro before the data is there)
	errs := persistedOperations.Validate(protectChain.ValidateQuery)
	if len(errs) > 0 {
		log.Warn("Errors found during validation of operations")
		formatErrors(os.Stdout, errs)

		shouldReturnError := false
		for _, err := range errs {
			var ruleResult validation.RuleValidationResult
			if errors.As(err, &ruleResult) && ruleResult.Result == validation.REJECTED {
				shouldReturnError = true
				break
			}
		}
		if shouldReturnError {
			return ErrValidationErrorsFound
		}
	}
	return nil
}

func formatErrors(w io.Writer, errs []validation.Error) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"#", "Hash", "OperationName", "Rule", "Error", "Result"})

	for i, err := range errs {
		var ruleResult validation.RuleValidationResult
		if errors.As(err, &ruleResult) {
			t.AppendRow(table.Row{i, err.Hash, ruleResult.OperationName, ruleResult.Rule, ruleResult.Message, ruleResult.Result})
		} else {
			t.AppendRow(table.Row{i, err.Hash, "-", err.Err.Rule, err.Err.Message, "FAILURE"})
		}
	}

	t.AppendFooter(table.Row{"Total", len(errs)})
	t.Render()
}
