package codexrpc

import "encoding/json"

// Message is a generic JSON-RPC 2.0 envelope used for requests, notifications, and responses.
// The app-server wire format may omit jsonrpc, so this field is optional.
type Message struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError matches the JSON-RPC error object shape.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func NewRequest(id any, method string, params any) (*Message, error) {
	rawParams, err := encodeRaw(params)
	if err != nil {
		return nil, err
	}
	return &Message{JSONRPC: "2.0", ID: id, Method: method, Params: rawParams}, nil
}

func NewNotification(method string, params any) (*Message, error) {
	rawParams, err := encodeRaw(params)
	if err != nil {
		return nil, err
	}
	return &Message{JSONRPC: "2.0", Method: method, Params: rawParams}, nil
}

func NewResponse(id any, result any) (*Message, error) {
	rawResult, err := encodeRaw(result)
	if err != nil {
		return nil, err
	}
	return &Message{JSONRPC: "2.0", ID: id, Result: rawResult}, nil
}

func NewErrorResponse(id any, code int, message string, data any) (*Message, error) {
	rawData, err := encodeRaw(data)
	if err != nil {
		return nil, err
	}
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    rawData,
		},
	}, nil
}

func (m *Message) IsResponse() bool {
	if m == nil {
		return false
	}
	return m.ID != nil && m.Method == "" && (len(m.Result) > 0 || m.Error != nil)
}

func (m *Message) IsRequest() bool {
	if m == nil {
		return false
	}
	return m.ID != nil && m.Method != ""
}

func (m *Message) IsNotification() bool {
	if m == nil {
		return false
	}
	return m.ID == nil && m.Method != ""
}

func encodeRaw(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
