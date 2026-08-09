package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var UnAuthorizedError = errors.New("Invalid Username or Token")

type contextKey string

const roleContextKey contextKey = "role"

func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleContextKey).(string)
	return role, ok
}

func Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		username := r.Header.Get("userid")
		token := r.Header.Get("Authorization")

		if username == "" || token == "" {
			log.Error(UnAuthorizedError)
			api.RequestErrorHandler(w, UnAuthorizedError)
			return
		}

		var database *sqltools.DatabaseInterface
		database, err = sqltools.NewDatabase()

		if err != nil {
			api.InternalErrorHandler(w)
			return
		}

		loginDetails := (*database).UserLoggedIn(username)

		if loginDetails == nil {
			log.Error(UnAuthorizedError)
			api.RequestErrorHandler(w, UnAuthorizedError)
			return
		}

		log.Printf("Login lookup succeeded for user %s", username)
		err = bcrypt.CompareHashAndPassword([]byte((*loginDetails).Code), []byte(token))
		if err != nil {
			log.Warn("Invalid login attempt for:", username)
			api.RequestErrorHandler(w, UnAuthorizedError)
			return
		}

		ctx := context.WithValue(r.Context(), roleContextKey, loginDetails.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
