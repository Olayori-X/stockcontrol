package generalhandlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("SearchUsersHandler called")
	query := r.URL.Query().Get("q")
	if query == "" {
		api.RequestErrorHandler(w, errors.New("search query cannot be empty"))
		return
	}

	// Exclude the authenticated user from results
	callerID := r.Header.Get("userid")

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	users, err := (*database).SearchUsers(query, callerID)
	if err != nil {
		log.Error("Error searching users: ", err)
		api.InternalErrorHandler(w)
		return
	}

	// Convert to safe public shape — never expose email/phone/password
	results := make([]api.UserSearchResult, 0, len(users))
	for _, u := range users {
		results = append(results, api.ToSearchResult(u))
	}

	log.Printf("Fetched users: %v", results)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.SearchUsersResponse{
		Users: results,
		Total: len(results),
	}); err != nil {
		log.Error("Failed to encode response: ", err)
		api.InternalErrorHandler(w)
		return
	}
}
