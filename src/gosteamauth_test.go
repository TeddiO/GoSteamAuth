package gosteamauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeHTTPClient struct {
	body string
	err  error
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

func validResults() map[string]string {
	steamID := "12345678901234567"
	claimed := steamIDPrefix + steamID

	return map[string]string{
		"openid.assoc_handle": "assoc",
		"openid.signed":       "claimed_id",
		"openid.sig":          "sig",
		"openid.ns":           openIDNS,
		"openid.claimed_id":   claimed,
		"openid.identity":     claimed,
	}
}

func TestValidateResponse_Valid(t *testing.T) {
	oldClient := httpClient
	httpClient = &fakeHTTPClient{body: "is_valid:true\n"}
	defer func() { httpClient = oldClient }()

	steamID, ok, err := ValidateResponse(validResults())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected valid response")
	}
	if steamID != "12345678901234567" {
		t.Fatalf("unexpected steamID: %s", steamID)
	}
}

func TestValidateResponse_Invalid(t *testing.T) {
	oldClient := httpClient
	httpClient = &fakeHTTPClient{body: "is_valid:false\n"}
	defer func() { httpClient = oldClient }()

	_, ok, _ := ValidateResponse(validResults())
	if ok {
		t.Fatalf("expected invalid response")
	}
}

func TestValidateResponse_NetworkErrorFailsClosed(t *testing.T) {
	oldClient := httpClient
	httpClient = &fakeHTTPClient{err: errors.New("network error")}
	defer func() { httpClient = oldClient }()

	_, ok, err := ValidateResponse(validResults())
	if ok {
		t.Fatalf("expected network error to fail closed")
	}
	if err == nil {
		t.Fatalf("expected error to be returned on network failure")
	}
}

func TestValidateResponse_TimeoutFailsClosed(t *testing.T) {
	oldClient := httpClient
	httpClient = &fakeHTTPClient{err: context.DeadlineExceeded}
	defer func() { httpClient = oldClient }()

	_, ok, err := ValidateResponse(validResults())
	if ok {
		t.Fatalf("expected timeout to fail closed")
	}
	if err == nil {
		t.Fatalf("expected timeout error to be returned")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got: %v", err)
	}
}

func TestValidateResponse_IdentityMismatchFailsClosed(t *testing.T) {
	oldClient := httpClient
	httpClient = &fakeHTTPClient{body: "is_valid:true\n"}
	defer func() { httpClient = oldClient }()

	results := validResults()
	results["openid.identity"] = steamIDPrefix + "00000000000000000"

	_, ok, err := ValidateResponse(results)
	if ok {
		t.Fatalf("expected identity mismatch to fail closed")
	}
	if err == nil {
		t.Fatalf("expected error on identity mismatch")
	}
}
