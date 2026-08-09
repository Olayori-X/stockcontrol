package middleware

import (
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	log "github.com/sirupsen/logrus"
)

var ForbiddenError = errors.New("you do not have access to this resource")

func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok || role == "" {
				log.Error("RequireRole: no role found in request context")
				api.InternalErrorHandler(w)
				return
			}

			for _, allowed := range allowedRoles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			log.Warnf("user with role %q denied access (needs one of %v)", role, allowedRoles)
			api.RequestErrorHandler(w, ForbiddenError)
		})
	}
}
