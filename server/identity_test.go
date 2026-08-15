// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOurs(t *testing.T) {
	// The point of the allowlist is that our identity goes nowhere else, so
	// the near misses matter more than the hits.
	for host, want := range map[string]bool{
		"api.hanzo.ai":             true,
		"hanzo.ai":                 true,
		"API.HANZO.AI":             true,
		"api.hanzo.ai:443":         true,
		"api.openai.com":           false,
		"hanzo.ai.evil.com":        false,
		"nothanzo.ai":              false,
		"api.hanzo.ai.attacker.io": false,
		"":                         false,
	} {
		require.Equal(t, want, ours(host), "host %q", host)
	}
}

func TestNewIdentity(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	t.Run("absent credentials mean no identity", func(t *testing.T) {
		require.Nil(t, newIdentity(http.DefaultClient, env(map[string]string{})))
		require.Nil(t, newIdentity(http.DefaultClient, env(map[string]string{iamIDEnv: "id"})))
		require.Nil(t, newIdentity(http.DefaultClient, env(map[string]string{iamSecretEnv: "secret"})))
	})

	t.Run("the token endpoint defaults and can be overridden", func(t *testing.T) {
		i := newIdentity(http.DefaultClient, env(map[string]string{iamIDEnv: "id", iamSecretEnv: "s"}))
		require.NotNil(t, i)
		require.Equal(t, defaultIAMTokenURL, i.tokenURL)

		i = newIdentity(http.DefaultClient, env(map[string]string{
			iamIDEnv: "id", iamSecretEnv: "s", iamTokenEnv: "https://iam.test/token",
		}))
		require.Equal(t, "https://iam.test/token", i.tokenURL)
	})
}

// iam stands in for the token endpoint, counting how often it is asked.
func iam(t *testing.T, expiresIn int) (*identity, *atomic.Int32) {
	t.Helper()
	var mints atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "client_credentials", r.Form.Get("grant_type"))
		n := mints.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"token-%d","token_type":"Bearer","expires_in":%d}`, n, expiresIn)
	}))
	t.Cleanup(srv.Close)

	return &identity{client: srv.Client(), tokenURL: srv.URL, id: "id", secret: "secret"}, &mints
}

func TestToken(t *testing.T) {
	t.Run("a live token is reused", func(t *testing.T) {
		i, mints := iam(t, int((7 * 24 * time.Hour).Seconds()))

		for range 3 {
			got, err := i.Token(context.Background())
			require.NoError(t, err)
			require.Equal(t, "token-1", got)
		}
		require.EqualValues(t, 1, mints.Load(), "the endpoint was asked more than once")
	})

	t.Run("a token close to expiring is replaced", func(t *testing.T) {
		// Shorter than the renewal window, so every call is due for a new one.
		i, mints := iam(t, int(time.Hour.Seconds()))

		first, err := i.Token(context.Background())
		require.NoError(t, err)
		second, err := i.Token(context.Background())
		require.NoError(t, err)

		require.NotEqual(t, first, second)
		require.EqualValues(t, 2, mints.Load())
	})

	t.Run("a refusal surfaces when there is nothing cached", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		i := &identity{client: srv.Client(), tokenURL: srv.URL, id: "id", secret: "secret"}
		_, err := i.Token(context.Background())
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret", "the error carried the credential")
	})

	t.Run("a refusal falls back to the token already held", func(t *testing.T) {
		var fail atomic.Bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if fail.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"held","expires_in":3600}`)
		}))
		defer srv.Close()

		i := &identity{client: srv.Client(), tokenURL: srv.URL, id: "id", secret: "secret"}
		first, err := i.Token(context.Background())
		require.NoError(t, err)
		require.Equal(t, "held", first)

		// Due for renewal (1h < 24h window) but IAM is now refusing: the
		// workspace keeps working on the token it has.
		fail.Store(true)
		again, err := i.Token(context.Background())
		require.NoError(t, err)
		require.Equal(t, "held", again)
	})
}
