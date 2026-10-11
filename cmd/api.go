package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/webapi"
	"github.com/spf13/cobra"
)

func newAPICommand() *cobra.Command {
	var listen, dbPath, accountsConfig string
	var origins []string
	var readTimeout, writeTimeout, idleTimeout, shutdownTimeout time.Duration
	command := &cobra.Command{
		Use:   "api",
		Short: "Serve the versioned Sudoku HTTP API",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAPI(cmd, apiConfig{listen: listen, dbPath: dbPath, accountsConfig: accountsConfig, origins: origins, readTimeout: readTimeout, writeTimeout: writeTimeout, idleTimeout: idleTimeout, shutdownTimeout: shutdownTimeout})
		},
	}
	flags := command.Flags()
	flags.StringVar(&listen, "listen", "127.0.0.1:8080", "HTTP listen address")
	flags.StringVar(&dbPath, "db", "", "Puzzle database path (defaults to the XDG data directory)")
	flags.StringVar(&accountsConfig, "accounts-config", "", "private JSON configuration enabling guest and account routes")
	flags.StringSliceVar(&origins, "allowed-origin", nil, "allowed browser origin (repeatable)")
	flags.DurationVar(&readTimeout, "read-timeout", 15*time.Second, "maximum request read time")
	flags.DurationVar(&writeTimeout, "write-timeout", 30*time.Second, "maximum response write time")
	flags.DurationVar(&idleTimeout, "idle-timeout", 60*time.Second, "keep-alive idle timeout")
	flags.DurationVar(&shutdownTimeout, "shutdown-timeout", 10*time.Second, "graceful shutdown timeout")
	return command
}

type apiConfig struct {
	listen, dbPath, accountsConfig                          string
	origins                                                 []string
	readTimeout, writeTimeout, idleTimeout, shutdownTimeout time.Duration
}

func runAPI(command *cobra.Command, config apiConfig) error {
	host, _, err := net.SplitHostPort(config.listen)
	if err != nil {
		return fmt.Errorf("invalid --listen address: %w", err)
	}
	ip := net.ParseIP(host)
	loopback := strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
	if !loopback {
		return errors.New("--listen must use a loopback address")
	}
	if err := validateAllowedOrigins(config.origins); err != nil {
		return err
	}

	options := game.NewDefaultOptions(solverStore)
	options.StrategySolverKeys = solverStore.GetAllStrategySolverKeys()
	server := webapi.NewServer()
	accounts, closeAccounts, err := loadAccountRuntime(command.Context(), config.accountsConfig, config.dbPath, options)
	if err != nil {
		return err
	}
	defer func() { _ = closeAccounts() }()
	server.SetAccountRuntime(accounts)

	httpServer := &http.Server{Addr: config.listen, Handler: webapi.NewHandler(server, config.origins), ReadTimeout: config.readTimeout, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: config.writeTimeout, IdleTimeout: config.idleTimeout, MaxHeaderBytes: 32 << 10}
	listener, err := net.Listen("tcp", config.listen)
	if err != nil {
		return err
	}
	fmt.Fprintf(command.ErrOrStderr(), "Sudoku API listening on http://%s\n", listener.Addr())

	serveErrors := make(chan error, 1)
	go func() { serveErrors <- httpServer.Serve(listener) }()
	ctx, stop := signal.NotifyContext(command.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), config.shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down API server: %w", err)
		}
		return nil
	}
}

func validateAllowedOrigins(origins []string) error {
	for _, origin := range origins {
		parsed, err := url.Parse(origin)
		if err != nil || strings.Contains(origin, "*") || parsed.String() != origin || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("invalid --allowed-origin %q: expected an exact http or https origin", origin)
		}
	}
	return nil
}
