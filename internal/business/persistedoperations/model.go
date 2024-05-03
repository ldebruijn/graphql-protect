package persistedoperations

import "strings"

type PersistedOperation struct {
	Operation string
	Name      string
}

func NewPersistedOperation(operation string) PersistedOperation {
	name := extractOperationNameFromPersistedOperation(operation)
	return PersistedOperation{
		Operation: operation,
		Name:      name,
	}
}

func extractOperationNameFromPersistedOperation(payload string) string {
	firstSpace := strings.Index(payload, " ")
	firstBracket := strings.Index(payload, "{")
	firstParenthesis := strings.Index(payload, "(")

	until := firstBracket
	if firstParenthesis < firstBracket {
		until = firstParenthesis
	}

	if firstSpace > until {
		return ""
	}

	return payload[firstSpace+1 : until]
}
