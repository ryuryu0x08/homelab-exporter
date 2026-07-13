package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ryuryu0x08/homelab-exporter/internal/aggregate"
	"github.com/ryuryu0x08/homelab-exporter/internal/app"
	"github.com/ryuryu0x08/homelab-exporter/internal/config"
	"github.com/ryuryu0x08/homelab-exporter/internal/platform"
	"github.com/ryuryu0x08/homelab-exporter/internal/platform/windows"
	"github.com/ryuryu0x08/homelab-exporter/internal/server"
)

const (
	commandDomain        = "services"
	commandResource      = "exporter"
	commandAction        = "serve"
	defaultConfigPath    = "config.toml"
	readHeaderTimeout    = 5 * time.Second
	idleTimeout          = 30 * time.Second
	shutdownTimeout      = 10 * time.Second
	minimumCommandTokens = 3
)

func main() {
	logger := log.New(os.Stderr, "homelab-exporter ", log.LstdFlags|log.LUTC)
	err := run(os.Args[1:], logger)
	if err != nil {
		logger.Printf("main.run failed: %v", err)
		os.Exit(1)
	}
}

func run(arguments []string, logger *log.Logger) error {
	configPath, err := parseCommand(arguments)
	if err != nil {
		return err
	}
	runtimeConfig, err := config.Load(configPath)
	if err != nil {
		return err
	}
	err = platform.Validate(runtimeConfig.Platform, runtime.GOOS)
	if err != nil {
		return err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	client := &http.Client{Transport: transport}
	httpScraper := aggregate.NewHTTPScraper(client, runtimeConfig.MaxBodyBytes)
	nvidiaSMIScraper := windows.NewNVIDIASMIScraper()
	scraper := aggregate.NewRoutedScraper(httpScraper, nvidiaSMIScraper)
	aggregator := aggregate.New(scraper, logger)
	service := app.NewService(aggregator, runtimeConfig.Sources, runtimeConfig.ScrapeTimeout)
	httpServer := &http.Server{
		Addr:              runtimeConfig.Listen,
		Handler:           server.New(service, logger),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
	return serve(httpServer, client, logger)
}

func parseCommand(arguments []string) (string, error) {
	if len(arguments) < minimumCommandTokens {
		return "", errors.New("usage: homelab-exporter services exporter serve --config <path>")
	}
	if arguments[0] != commandDomain || arguments[1] != commandResource || arguments[2] != commandAction {
		return "", errors.New("command must be: services exporter serve")
	}
	flags := flag.NewFlagSet(commandAction, flag.ContinueOnError)
	configPath := flags.String("config", defaultConfigPath, "TOML configuration path")
	err := flags.Parse(arguments[minimumCommandTokens:])
	if err != nil {
		return "", fmt.Errorf("parse serve flags: %w", err)
	}
	if flags.NArg() != 0 {
		return "", fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	return *configPath, nil
}

func serve(httpServer *http.Server, client *http.Client, logger *log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errorChannel := make(chan error, 1)
	go func() {
		logger.Printf("serve listening on %s", httpServer.Addr)
		err := httpServer.ListenAndServe()
		errorChannel <- err
	}()

	select {
	case err := <-errorChannel:
		client.CloseIdleConnections()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err := httpServer.Shutdown(shutdownContext)
	client.CloseIdleConnections()
	if err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}
