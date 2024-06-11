package accesslogging

import "encoding/json"

type accesslog struct {
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	Payload       string                 `json:"payload,omitempty"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
}

func (a *accesslog) WithOperationName(name string) {
	a.OperationName = name
}

func (a *accesslog) WithVariables(variables map[string]interface{}) {
	a.Variables = variables
}

func (a *accesslog) WithPayload(payload string) {
	a.Payload = payload
}

func (a *accesslog) WithHeaders(headers map[string]interface{}) {
	a.Headers = headers
}

func (a *accesslog) JSON() ([]byte, error) {
	return json.Marshal(a)
}
