package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/drudge/wgrift/internal/auth"
	"github.com/drudge/wgrift/internal/config"
	"github.com/drudge/wgrift/internal/crypto"
	"github.com/drudge/wgrift/internal/store"
	"github.com/drudge/wgrift/internal/wg"
)

const cookieName = "wgrift_session"

// Server is the wgRift HTTP server.
type Server struct {
	cfg     config.Config
	auth    *auth.Service
	oidc    *auth.OIDCService
	manager *wg.Manager
	store   store.Store
	enc     *crypto.Encryptor
	poller  *Poller
	mux     *http.ServeMux
	httpSrv *http.Server
}

// New creates a new Server.
func New(cfg config.Config, authSvc *auth.Service, oidcSvc *auth.OIDCService, mgr *wg.Manager, s store.Store, enc *crypto.Encryptor) *Server {
	srv := &Server{
		cfg:     cfg,
		auth:    authSvc,
		oidc:    oidcSvc,
		manager: mgr,
		store:   s,
		enc:     enc,
		mux:     http.NewServeMux(),
	}

	srv.poller = NewPoller(mgr, s, cfg)
	srv.registerRoutes()

	srv.httpSrv = &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: srv.mux,
	}

	return srv
}

// externalURL returns the external URL for this server, used for OIDC callbacks.
// Checks the settings DB first, then config, then derives from the request.
func (s *Server) externalURL(r *http.Request) string {
	// Try DB setting first
	if url, err := s.store.GetSetting("external_url"); err == nil && url != "" {
		return strings.TrimRight(url, "/")
	}
	// Fall back to config
	if s.cfg.Server.ExternalURL != "" {
		return strings.TrimRight(s.cfg.Server.ExternalURL, "/")
	}
	// Derive from request
	scheme := "https"
	if s.cfg.Server.TLS.Mode == "none" {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

// Start starts the HTTP server and connection poller.
func (s *Server) Start(ctx context.Context) error {
	// Start connection poller
	go s.poller.Run(ctx)

	// Clean expired sessions periodically
	go func() {
		s.auth.CleanExpiredSessions()
	}()

	log.Printf("wgRift server listening on %s", s.cfg.Server.Listen)

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down server...")
		return s.httpSrv.Shutdown(context.Background())
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}
}
