package admin

import (
	"encoding/json"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	var database *sqltools.DatabaseInterface
	var err error

	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	users, errGet := (*database).GetUsers()
	if errGet != nil {
		log.Error("Failed to get users: ", errGet)
		api.InternalErrorHandler(w) // was api.DuplicateError — a read failure isn't a duplicate-record error
		return
	}

	grouped := api.GroupedUsersResponse{
		Admins:       []api.UserSummary{},
		Sales:        []api.UserSummary{},
		Distributors: []api.UserSummary{},
	}

	for _, u := range users {
		summary := api.ToUserSummary(u)
		switch u.Role {
		case "admin":
			grouped.Admins = append(grouped.Admins, summary)
		case "sales":
			grouped.Sales = append(grouped.Sales, summary)
		case "distributor":
			grouped.Distributors = append(grouped.Distributors, summary)
		default:
			log.Warnf("user %s has unrecognized role %q, omitted from grouped response", u.UserID, u.Role)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(grouped); err != nil {
		log.Error("Failed to encode response: ", err)
		api.InternalErrorHandler(w)
		return
	}
}
