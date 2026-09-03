// Package sso signs a person in with the account their school already gave
// them, over OpenID Connect's authorization code flow.
package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Provider is one school's configured identity provider.
type Provider struct {
	Issuer       string
	ClientID     string
	ClientSecret string
}

// Login is what a browser is sent away with, and what the callback needs back.
type Login struct {
	URL      string
	State    string
	Nonce    string
	Verifier string
}

// Person is who the provider says signed in.
type Person struct {
	Subject string
	Email   string
	Name    string
}

// Client talks to providers. The discovery document is cached because it
// changes about as often as the provider's domain does.
type Client struct {
	HTTP  *http.Client
	cache sync.Map
}

type discovery struct {
	Issuer        string `json:"issuer"`
	AuthURL       string `json:"authorization_endpoint"`
	TokenURL      string `json:"token_endpoint"`
	UserInfoURL   string `json:"userinfo_endpoint"`
	fetchedAt     time.Time
	CodeChallenge []string `json:"code_challenge_methods_supported"`
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c *Client) discover(ctx context.Context, issuer string) (discovery, error) {
	issuer = strings.TrimSuffix(strings.TrimSpace(issuer), "/")
	if cached, ok := c.cache.Load(issuer); ok {
		found := cached.(discovery)
		if time.Since(found.fetchedAt) < time.Hour {
			return found, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return discovery{}, err
	}
	res, err := c.client().Do(req)
	if err != nil {
		return discovery{}, fmt.Errorf("sso: %s did not answer", issuer)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return discovery{}, fmt.Errorf("sso: %s is not an OpenID provider", issuer)
	}

	var found discovery
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&found); err != nil {
		return discovery{}, fmt.Errorf("sso: %s answered with something unreadable", issuer)
	}
	// The issuer in the document is the one that counts, and it must be the
	// one we asked, or the provider is not who the school named.
	if strings.TrimSuffix(found.Issuer, "/") != issuer {
		return discovery{}, fmt.Errorf("sso: %s calls itself %s", issuer, found.Issuer)
	}
	if found.AuthURL == "" || found.TokenURL == "" {
		return discovery{}, fmt.Errorf("sso: %s does not offer the sign-in endpoints", issuer)
	}
	found.fetchedAt = time.Now()
	c.cache.Store(issuer, found)
	return found, nil
}

// Start builds the URL to send the browser to, with the one-time values that
// tie the answer to this request.
func (c *Client) Start(ctx context.Context, p Provider, redirectURI string) (Login, error) {
	doc, err := c.discover(ctx, p.Issuer)
	if err != nil {
		return Login{}, err
	}

	state, nonce, verifier := random(), random(), random()+random()
	sum := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {p.ClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	join := "?"
	if strings.Contains(doc.AuthURL, "?") {
		join = "&"
	}
	return Login{
		URL: doc.AuthURL + join + query.Encode(), State: state, Nonce: nonce, Verifier: verifier,
	}, nil
}

type tokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// Exchange turns the code into who signed in.
//
// The ID token's signature is not checked, and does not need to be: it comes
// straight back from the provider's token endpoint over TLS, which OpenID
// Connect allows in place of verifying the signature. The claims are still
// checked, and the nonce ties the answer to the request that started it.
func (c *Client) Exchange(ctx context.Context, p Provider, code, verifier, nonce, redirectURI string) (Person, error) {
	doc, err := c.discover(ctx, p.Issuer)
	if err != nil {
		return Person{}, err
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {p.ClientID},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, doc.TokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return Person{}, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")
	if p.ClientSecret != "" {
		req.SetBasicAuth(url.QueryEscape(p.ClientID), url.QueryEscape(p.ClientSecret))
	}

	res, err := c.client().Do(req)
	if err != nil {
		return Person{}, fmt.Errorf("sso: the provider did not answer")
	}
	defer res.Body.Close()

	var token tokenResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&token); err != nil {
		return Person{}, fmt.Errorf("sso: the provider answered with something unreadable")
	}
	if token.Error != "" {
		return Person{}, fmt.Errorf("sso: %s", firstOf(token.Description, token.Error))
	}
	if res.StatusCode != http.StatusOK || token.IDToken == "" {
		return Person{}, fmt.Errorf("sso: the provider refused the sign-in")
	}

	claims, err := claimsOf(token.IDToken)
	if err != nil {
		return Person{}, err
	}
	if strings.TrimSuffix(claims.Issuer, "/") != strings.TrimSuffix(p.Issuer, "/") {
		return Person{}, fmt.Errorf("sso: the answer came from somewhere else")
	}
	if !claims.audienceIs(p.ClientID) {
		return Person{}, fmt.Errorf("sso: the answer was meant for another application")
	}
	if claims.Nonce != nonce {
		return Person{}, fmt.Errorf("sso: this sign-in does not match the one that started")
	}
	if claims.Expiry > 0 && time.Now().After(time.Unix(claims.Expiry, 0)) {
		return Person{}, fmt.Errorf("sso: the sign-in took too long; try again")
	}
	if claims.Subject == "" {
		return Person{}, fmt.Errorf("sso: the provider did not say who signed in")
	}

	person := Person{Subject: claims.Subject, Email: strings.ToLower(claims.Email), Name: claims.Name}
	// Some providers keep the address for the userinfo endpoint.
	if person.Email == "" && doc.UserInfoURL != "" && token.AccessToken != "" {
		person = c.fill(ctx, doc.UserInfoURL, token.AccessToken, person)
	}
	if person.Email == "" {
		return Person{}, fmt.Errorf("sso: the provider did not share an email address")
	}
	if person.Name == "" {
		person.Name = person.Email
	}
	return person, nil
}

func (c *Client) fill(ctx context.Context, endpoint, accessToken string, person Person) Person {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return person
	}
	req.Header.Set("authorization", "Bearer "+accessToken)
	res, err := c.client().Do(req)
	if err != nil {
		return person
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return person
	}

	var info struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&info); err != nil {
		return person
	}
	if person.Email == "" {
		person.Email = strings.ToLower(strings.TrimSpace(info.Email))
	}
	if person.Name == "" {
		person.Name = strings.TrimSpace(info.Name)
	}
	return person
}

type claims struct {
	Issuer   string          `json:"iss"`
	Subject  string          `json:"sub"`
	Audience json.RawMessage `json:"aud"`
	Nonce    string          `json:"nonce"`
	Email    string          `json:"email"`
	Name     string          `json:"name"`
	Expiry   int64           `json:"exp"`
}

// audienceIs reads aud in both shapes the spec allows: one string, or a list.
func (c claims) audienceIs(clientID string) bool {
	var one string
	if err := json.Unmarshal(c.Audience, &one); err == nil {
		return one == clientID
	}
	var many []string
	if err := json.Unmarshal(c.Audience, &many); err != nil {
		return false
	}
	for _, entry := range many {
		if entry == clientID {
			return true
		}
	}
	return false
}

func claimsOf(idToken string) (claims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return claims{}, fmt.Errorf("sso: the provider's token is not readable")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims{}, fmt.Errorf("sso: the provider's token is not readable")
	}
	var out claims
	if err := json.Unmarshal(body, &out); err != nil {
		return claims{}, fmt.Errorf("sso: the provider's token is not readable")
	}
	return out, nil
}

// AddressAllowed keeps sign-in to the domains a school named. No domains named
// means anybody the provider vouches for.
func AddressAllowed(email string, domains []string) bool {
	if len(domains) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	host := strings.ToLower(email[at+1:])
	for _, domain := range domains {
		if host == strings.ToLower(strings.TrimSpace(domain)) {
			return true
		}
	}
	return false
}

func random() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand does not fail on the platforms we run on; if it ever
		// did, an unusable value is safer than a guessable one.
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func firstOf(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "the sign-in was refused"
}
