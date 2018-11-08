package oauthclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	oidc "github.com/coreos/go-oidc"
	"golang.org/x/oauth2"

	"github.com/sdrozdkov/kubectl-login/helpers"
	"github.com/sdrozdkov/kubectl-login/kubeconfig"
)

type App struct {
	redirectURI string // place this in env variable or kubeconfig
	kubeConfig  *kubeconfig.OIDCKubeConfig
	verifier    *oidc.IDTokenVerifier
	provider    *oidc.Provider
	state       string

	offlineAsScope bool

	client *http.Client
	server *http.Server
}

func (a *App) oauth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.kubeConfig.ClientID,
		ClientSecret: a.kubeConfig.ClientSecret,
		Endpoint:     a.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  a.redirectURI,
	}
}

func (a *App) Init(config *kubeconfig.OIDCKubeConfig, port string) {
	a.redirectURI = fmt.Sprintf("http://localhost:%s/auth/callback", port)

	a.kubeConfig = config

	if a.client == nil {
		a.client = http.DefaultClient
	}

	ctx := oidc.ClientContext(context.Background(), a.client)
	provider, err := oidc.NewProvider(ctx, a.kubeConfig.Issuer)
	if err != nil {
		log.Fatalf("Failed to query provider %q: %v", a.kubeConfig.Issuer, err)
	}

	a.provider = provider
	a.verifier = provider.Verifier(&oidc.Config{ClientID: a.kubeConfig.ClientID})

	var s struct {
		ScopesSupported []string `json:"scopes_supported"`
	}
	if err := provider.Claims(&s); err != nil {
		log.Fatalf("Failed to parse provider scopes_supported: %v", err)
	}

	http.HandleFunc("/auth/callback", a.handleCallback)
	http.HandleFunc("/auth", a.handleLogin)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	a.state = helpers.StateGenerator()

	var scopes []string
	if extraScopes := r.FormValue("extra_scopes"); extraScopes != "" {
		scopes = strings.Split(extraScopes, " ")
	}
	var clients []string
	if crossClients := r.FormValue("cross_client"); crossClients != "" {
		clients = strings.Split(crossClients, " ")
	}
	for _, client := range clients {
		scopes = append(scopes, "audience:server:client_id:"+client)
	}

	authCodeURL := ""
	scopes = append(scopes, "openid", "profile", "email", "groups", "offline_access")
	if r.FormValue("offline_access") != "yes" {
		authCodeURL = a.oauth2Config(scopes).AuthCodeURL(a.state)
	} else if a.offlineAsScope {
		scopes = append(scopes, "offline_access")
		authCodeURL = a.oauth2Config(scopes).AuthCodeURL(a.state)
	} else {
		authCodeURL = a.oauth2Config(scopes).AuthCodeURL(a.state, oauth2.AccessTypeOffline)
	}

	// Redirect to k8s ldap connector
	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func (a *App) handleCallback(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		token *oauth2.Token
	)

	ctx := oidc.ClientContext(r.Context(), a.client)
	oauth2Config := a.oauth2Config(nil)
	switch r.Method {
	case "GET":
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, errMsg+": "+r.FormValue("error_description"), http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		if code == "" {
			http.Error(w, fmt.Sprintf("no code in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		if state := r.FormValue("state"); state != a.state {
			fmt.Println(state)
			http.Error(w, fmt.Sprintf("expected state %q got %q", a.state, state), http.StatusBadRequest)
			return
		}
		token, err = oauth2Config.Exchange(ctx, code)
	case "POST":
		refresh := r.FormValue("refresh_token")
		if refresh == "" {
			http.Error(w, fmt.Sprintf("no refresh_token in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		t := &oauth2.Token{
			RefreshToken: refresh,
			Expiry:       time.Now().Add(-time.Hour),
		}
		token, err = oauth2Config.TokenSource(ctx, t).Token()
	default:
		http.Error(w, fmt.Sprintf("method not implemented: %s", r.Method), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in token response", http.StatusInternalServerError)
		return
	}

	_, err = a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to verify ID token: %v", err), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, clsTmpl, nil)

	a.kubeConfig.IDToken = rawIDToken
	a.kubeConfig.RefreshToken = token.RefreshToken

	if err := a.kubeConfig.WriteNewTokens(); err != nil {
		log.Fatalf("Can not write new config: %v", err)
	}

	go a.server.Shutdown(nil)
	return
}

func (a *App) Run(port string) {
	srv := &http.Server{Addr: fmt.Sprintf(":%s", port), Handler: nil}

	a.server = srv

	// TODO: fix silent errors
	a.server.ListenAndServe()
}
