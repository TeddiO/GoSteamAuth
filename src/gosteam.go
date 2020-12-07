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

func ConstructURL(responsePath string) string {

	if responsePath[0:4] != "http" {
		log.Fatal("http was not found in the responsePath!")
	}

	if responsePath[5:5] != "s" {
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

	urlObj, _ := url.Parse(provider)
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

func RedirectClient(response http.ResponseWriter, request *http.Request, returnPath string) {
	response.Header().Add("Content-Type", "application/x-www-form-urlencoded")
	http.Redirect(response, request, fmt.Sprintf("%s?%s", provider, returnPath), 303)
	return
}

// Utility function
func ValuesToMap(fakeMap url.Values) map[string]string {
	returnMap := map[string]string{}
	for k, v := range fakeMap {
		returnMap[k] = v[0]
	}

	return returnMap
}
