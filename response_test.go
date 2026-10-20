package vroomy

import "testing"

func TestResponseConstructors(t *testing.T) {
	var payload = &struct{ Message string }{Message: "created"}
	var response = NewResponse(201, "application/json", payload)
	if response.StatusCode != 201 || response.ContentType != "application/json" || response.Value != payload || response.Adopted || response.Callback != "" {
		t.Fatalf("NewResponse() = %#v", response)
	}

	var adopted = NewAdopedtResponse()
	if !adopted.Adopted || adopted.StatusCode != 0 || adopted.ContentType != "" || adopted.Value != nil || adopted.Callback != "" {
		t.Fatalf("NewAdopedtResponse() = %#v", adopted)
	}
}
