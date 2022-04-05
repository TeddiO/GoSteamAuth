package gosteamauth

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var validCredRx *regexp.Regexp
var steamRx *regexp.Regexp
var provider string = "https://steamcommunity.com/openid/login"

func init() {
	validCredRx = regexp.MustCompile("is_valid:true")
	steamRx = regexp.MustCompile(`https://steamcommunity\.com/openid/id/(\d+)`)
}

// StringMapToString is a utility function that aims to efficiently build a query string
// with a tiny footprint. theMap is expected to be a map of key strings with a value type of a string.
func StringMapToString(theMap map[string]string) string {

	mapLength := len(theMap)
	strSeparator := "&"
	i := 1

	var builder strings.Builder
	builder.Grow(66) // We already roughly know our base size.

	for k, v := range theMap {

		if i == mapLength {
			strSeparator = ""
		}

		i++

		fmt.Fprintf(&builder, "%s=%s%s", k, url.QueryEscape(v), strSeparator)
	}

	return builder.String()
}

// BuildQueryString is more or less building up a query string to be passed when reaching
// Steam's openid 2.0 provider (or technically any openid 2.0 provider). We only care
// that the Scheme is either http or https. Any other validation should really be done
// before using this function.
func BuildQueryString(responsePath string) string {

	if responsePath[0:4] != "http" {
		log.Fatal("http was not found in the responsePath!")
	}

	if responsePath[4:5] != "s" {
		log.Println("https isn't being used! Is this intentional?")
	}

	// Even though the below URLs no longer function, the oauth 2.0 process formally calls
	// for them and Valve actively checks for their presence.
	openIdParameters := map[string]string{
		"openid.mode":       "checkid_setup",
		"openid.return_to":  responsePath,
		"openid.realm":      responsePath,
		"openid.ns":         "http://specs.openid.net/auth/2.0",
		"openid.identity":   "http://specs.openid.net/auth/2.0/identifier_select",
		"openid.claimed_id": "http://specs.openid.net/auth/2.0/identifier_select",
	}

	return StringMapToString(openIdParameters)
}

// ValidateResponse is the real chunk of work that goes on. When the client comes back to our site
// we need to take what they give us in the query string and hit up the openid 2.0 provider directly
// to verify what we're being provided with is well, valid.
// If we end up with "is_valid:true" response from the Steam then isValid will always return true.
// In any other situation (credential failure, error etc) isValid will always return false.
// Takes a map[string]string to be agnostic among various http clients that exist out there
func ValidateResponse(results map[string]string) (steamID64 string, isValid bool, err error) {

	openIdValidation := map[string]string{
		"openid.assoc_handle": results["openid.assoc_handle"],
		"openid.signed":       results["openid.signed"],
		"openid.sig":          results["openid.sig"],
		"openid.ns":           results["openid.ns"],
		"openid.mode":         "check_authentication",
	}

	signedParams := strings.Split(results["openid.signed"], ",")

	for _, value := range signedParams {
		item := fmt.Sprintf("openid.%s", value)
		if _, exists := openIdValidation[item]; !exists {
			openIdValidation[item] = results[item]
		}
	}

	urlObj, err := url.Parse(provider)
	if err != nil {
		return "", false, err
	}

	urlObj.RawQuery = StringMapToString(openIdValidation)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	validationResp, err := httpClient.Get(urlObj.String())
	if err != nil {
		log.Printf("Failed to validate %s. Error: %s ", results["openid.claimed_id"], err)
		return "", false, err
	}

	defer validationResp.Body.Close()
	returnedBytes, err := ioutil.ReadAll(validationResp.Body)
	if err != nil {
		return "", false, err
	}

	if validCredRx.MatchString(string(returnedBytes)) == true {
		return steamRx.FindStringSubmatch(results["openid.claimed_id"])[1], true, nil
	}

	return "", false, nil
}

// RedirectClient is a helper function that does the redirection to Steam with the
// correct properties on our behalf. Pass it the appropriate http request / response objects
// alongside the queryString and it'll get the user to the right place
func RedirectClient(response http.ResponseWriter, request *http.Request, queryString string) {
	returnUrlObject, _ := url.Parse(provider)
	returnUrlObject.RawQuery = queryString

	response.Header().Add("Content-Type", "application/x-www-form-urlencoded")
	http.Redirect(response, request, returnUrlObject.String(), 303)
	return
}

// ValuesToMap is a boilerplate function designed to convert the results of a url.Values
// in to a readable map[string]string for ValidateResponse.
// We don't get duplicate query keys supplied normally - but we'll always take the first one anyways
func ValuesToMap(fakeMap url.Values) map[string]string {
	returnMap := map[string]string{}
	for k, v := range fakeMap {
		returnMap[k] = v[0]
	}

	return returnMap
}
