package auth

import (
	"context"
	"net/http"
)

type contextKey string

const (
	UserKey   contextKey = "user"
	TenantKey contextKey = "tenant"
)

// GetUser retrieves the User from context.
func GetUser(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(UserKey).(*User)
	return u, ok
}

// GetTenant retrieves the Tenant from context.
func GetTenant(ctx context.Context) (*Tenant, bool) {
	t, ok := ctx.Value(TenantKey).(*Tenant)
	return t, ok
}

// SessionMiddleware validates incoming session cookies and embeds User/Tenant into request contexts.
func SessionMiddleware(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("layerinvoice_session")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			user, tenant, err := store.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				// Expired or invalid session, clear cookie
				http.SetCookie(w, &http.Cookie{
					Name:     "layerinvoice_session",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					Secure:   r.TLS != nil,
					SameSite: http.SameSiteLaxMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			// Inject User and Tenant into request context
			ctx := context.WithValue(r.Context(), UserKey, user)
			ctx = context.WithValue(ctx, TenantKey, tenant)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth enforces authenticated sessions and issues HTMX-aware redirects.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r.Context())
		if !ok || user == nil {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole enforces specific roles (e.g. owner or employee).
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			allowed := false
			for _, role := range roles {
				if user.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
