/*
Copyright The Helm Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package getter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"helm.sh/helm/v3/internal/test"
	"helm.sh/helm/v3/internal/version"
	"helm.sh/helm/v3/pkg/cli"
)

func TestHTTPGetter(t *testing.T) {
	g, err := NewHTTPGetter(WithURL("http://example.com"))
	if err != nil {
		t.Fatal(err)
	}

	if hg, ok := g.(*HTTPGetter); !ok {
		t.Fatal("Expected NewHTTPGetter to produce an *HTTPGetter")
	} else if hg.client != http.DefaultClient {
		t.Fatal("Expected NewHTTPGetter to return a default HTTP client.")
	}

	// Test with SSL:
	cd := "../../testdata"
	join := filepath.Join
	ca, pub, priv := join(cd, "ca.pem"), join(cd, "crt.pem"), join(cd, "key.pem")
	g, err = NewHTTPGetter(
		WithURL("http://example.com"),
		WithTLSClientConfig(pub, priv, ca),
	)
	if err != nil {
		t.Fatal(err)
	}

	hg, ok := g.(*HTTPGetter)
	if !ok {
		t.Fatal("Expected NewHTTPGetter to produce an *HTTPGetter")
	}

	transport, ok := hg.client.Transport.(*http.Transport)
	if !ok {
		t.Errorf("Expected NewHTTPGetter to set up an HTTP transport")
	}

	test.AssertGoldenString(t, transport.TLSClientConfig.ServerName, "output/httpgetter-servername.txt")

	// Test other options
	g, err = NewHTTPGetter(
		WithBasicAuth("I", "Am"),
		WithUserAgent("Groot"),
	)
	if err != nil {
		t.Fatal(err)
	}

	hg, ok = g.(*HTTPGetter)
	if !ok {
		t.Fatal("expected NewHTTPGetter to produce an *HTTPGetter")
	}

	if hg.opts.username != "I" {
		t.Errorf("Expected NewHTTPGetter to contain %q as the username, got %q", "I", hg.opts.username)
	}

	if hg.opts.password != "Am" {
		t.Errorf("Expected NewHTTPGetter to contain %q as the password, got %q", "Am", hg.opts.password)
	}

	if hg.opts.userAgent != "Groot" {
		t.Errorf("Expected NewHTTPGetter to contain %q as the user agent, got %q", "Groot", hg.opts.userAgent)
	}
}

func TestDownload(t *testing.T) {
	expect := "Call me Ishmael"
	expectedUserAgent := "I am Groot"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defaultUserAgent := "Helm/" + strings.TrimPrefix(version.GetVersion(), "v")
		if r.UserAgent() != defaultUserAgent {
			t.Errorf("Expected '%s', got '%s'", defaultUserAgent, r.UserAgent())
		}
		fmt.Fprint(w, expect)
	}))
	defer srv.Close()

	g, err := All(new(cli.EnvSettings)).ByScheme("http")
	if err != nil {
		t.Fatal(err)
	}
	got, err := g.Get(srv.URL, WithURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != expect {
		t.Errorf("Expected %q, got %q", expect, got.String())
	}

	// test with http server
	basicAuthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "username" || password != "password" {
			t.Errorf("Expected request to use basic auth and for username == 'username' and password == 'password', got '%v', '%s', '%s'", ok, username, password)
		}
		if r.UserAgent() != expectedUserAgent {
			t.Errorf("Expected '%s', got '%s'", expectedUserAgent, r.UserAgent())
		}
		fmt.Fprint(w, expect)
	}))

	defer basicAuthSrv.Close()

	u, _ := url.ParseRequestURI(basicAuthSrv.URL)
	httpgetter, err := NewHTTPGetter(
		WithURL(u.String()),
		WithBasicAuth("username", "password"),
		WithUserAgent(expectedUserAgent),
	)
	if err != nil {
		t.Fatal(err)
	}
	got, err = httpgetter.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != expect {
		t.Errorf("Expected %q, got %q", expect, got.String())
	}
}
