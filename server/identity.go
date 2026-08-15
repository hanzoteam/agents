// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-agents/v2/bots"
	"github.com/mattermost/mattermost-plugin-agents/v2/config"
)

// This workspace reaches Hanzo services as itself. It is an application
// registered in Hanzo IAM, and a client-credentials grant returns a bearer
// whose claims carry the org and the billing account the usage is metered to.
// There is no API key anywhere in this path: a service configured without a key
// of its own is one of ours, and is reached with that bearer.
//
// The credentials are the pair the server already holds for signing users in.
// It is one application and one identity, so there is nothing further to seal,
// rotate, or keep in step.
const (
	iamIDEnv     = "MM_HANZOSETTINGS_ID"
	iamSecretEnv = "MM_HANZOSETTINGS_SECRET" // #nosec G101 -- the name of a variable, not a credential

	// Where Hanzo IAM issues tokens. Fixed, not configurable: it is one address
	// for the whole fleet, and a setting for it is a setting that can disagree
	// with the credentials sitting next to it.
	iamTokenURL = "https://hanzo.id/v1/iam/oauth/token"

	// Tokens are minted well before they lapse, because the cost of an early
	// mint is one request and the cost of a late one is every agent in the
	// workspace answering with an authentication error.
	iamRenewBefore = 24 * time.Hour
)

// identity holds the workspace's own credentials and the access token minted
// from them. The token is cached until shortly before it expires.
type identity struct {
	client   *http.Client
	tokenURL string
	id       string
	secret   string

	mu      sync.Mutex
	token   string
	expires time.Time
}

// newIdentity reads the workspace's credentials from the environment, which is
// where KMS delivers them. It returns nil when they are absent, which is the
// ordinary case for a server that is not part of a Hanzo installation.
func newIdentity(client *http.Client, env func(string) string) *identity {
	id, secret := env(iamIDEnv), env(iamSecretEnv)
	if id == "" || secret == "" {
		return nil
	}

	return &identity{client: client, tokenURL: iamTokenURL, id: id, secret: secret}
}

// Token returns a valid access token, minting a new one when the cached token
// is missing or close to expiring.
func (i *identity) Token(ctx context.Context) (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.token != "" && time.Until(i.expires) > iamRenewBefore {
		return i.token, nil
	}

	token, lifetime, err := i.mint(ctx)
	if err != nil {
		// The old token is kept: it may still have hours left on it, and a
		// stale token that works beats no token because IAM was briefly
		// unreachable.
		if i.token != "" {
			return i.token, nil
		}
		return "", err
	}

	i.token, i.expires = token, time.Now().Add(lifetime)
	return i.token, nil
}

func (i *identity) mint(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {i.id},
		"client_secret": {i.secret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := i.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to reach %s: %w", i.tokenURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Deliberately without the body: a failed token response can echo the
		// request, and the request carries the client secret.
		return "", 0, fmt.Errorf("%s answered %d", i.tokenURL, resp.StatusCode)
	}

	var grant struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&grant); err != nil {
		return "", 0, fmt.Errorf("failed to read token response: %w", err)
	}
	if grant.AccessToken == "" {
		return "", 0, fmt.Errorf("%s returned no access token", i.tokenURL)
	}

	lifetime := time.Duration(grant.ExpiresIn) * time.Second
	if lifetime <= 0 {
		lifetime = 2 * iamRenewBefore
	}
	return grant.AccessToken, lifetime, nil
}

// ours reports whether a host is a Hanzo endpoint, and so may be sent this
// workspace's token.
//
// An allowlist rather than "any service without a key", because a service
// missing its key is indistinguishable from one that was never given one, and
// the failure mode of guessing wrong is handing our identity to somebody
// else's endpoint.
func ours(host string) bool {
	host = strings.ToLower(host)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host == "hanzo.ai" || strings.HasSuffix(host, ".hanzo.ai")
}

// authenticate gives every Hanzo service that carries no key of its own the
// workspace's current access token, and rebuilds the agents so they pick it up.
//
// The token is written to the copy this process holds, never to the stored
// configuration: the admin API reads the database, so the settings page keeps
// showing the empty key it was given, and saving it writes that empty key back.
func (p *Plugin) authenticate(ctx context.Context, ident *identity, botsService *bots.MMBots) error {
	live := p.configuration.Config()
	if live == nil {
		return nil
	}
	// A copy: Config() hands back the pointer this process is serving requests
	// from, and writing a token into it would be a write to shared state.
	cfg, err := config.DeepCopyJSON(*live)
	if err != nil {
		return fmt.Errorf("failed to copy configuration: %w", err)
	}

	var wanted bool
	for _, svc := range cfg.Services {
		if svc.APIKey != "" {
			continue
		}
		if u, err := url.Parse(svc.APIURL); err == nil && ours(u.Host) {
			wanted = true
			break
		}
	}
	if !wanted {
		return nil
	}

	token, tokenErr := ident.Token(ctx)
	if tokenErr != nil {
		return tokenErr
	}

	for i := range cfg.Services {
		if cfg.Services[i].APIKey != "" {
			continue
		}
		if u, err := url.Parse(cfg.Services[i].APIURL); err == nil && ours(u.Host) {
			cfg.Services[i].APIKey = token
		}
	}

	// Without notifying: the listeners rebuild bots, and this is called from
	// activation where that is already about to happen.
	if err := p.configuration.StorePersistedConfigWithoutNotify(&cfg); err != nil {
		return fmt.Errorf("failed to store configuration: %w", err)
	}

	botsService.ForceRefreshOnNextEnsure()
	return botsService.EnsureBots()
}

// renew keeps the token current for as long as the plugin is running. A token
// outlives this interval several times over, so a missed round is not an
// outage.
func (p *Plugin) renew(ctx context.Context, ident *identity, botsService *bots.MMBots) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.authenticate(ctx, ident, botsService); err != nil {
				p.pluginAPI.Log.Error("Failed to renew the workspace access token", "error", err.Error())
			}
		}
	}
}

// startRenew runs the renewal loop and returns the cancel that stops it.
func (p *Plugin) startRenew(ident *identity, botsService *bots.MMBots) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go p.renew(ctx, ident, botsService)
	return cancel
}
