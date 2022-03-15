package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	gosteamauth "github.com/TeddiO/GoSteamAuth/src"
)

func main() {
	serverRouter := http.NewServeMux()
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      serverRouter,
	}

	serverRouter.HandleFunc("/", ExamplePage)
	serverRouter.HandleFunc("/process", ProcessSteamLogin)
	log.Fatal(server.ListenAndServe())
}

// ExamplePage is just your average default page handler. In this example
// We're just using the one liner to redirect the client and at the same time notify
// the openid provider (Steam) where to return us.
func ExamplePage(resp http.ResponseWriter, req *http.Request) {
	queryString := req.URL.Query()

	if queryString.Get("login") == "true" {
		gosteamauth.RedirectClient(resp, req, gosteamauth.BuildQueryString("http://localhost:8080/process"))
		return
	}

	loadingTemplate := template.New("index.html")
	loadingTemplate, _ = template.ParseFiles("index.html")

	if err := loadingTemplate.Execute(resp, nil); err != nil {
		fmt.Println(err)
	}

}

// ProcessSteamLogin is where the real magic happens in terms of validation.
// As long as isValid is true we should always be able to trust the SteamID64 returned.
func ProcessSteamLogin(resp http.ResponseWriter, req *http.Request) {
	queryString, _ := url.ParseQuery(req.URL.RawQuery)

	// Due to ParseQuery() returning a url.Values in form map[string][]string we're going to
	// convert that data structure to map[string]string so we can validate.
	queryMap := gosteamauth.ValuesToMap(queryString)

	steamID64, isValid, err := gosteamauth.ValidateResponse(queryMap)
	if err != nil {
		fmt.Fprintf(resp, "Failed to log in\nError: %s", err)
		return
	}

	// The below is purely for demonstrative purposes, typically you would move the
	// client on away from this page, set cookies / sessions and so on.

	if isValid {
		fmt.Fprintf(resp, "Successfully logged in!\nSteamID: %s", steamID64)
	} else {
		io.WriteString(resp, "Failed to log in.")
	}

}
