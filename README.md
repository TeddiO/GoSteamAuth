# GoSteamAuth
A small set of utility functions to quickly process Steam OpenID 2.0 logins.
Cousin to the Python library designed to do the same thing: [pySteamSignIn](https://github.com/TeddiO/pySteamSignIn)

Similar to why the Python edition was wrote there's no straightforwards Steam authentication flow for Go, so this exists to fulfil the same purpose. Another language, same idea!

## Installing

To install: `go get github.com/TeddiO/GoSteamAuth/src`
## Authentication

Assuming you're using the typical `net/http` package, then the entire process is effectively a one-liner:

```go
func ExamplePage(response http.ResponseWriter, request *http.Request) {
    gosteamauth.RedirectClient(response, request, gosteamauth.ConstructURL("http://localhost:8080/process"))
    return
}
```
(Replace `http://localhost:8080/process` with whatever your landing point for the user would be)

This redirects the user to `https://steamcommunity.com/openid/login` with all the required parameters and takes them through the auth flow on Steam's end. If they successfully log in then they'll be returned to `http://localhost:8080/process`.

### Verifying the request

When the user returns, they bring with them a response from Steam that tacks on a sizeable query string which is used by us to validate what the client is bringing with them is valid.
```go
// Some function signature that conforms to response / request args
queryString, _ := url.ParseQuery(request.URL.RawQuery)
queryMap := gosteamauth.ValuesToMap(queryString)

steamID64, isValid, err := gosteamauth.ValidateResponse(queryMap)
if err != nil {
    fmt.Fprintf(response, "Failed to log in\nError: %s", err)
    return
}
    
if isValid {
    fmt.Fprintf(response, "Successfully logged in!\nSteamID: %s", steamID64)
} else {
    io.WriteString(response, "Failed to log in.")
}
```

And that's it! A full example is available [here](https://github.com/TeddiO/GoSteamAuth/blob/master/example/example.go)