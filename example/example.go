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

func ProcessSteamLogin(resp http.ResponseWriter, req *http.Request) {
	queryString, _ := url.ParseQuery(req.URL.RawQuery)

	// Lack of generics means we're joining to transform the queryString (Type: Values) in to a Map.
	queryMap := gosteamauth.ValuesToMap(queryString)

	steamID64, isValid, err := gosteamauth.ValidateResponse(queryMap)
	if err != nil {
		fmt.Fprintf(resp, "Failed to log in\nError: %s", err)
		return
	}

	if isValid {
		fmt.Fprintf(resp, "Successfully logged in!\nSteamID: %s", steamID64)
	} else {
		io.WriteString(resp, "Failed to log in.")
	}

}
