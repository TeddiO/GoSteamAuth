package gosteamauth

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	provider      = "https://steamcommunity.com/openid/login"
	steamIDPrefix = "https://steamcommunity.com/openid/id/"
	openIDNS      = "http://specs.openid.net/auth/2.0"
	identifierSel = "http://specs.openid.net/auth/2.0/identifier_select"
)

var HTTPTimeout = 15 * time.Second

func init() {
	if v := os.Getenv("STEAM_AUTH_HTTP_TIMEOUT"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			HTTPTimeout = time.Duration(secs) * time.Second
		}
	}
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

var httpClient httpDoer = &http.Client{
	Timeout: HTTPTimeout,
}

// BuildQueryString uses a url.Values map to correctly structure the paramters to send to Steam
// Strictly speaking we only care that the Scheme is either http or https.
// Any other validation should really be done before using this function.
func BuildQueryString(responsePath string) string {
	if !strings.HasPrefix(responsePath, "http") {
		log.Fatal("http was not found in the responsePath!")
	}

	if !strings.HasPrefix(responsePath, "https") {
		log.Println("https isn't being used! Is this intentional?")
	}

	values := url.Values{}
	values.Set("openid.mode", "checkid_setup")
	values.Set("openid.return_to", responsePath)
	values.Set("openid.realm", responsePath)
	values.Set("openid.ns", openIDNS)
	values.Set("openid.identity", identifierSel)
	values.Set("openid.claimed_id", identifierSel)

	return values.Encode()
}

// ValidateResponse is the real chunk of work that goes on. When the client comes back to our site
// we need to take what they give us in the query string and hit up Steam directly
// to verify what we're being provided with is valid.
// If we end up with "is_valid:true" response from the Steam then isValid will always return true.
// In any other situation (credential failure, error etc) isValid will always return false and we aim
// to provide a descriptive error where possible.
func ValidateResponse(results map[string]string) (steamID64 string, isValid bool, err error) {
	values := url.Values{}
	values.Set("openid.mode", "check_authentication")
	values.Set("openid.assoc_handle", results["openid.assoc_handle"])
	values.Set("openid.signed", results["openid.signed"])
	values.Set("openid.sig", results["openid.sig"])
	values.Set("openid.ns", results["openid.ns"])

	signedParams := strings.Split(results["openid.signed"], ",")
	for _, key := range signedParams {
		fullKey := "openid." + key
		values.Set(fullKey, results[fullKey])
	}

	reqURL, err := url.Parse(provider)
	if err != nil {
		return "", false, err
	}
	reqURL.RawQuery = values.Encode()

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return "", false, err
	}

	validationResp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to validate %s. Error: %s", results["openid.claimed_id"], err)
		return "", false, err
	}
	defer validationResp.Body.Close()

	returnedBytes, err := io.ReadAll(validationResp.Body)
	if err != nil {
		return "", false, err
	}

	if !strings.Contains(string(returnedBytes), "is_valid:true") {
		return "", false, errors.New("openid validation failed: steam returned is_valid:false")
	}

	claimedID := results["openid.claimed_id"]
	identity := results["openid.identity"]

	if claimedID == "" || identity == "" {
		return "", false, errors.New("openid validation failed: missing claimed_id or identity")
	}

	if claimedID != identity {
		return "", false, errors.New("openid validation failed: claimed_id and identity mismatch")
	}

	if !strings.HasPrefix(claimedID, steamIDPrefix) {
		return "", false, errors.New("openid validation failed: identifier is not a steam openid")
	}

	steamID := strings.TrimPrefix(claimedID, steamIDPrefix)
	return steamID, true, nil
}

// RedirectClient is a helper function that does the redirection to Steam with the
// correct properties on our behalf.
func RedirectClient(w http.ResponseWriter, r *http.Request, queryString string) {
	u, _ := url.Parse(provider)
	u.RawQuery = queryString

	w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// ValuesToMap is a boilerplate function designed to convert the results of a url.Values
// in to a readable map[string]string for ValidateResponse.
func ValuesToMap(fakeMap url.Values) map[string]string {
	returnMap := map[string]string{}
	for k, v := range fakeMap {
		returnMap[k] = v[0]
	}

	return returnMap
}
