package accesslogging

type accessLog struct {
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	Payload       string                 `json:"payload,omitempty"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
}

func (a *accessLog) WithOperationName(name string) {
	a.OperationName = name
}

func (a *accessLog) WithVariables(variables map[string]interface{}) {
	a.Variables = variables
}

func (a *accessLog) WithPayload(payload string) {
	a.Payload = payload
}

func (a *accessLog) WithHeaders(headers map[string]interface{}) {
	a.Headers = headers
}
