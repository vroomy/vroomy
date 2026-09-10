package vroomy

// NewResponse will return a new Response
func NewResponse(statusCode int, contentType string, value interface{}) *Response {
	r := makeResponse(statusCode, contentType, value)
	return &r
}

// NewAdopedtResponse returns a Response with Adopted set to true.
// The exported name retains its historical spelling.
func NewAdopedtResponse() *Response {
	return &Response{Adopted: true}
}

// Response holds legacy response metadata. The current plugin handler resolver
// does not consume it; handlers write through httpserve.Context instead.
type Response struct {
	StatusCode  int
	ContentType string
	Value       interface{}

	// Legacy adoption and callback metadata.
	Adopted  bool
	Callback string
}

func makeResponse(statusCode int, contentType string, value interface{}) (r Response) {
	r.StatusCode = statusCode
	r.ContentType = contentType
	r.Value = value
	return
}
