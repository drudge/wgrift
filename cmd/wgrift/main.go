package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drudge/wgrift/internal/auth"
	"github.com/drudge/wgrift/internal/config"
	"github.com/drudge/wgrift/internal/crypto"
	"github.com/drudge/wgrift/internal/server"
	"github.com/drudge/wgrift/internal/store"
	"github.com/drudge/wgrift/internal/wg"
	"github.com/spf13/cobra"
)

func main() {
	var cfgPath string

	root := &cobra.Command{
		Use:   "wgrift",
		Short: "WireGuard VPN management",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the wgRift server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cfgPath)
		},
	}
	serveCmd.Flags().StringVarP(&cfgPath, "config", "c", "", "config file path")

	root.AddCommand(serveCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServe(cfgPath string) error {
	// Load config
	cfg := config.Defaults()
	if cfgPath != "" {
		var err error
		cfg, err = config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}

	// Allow listen address override via env
	if listen := os.Getenv("WGRIFT_LISTEN"); listen != "" {
		cfg.Server.Listen = listen
	}

	// Master key
	masterKey, err := cfg.MasterKey()
	if err != nil {
		return fmt.Errorf("master key: %w", err)
	}

	enc := crypto.NewEncryptor(masterKey)

	// Database
	db, err := store.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}

	// Auth service
	sessionTimeout, _ := time.ParseDuration(cfg.Auth.SessionTimeout)
	if sessionTimeout == 0 {
		sessionTimeout = 30 * time.Minute
	}
	maxSessionAge, _ := time.ParseDuration(cfg.Auth.MaxSessionAge)
	if maxSessionAge == 0 {
		maxSessionAge = 24 * time.Hour
	}
	minPwLen := cfg.Auth.Local.MinPasswordLength
	if minPwLen == 0 {
		minPwLen = 16
	}
	authSvc := auth.NewService(db, sessionTimeout, maxSessionAge, minPwLen)

	// WireGuard manager
	nm := wg.NewNetManager()
	mgr, err := wg.NewManager(db, enc, nm, "")
	if err != nil {
		return fmt.Errorf("creating wg manager: %w", err)
	}
	defer mgr.Close()

	// Server
	srv := server.New(cfg, authSvc, mgr, db, enc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
	}()

	return srv.Start(ctx)
}
