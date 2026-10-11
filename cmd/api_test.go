package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestValidateAllowedOrigins(t *testing.T) {
	if err := validateAllowedOrigins([]string{"http://localhost:3000", "https://client.example"}); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"*", "null", "https://*.example", "ftp://client.example", "https://client.example/", "https://client.example/path", "https://client.example?x=1", "https://user@client.example"} {
		t.Run(origin, func(t *testing.T) {
			if err := validateAllowedOrigins([]string{origin}); err == nil || !strings.Contains(err.Error(), "exact http or https origin") {
				t.Fatalf("origin %q accepted or returned unstable error: %v", origin, err)
			}
		})
	}
}

func TestRunAPIRejectsUnsafeConfigurationBeforeStartup(t *testing.T) {
	command := &cobra.Command{}
	command.SetContext(context.Background())
	base := apiConfig{
		listen:          "127.0.0.1:8080",
		readTimeout:     time.Second,
		writeTimeout:    time.Second,
		idleTimeout:     time.Second,
		shutdownTimeout: time.Second,
	}
	for _, test := range []struct {
		name   string
		mutate func(*apiConfig)
		want   string
	}{
		{name: "invalid listen", mutate: func(c *apiConfig) { c.listen = "not-an-address" }, want: "invalid --listen address"},
		{name: "remote listener", mutate: func(c *apiConfig) { c.listen = "0.0.0.0:8080" }, want: "--listen must use a loopback address"},
		{name: "invalid origin", mutate: func(c *apiConfig) { c.origins = []string{"https://client.example/path"} }, want: "invalid --allowed-origin"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := base
			test.mutate(&config)
			err := runAPI(command, config)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}
