package main

import (
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/business/persisted_operations"
	"github.com/ldebruijn/graphql-protect/internal/business/protect"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"log/slog"
	"os"
	"strings"
)

var ErrValidationErrorsFound = errors.New("errors found during validation")

func validate(log *slog.Logger, cfg *config.Config, shutdown chan os.Signal) error {
	if !cfg.PersistedOperations.Enabled {
		err := fmt.Errorf("persisted operations must be enabled to run validate")
		log.Error("Error running validations", "err", err)
		return err
	}

	remoteLoader, err := persisted_operations.RemoteLoaderFromConfig(cfg.PersistedOperations, log)
	if err != nil && !errors.Is(err, persisted_operations.ErrNoRemoteLoaderSpecified) {
		log.Warn("Error initializing remote loader", "err", err)
	}

	po, err := persisted_operations.NewPersistedOperations(log, cfg.PersistedOperations, persisted_operations.NewLocalDirLoader(cfg.PersistedOperations), remoteLoader)
	if err != nil {
		log.Error("Error initializing Persisted Operations", "err", err)
		return nil
	}

	schemaProvider, err := schema.NewSchema(cfg.Schema, log)
	if err != nil {
		log.Error("Error initializing schema", "err", err)
		return nil
	}

	protectHandler, err := protect.NewGraphQLProtect(log, cfg, po, schemaProvider, nil)
	if err != nil {
		log.Error("Error initializing GraphQL Protect", "err", err)
		return err
	}

	errs := po.Validate(protectHandler.ValidateQuery)
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
