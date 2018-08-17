# atlassian-jwt

<h1 align="center">
  <br>
  <img src="https://i.imgur.com/aG5AhlH.png" alt="atlassian-jwt">
  <br>
  JWT library for Atlassian in Go.
  <br>
  <br>
</h1>

<p align="center">
<a href="https://travis-ci.org/rbriski/atlassian-jwt"><img src="https://travis-ci.org/rbriski/atlassian-jwt.svg?branch=master" alt="Build Status"></a>
</p>

`atlassian-jwt` is a library that makes it easy to authenticate with JIRA from a variety of app types.

## Installation

```bash
go get github.com/rbriski/atlassian-jwt
```

## Usage (with go-jira)

```go
import (
    jira "github.com/andygrunwald/go-jira"
    jwt "github.com/rbriski/atlassian-jwt"
)

c := &jwt.Config{
    Key: "some_key",
    ClientKey: "some_client_key",
    SharedSecret: "so_freakin_secret",
    BaseUrl: "http://example.com",
}

// Pass the JWT client into the library client
jiraClient, _ := jira.NewClient(c.Client(), c.BaseURL)
```

## Examples

There are a number of different ways that an app can authenticate with JIRA.  Right now, `atlassian-jwt` only handles JWT authentication as an add-on.  

Using ngrok, you can spin up a [working example](https://github.com/rbriski/atlassian-jwt/blob/master/examples/jwt/main.go) to authenticate with.

```bash
> cd examples/jwt
> BASE_URL=https://<some_string>.ngrok.io go run main.go
```
