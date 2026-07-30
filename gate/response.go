package gate

// Response represents an authorization response
type Response struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewAllowed creates an allowed response
func NewAllowed() *Response {
	return &Response{Allowed: true}
}

// NewDenied creates a denied response with a reason
func NewDenied(reason string) *Response {
	return &Response{
		Allowed: false,
		Reason:  reason,
	}
}

// NewDeniedWithMessage creates a denied response with a reason and message
func NewDeniedWithMessage(reason, message string) *Response {
	return &Response{
		Allowed: false,
		Reason:  reason,
		Message: message,
	}
}

// IsAllowed checks if the response is allowed
func (r *Response) IsAllowed() bool {
	return r.Allowed
}

// IsDenied checks if the response is denied
func (r *Response) IsDenied() bool {
	return !r.Allowed
}
