package main

import (
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/business/persistedoperations"
	"github.com/ldebruijn/graphql-protect/internal/business/protect"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"log/slog"
	"os"
	"strings"
)

var ErrValidationErrorsFound = errors.New("errors found during validation")

func validate(log *slog.Logger, cfg *config.Config, _ chan os.Signal) error {
	loader, err := persistedoperations.NewLoaderFromConfig(cfg.PersistedOperations, log)
	if err != nil {
		err := fmt.Errorf("store must be defined to have files to validate")
		log.Error("Error running validations", "err", err)
		return err
	}

	// Load the persisted operations from the local dir into memory
	persistedOperations, err := persistedoperations.NewPersistedOperations(log, cfg.PersistedOperations, loader)
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
		formatErrors(errs)
		return ErrValidationErrorsFound
	}
	return nil
}

func formatErrors(errs gqlerror.List) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Hash", "Error"})

	for i, err := range errs {
		// try and format this nicely
		fIndex := strings.Index(err.Message, "[")
		lIndex := strings.Index(err.Message, "], ")
		if fIndex < 0 || lIndex < 0 || len(err.Message) < fIndex+1 || len(err.Message) < lIndex+3 {
			// prevent breaking when the expected log format is not met
			t.AppendRow(table.Row{i, "", err.Message})
			continue
		}
		hash := err.Message[fIndex+1 : lIndex]
		t.AppendRow(table.Row{i, hash, err.Message[lIndex+3:]})
	}

	t.AppendFooter(table.Row{"Total", len(errs)})
	t.Render()
}
