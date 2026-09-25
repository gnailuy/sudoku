package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/playrun"
	"github.com/gnailuy/sudoku/recovery"
	"github.com/gnailuy/sudoku/solver"
	"github.com/gnailuy/sudoku/webapi"
	"github.com/spf13/cobra"
)

func newAPICommand() *cobra.Command {
	var listen, token, dbPath string
	var origins []string
	var readTimeout, writeTimeout, idleTimeout, shutdownTimeout time.Duration
	command := &cobra.Command{
		Use:   "api",
		Short: "Serve the versioned Sudoku HTTP API",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAPI(cmd, apiConfig{listen: listen, token: token, dbPath: dbPath, origins: origins, readTimeout: readTimeout, writeTimeout: writeTimeout, idleTimeout: idleTimeout, shutdownTimeout: shutdownTimeout})
		},
	}
	flags := command.Flags()
	flags.StringVar(&listen, "listen", "127.0.0.1:8080", "HTTP listen address")
	flags.StringVar(&token, "auth-token", "", "bearer token required for non-loopback listeners")
	flags.StringVar(&dbPath, "db", "", "Puzzle database path (defaults to the XDG data directory)")
	flags.StringSliceVar(&origins, "allowed-origin", nil, "allowed browser origin (repeatable)")
	flags.DurationVar(&readTimeout, "read-timeout", 15*time.Second, "maximum request read time")
	flags.DurationVar(&writeTimeout, "write-timeout", 30*time.Second, "maximum response write time")
	flags.DurationVar(&idleTimeout, "idle-timeout", 60*time.Second, "keep-alive idle timeout")
	flags.DurationVar(&shutdownTimeout, "shutdown-timeout", 10*time.Second, "graceful shutdown timeout")
	return command
}

type apiConfig struct {
	listen, token, dbPath                                   string
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
	if !loopback && strings.TrimSpace(config.token) == "" {
		return errors.New("--auth-token is required for non-loopback listeners")
	}
	if strings.ContainsAny(config.token, "\r\n") {
		return errors.New("--auth-token contains invalid characters")
	}
	if err := validateAllowedOrigins(config.origins); err != nil {
		return err
	}

	base, err := recovery.DefaultDirectory()
	if err != nil {
		return err
	}
	apiDirectory := filepath.Join(base, "api")
	lock, err := acquireAPILock(apiDirectory)
	if err != nil {
		return err
	}
	defer lock.Close()

	options := game.NewDefaultOptions(solverStore)
	options.StrategySolverKeys = solverStore.GetAllStrategySolverKeys()
	registry, err := webapi.NewRegistry(recovery.NewStore(apiDirectory), options)
	if err != nil {
		return fmt.Errorf("load API recovery sessions: %w", err)
	}
	registry.SetTrackerFactory(func(current game.Game) *playrun.Tracker { return newCompletionTracker(current, config.dbPath) })
	server := webapi.NewTrackedServer(registry, func(kind, value string) (game.Game, *playrun.Tracker, webapi.SessionDifficulty, error) {
		request := sessionRequest{}
		var requested *webapi.Difficulty
		switch kind {
		case "difficulty":
			request.level = value
			d := webapi.Difficulty(value)
			requested = &d
		case "puzzle":
			request.input = value
		default:
			return game.Game{}, nil, webapi.SessionDifficulty{}, errors.New("invalid source")
		}
		request.dbPath = config.dbPath
		created, _, tracker, createErr := createTrackedSession(request, io.Discard, io.Discard)
		if createErr != nil {
			return game.Game{}, nil, webapi.SessionDifficulty{}, createErr
		}
		classification := solver.ClassifyPuzzle(solverStore, created.ProblemBoard())
		if classification.Outcome != solver.ClassificationSolved {
			return game.Game{}, nil, webapi.SessionDifficulty{}, errors.New("puzzle is not solvable by the strategy classifier")
		}
		return created, tracker, webapi.SessionDifficulty{Requested: requested, Actual: webapi.Difficulty(classification.Difficulty)}, nil
	}, func(current game.Game) (webapi.Difficulty, error) {
		classification := solver.ClassifyPuzzle(solverStore, current.ProblemBoard())
		if classification.Outcome != solver.ClassificationSolved {
			return "", errors.New("puzzle is not solvable by the strategy classifier")
		}
		return webapi.Difficulty(classification.Difficulty), nil
	})

	httpServer := &http.Server{Addr: config.listen, Handler: webapi.NewHandler(server, config.token, config.origins), ReadTimeout: config.readTimeout, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: config.writeTimeout, IdleTimeout: config.idleTimeout, MaxHeaderBytes: 32 << 10}
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

func acquireAPILock(directory string) (*os.File, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("create API state directory: %w", err)
	}
	if err := os.Chmod(directory, 0700); err != nil {
		return nil, fmt.Errorf("protect API state directory: %w", err)
	}
	path := filepath.Join(directory, ".lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open API process lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, errors.New("another sudoku api process owns the recovery namespace")
	}
	return file, nil
}
