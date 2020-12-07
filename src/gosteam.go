package gosteamauth

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

func StringMapToString(theMap map[string]string) string {

	mapLength := len(theMap)
	strSeparator := ","
	i := 1

	var builder strings.Builder

	for k, v := range theMap {

		if i == mapLength {
			strSeparator = ""
		}

		i++

		fmt.Fprintf(&builder, "%s=%s%s", k, v, strSeparator)
	}

	return builder.String()
}

func ConstructURL(responsePath string) string {

	if responsePath[0:4] != "http" {
		log.Fatal("http was not found in the responsePath!")
	}

	// Omit the other parameters we used to put here.
	// Those links are long since dead.
	openIdParameters := map[string]string{
		"openid.mode":      "checkid_setup",
		"openid.return_to": responsePath,
		"openid.realm":     responsePath,
	}

	return url.QueryEscape(StringMapToString(openIdParameters))
}

func ValidateResponse(something map[string]string) {

}

func RedirectClient() {

}
