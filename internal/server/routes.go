package server

import "net/http"

func (s *Server) registerRoutes() {
	// Auth endpoints (no auth middleware)
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/v1/auth/session", s.handleGetSession)
	s.mux.HandleFunc("POST /api/v1/setup", s.handleSetup)

	// Protected API routes
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/dashboard", s.handleDashboard)

	// Interfaces
	protected.HandleFunc("GET /api/v1/interfaces", s.handleListInterfaces)
	protected.HandleFunc("POST /api/v1/interfaces", s.handleCreateInterface)
	protected.HandleFunc("POST /api/v1/interfaces/import", s.handleImportInterface)
	protected.HandleFunc("POST /api/v1/interfaces/adopt", s.handleAdoptInterface)
	protected.HandleFunc("GET /api/v1/interfaces/{id}", s.handleGetInterface)
	protected.HandleFunc("PUT /api/v1/interfaces/{id}", s.handleUpdateInterface)
	protected.HandleFunc("DELETE /api/v1/interfaces/{id}", s.handleDeleteInterface)
	protected.HandleFunc("POST /api/v1/interfaces/{id}/sync", s.handleSyncInterface)
	protected.HandleFunc("POST /api/v1/interfaces/{id}/start", s.handleStartInterface)
	protected.HandleFunc("POST /api/v1/interfaces/{id}/stop", s.handleStopInterface)
	protected.HandleFunc("POST /api/v1/interfaces/{id}/restart", s.handleRestartInterface)
	protected.HandleFunc("GET /api/v1/interfaces/{id}/status", s.handleInterfaceStatus)

	// Peers
	protected.HandleFunc("GET /api/v1/interfaces/{ifaceId}/peers", s.handleListPeers)
	protected.HandleFunc("POST /api/v1/interfaces/{ifaceId}/peers", s.handleAddPeer)
	protected.HandleFunc("PUT /api/v1/interfaces/{ifaceId}/peers/{id}", s.handleUpdatePeer)
	protected.HandleFunc("DELETE /api/v1/interfaces/{ifaceId}/peers/{id}", s.handleDeletePeer)
	protected.HandleFunc("POST /api/v1/interfaces/{ifaceId}/peers/{id}/enable", s.handleEnablePeer)
	protected.HandleFunc("POST /api/v1/interfaces/{ifaceId}/peers/{id}/disable", s.handleDisablePeer)
	protected.HandleFunc("PUT /api/v1/interfaces/{ifaceId}/peers/{id}/private-key", s.handleSetPeerPrivateKey)
	protected.HandleFunc("GET /api/v1/interfaces/{ifaceId}/peers/{id}/config", s.handlePeerConfig)
	protected.HandleFunc("GET /api/v1/interfaces/{ifaceId}/peers/{id}/qr", s.handlePeerQR)

	// Connection logs
	protected.HandleFunc("GET /api/v1/interfaces/{ifaceId}/logs", s.handleInterfaceLogs)
	protected.HandleFunc("GET /api/v1/peers/{id}/logs", s.handlePeerLogs)

	// Users (admin only routes are checked in handlers)
	protected.HandleFunc("GET /api/v1/users", s.handleListUsers)
	protected.HandleFunc("POST /api/v1/users", s.handleCreateUser)
	protected.HandleFunc("DELETE /api/v1/users/{id}", s.handleDeleteUser)
	protected.HandleFunc("PUT /api/v1/users/{id}/password", s.handleChangePassword)

	// Wrap protected routes with auth + CSRF middleware
	authed := authRequired(s.auth, cookieName)(csrfProtect(protected))
	s.mux.Handle("/api/v1/", authed)

	// Static file serving (SPA)
	s.mux.Handle("/", s.staticHandler())
}
