package problem

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestNewAndWithErrors(t *testing.T) {
    fieldErrors := []FieldError{{Field: "name", Message: "required"}}
    p := New(http.StatusBadRequest, "bad-request", "Bad Request", "details").WithErrors(fieldErrors)

    if got, want := p.Type, BaseURI+"/bad-request"; got != want {
        t.Fatalf("unexpected type: got %q want %q", got, want)
    }
    if p.Status != http.StatusBadRequest {
        t.Fatalf("unexpected status: %d", p.Status)
    }
    if len(p.Errors) != 1 || p.Errors[0] != fieldErrors[0] {
        t.Fatalf("errors not set: %+v", p.Errors)
    }
}

func TestProblemWrite(t *testing.T) {
    resp := httptest.NewRecorder()
    p := BadRequest("invalid")
    p.Write(resp)

    if resp.Code != http.StatusBadRequest {
        t.Fatalf("unexpected status: %d", resp.Code)
    }
    if got := resp.Header().Get("Content-Type"); got != ContentType {
        t.Fatalf("missing content type: %s", got)
    }

    var decoded Problem
    if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
        t.Fatalf("failed to decode body: %v", err)
    }
    if decoded.Title != "Bad Request" || decoded.Detail != "invalid" {
        t.Fatalf("unexpected payload: %+v", decoded)
    }
}
