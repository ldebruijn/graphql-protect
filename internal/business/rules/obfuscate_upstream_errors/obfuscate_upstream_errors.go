package obfuscate_upstream_errors // nolint:revive

type ObfuscateUpstreamErrors struct {
	enabled bool
}

func NewObfuscateUpstreamErrors(obfuscateUpstreamErrors bool) *ObfuscateUpstreamErrors {
	return &ObfuscateUpstreamErrors{
		enabled: obfuscateUpstreamErrors,
	}
}

func (a *ObfuscateUpstreamErrors) ProcessBody(payload map[string]interface{}) map[string]interface{} {
	redactedErrorArray := []map[string]interface{}{
		{
			"message": "Error(s) redacted",
		},
	}

	if payload["errors"] != nil {
		payload["errors"] = redactedErrorArray
	}

	return payload
}

func (a *ObfuscateUpstreamErrors) Enabled() bool {
	return a.enabled
}
