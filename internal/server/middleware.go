package server

import (
	"context"
	"net/http"

	"github.com/drudge/wgrift/internal/auth"
	"github.com/drudge/wgrift/internal/models"
)

type contextKey string

const (
	ctxUser    contextKey = "user"
	ctxSession contextKey = "session"
)

func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(ctxUser).(*models.User)
	return u
}

func SessionFromContext(ctx context.Context) *models.Session {
	s, _ := ctx.Value(ctxSession).(*models.Session)
	return s
}

func authRequired(authSvc *auth.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				writeError(w, http.StatusForbidden, "authentication required")
				return
			}

			session, user, err := authSvc.ValidateSession(cookie.Value)
			if err != nil {
				writeError(w, http.StatusForbidden, "invalid session")
				return
			}

			ctx := context.WithValue(r.Context(), ctxUser, user)
			ctx = context.WithValue(ctx, ctxSession, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		session := SessionFromContext(r.Context())
		if session == nil {
			writeError(w, http.StatusForbidden, "no session")
			return
		}

		token := r.Header.Get("X-CSRF-Token")
		if token == "" || token != session.CSRFToken {
			writeError(w, http.StatusForbidden, "invalid CSRF token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil || user.Role != "admin" {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
