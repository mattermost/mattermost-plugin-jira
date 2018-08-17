package jwt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// Config holds the configuration information for JWT operation
// between an app and JIRA
type Config struct {
	// Key holds the app key described in the Atlassian Connect
	// JSON file
	Key string

	// ClientKey holds the key that JIRA returns to validate JWT
	// tokens from Jira
	ClientKey string

	// SharedSecret is the signing secret for the Authorization header
	SharedSecret string

	// BaseURL is the base URL of the JIRA instance
	BaseURL string
}

// AtlassianClaims are all mandatory claims for Atlassian JWT
type AtlassianClaims struct {
	QSH string `json:"qsh"`

	jwt.StandardClaims
}

// A AuthSetter is anything that can set the authorization header
// on an http.Request
type AuthSetter interface {
	// SetAuthHeader takes a request pointer and sets the
	// Authorization header with a valid Atlassian JWT
	SetAuthHeader(*http.Request) error
}

// Claims returns a valid set of claims for creating
// an Atlassian JWT
func (c *Config) Claims(qsh string) *AtlassianClaims {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(180 * time.Second)

	return &AtlassianClaims{
		qsh,
		jwt.StandardClaims{
			IssuedAt:  issuedAt.Unix(),
			ExpiresAt: expiresAt.Unix(),
			Issuer:    c.Key,
		},
	}
}

// Client returns an *http.Client that makes requests that are authenticated
// using Atlassian JWT authentication
func (c *Config) Client() *http.Client {
	return &http.Client{
		Transport: &Transport{
			Config: c,
		},
	}
}

// Token returns an unsigned Atlassian JWT
func (c *Config) Token(r *http.Request) *jwt.Token {
	qsh := c.QSH(r)
	claims := c.Claims(qsh)
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
}

// SetAuthHeader takes a request pointer and sets the
// Authorization header with a valid Atlassian JWT
func (c *Config) SetAuthHeader(r *http.Request) error {
	token := c.Token(r)
	ss, err := token.SignedString([]byte(c.SharedSecret))
	if err != nil {
		return errors.Wrap(err, "failed to sign token")
	}

	r.Header.Set("Authorization", "JWT "+ss)
	return nil
}

// QSH returns the query string hash for this request
// https://developer.atlassian.com/cloud/bitbucket/query-string-hash/
func (c *Config) QSH(req *http.Request) string {
	// Uppercase method
	method := strings.ToUpper(req.Method)

	// Path can not contain &
	path := strings.Replace(req.URL.Path, "&", "%26", -1)
	params := encodeQuery(req.URL.Query())

	// Join method, path and params with &
	canonicalURL := strings.Join([]string{method, path, params}, "&")

	// SHA-256 encoding
	h := sha256.New()
	h.Write([]byte(canonicalURL))

	// Must return the hash as hex
	return hex.EncodeToString(h.Sum(nil))
}

// encodeQuery uses the QSH description from
// https://developer.atlassian.com/cloud/bitbucket/query-string-hash/
func encodeQuery(vals url.Values) string {
	// Empty params still must be treated as a value
	if len(vals) == 0 {
		return ""
	}

	var buf bytes.Buffer
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}

	// Keys must be sorted
	sort.Strings(keys)
	for _, k := range keys {

		// Exclude any JWT keys
		if strings.ToUpper(k) == "JWT" {
			continue
		}

		vs := vals[k]

		// Escaped encoding is upper case
		// QueryEscape does this for us
		encKey := url.QueryEscape(k)

		// QueryEscape encodes spaces as +.  According to Atlassian, they
		// must be encoded as %20
		prefix := strings.Replace(encKey, "+", "%20", -1) + "="

		encodedVals := make([]string, 0, len(keys))

		// Repeated parameters must be sorted
		sort.Strings(vs)
		for _, v := range vs {
			encVal := url.QueryEscape(v)
			encodedVals = append(encodedVals, strings.Replace(encVal, "+", "%20", -1))
		}
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(prefix)

		// Repeated parameters to be in comma-delimited list
		buf.WriteString(strings.Join(encodedVals, ","))
	}

	return buf.String()
}

// Transport is a http.RoundTripper for tagging requests
// to Atlassian with a JWT auth header
type Transport struct {
	// SetAuth sets the
	// Authorization headers.
	Config AuthSetter

	// Base is the base RoundTripper used to make HTTP requests.
	// If nil, http.DefaultTransport is used.
	Base http.RoundTripper
}

// RoundTrip authenticates the request with a JWT token
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Config == nil {
		return nil, errors.New("jwt: Transport's config is nil")
	}

	req2 := cloneRequest(req)
	err := t.Config.SetAuthHeader(req2)
	if err != nil {
		return nil, err
	}

	return t.base().RoundTrip(req2)
}

func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}
