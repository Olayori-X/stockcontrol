package authhandlers

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	"github.com/Olayori-X/stock-control-backend/functions"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var params = api.LoginParams{}
	var err error

	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if strings.TrimSpace(params.Email) == "" {
		api.RequestErrorHandler(w, errors.New("email or username cannot be empty"))
		return
	}
	if strings.TrimSpace(params.Password) == "" {
		api.RequestErrorHandler(w, errors.New("password cannot be empty"))
		return
	}

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	var tokenDetails *sqltools.LoginDetails = (*database).GetUserLoginDetails(params.Email)
	if tokenDetails == nil {
		log.Warn("Login attempt with non-existing user:", params.Email)
		api.UnAuthorizedError(w)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(tokenDetails.Password), []byte(params.Password))
	if err != nil {
		log.Warn("Invalid login attempt for:", params.Email)
		api.UnAuthorizedError(w)
		return
	}

	// ── Sales associate resumption geofence ────────────────────────────────
	// A sales associate must be near at least one of today's planned outlets
	// to be allowed to log in. Other roles skip this entirely.
	if tokenDetails.Role == "sales" {
		if params.Latitude == nil || params.Longitude == nil {
			api.RequestErrorHandler(w, errors.New("location is required to log in"))
			return
		}

		routeDay := time.Now().Weekday().String() // "Monday", "Tuesday", ...

		plannedOutlets, err := (*database).GetPlannedOutletsForDay(tokenDetails.UserID, routeDay)
		if err != nil {
			log.Error("Failed to fetch planned outlets: ", err)
			api.InternalErrorHandler(w)
			return
		}

		if len(plannedOutlets) == 0 {
			// No route assigned for today (e.g. Sunday, or nothing planned yet).
			// Nothing to be "close to" — log in without a resumption check.
			log.Infof("No planned outlets for %s on %s, skipping resumption check", tokenDetails.UserID, routeDay)
		} else {
			radiusKM, err := (*database).GetSettingFloat("resumption_radius_km")
			if err != nil {
				log.Error("Failed to read resumption_radius_km setting: ", err)
				api.InternalErrorHandler(w)
				return
			}
			radiusMeters := radiusKM * 1000

			// Nearest of today's planned outlets. This approximates "distance
			// to route corridor" from the spec — true corridor/polyline
			// distance would need each outlet's route sequence and a
			// point-to-line-segment calculation, which we can add later if
			// nearest-outlet proves too permissive in practice.
			nearestDistance := math.MaxFloat64
			for _, outlet := range plannedOutlets {
				d := functions.HaversineMeters(*params.Latitude, *params.Longitude, outlet.Latitude, outlet.Longitude)
				if d < nearestDistance {
					nearestDistance = d
				}
			}

			result := "FAIL"
			if nearestDistance <= radiusMeters {
				result = "PASS"
			}

			if err := (*database).RecordResumption(
				tokenDetails.UserID, routeDay,
				*params.Latitude, *params.Longitude,
				nearestDistance, result, params.DeviceRef,
			); err != nil {
				log.Error("Failed to record resumption: ", err)
				api.InternalErrorHandler(w)
				return
			}

			if result == "FAIL" {
				log.Warnf("Resumption FAIL for %s: %.0fm from nearest planned outlet (radius %.0fm)",
					tokenDetails.UserID, nearestDistance, radiusMeters)
				api.RequestErrorHandler(w, errors.New("you are not near your planned route for today"))
				return
			}
		}
	}

	// ── Generate session token ─────────────────────────────────────────────
	code, err := functions.GenerateAuthorizationCode()
	if err != nil {
		log.Error("Generating authorization code failed", err)
		api.InternalErrorHandler(w)
		return
	}

	token, err := functions.HashString(code)
	if err != nil {
		log.Error("Hashing authorization code failed", err)
		api.InternalErrorHandler(w)
		return
	}

	err = (*database).UpsertLoggedInUser(tokenDetails.UserID, token, tokenDetails.Role)
	if err != nil {
		log.Error("An error occurred with authorization table", err)
		api.InternalErrorHandler(w)
		return
	}

	// ── Unverified user: send a fresh OTP and update code in DB ───────────
	if !tokenDetails.Verified {
		otp, err := functions.GenerateOTPCode(6)
		if err != nil {
			log.Error("Failed to generate OTP: ", err)
			api.InternalErrorHandler(w)
			return
		}

		hashedOtp, err := functions.HashString(otp)
		if err != nil {
			log.Error("Failed to hash OTP: ", err)
			api.InternalErrorHandler(w)
			return
		}

		errChan := make(chan error, 2)

		go func() {
			_, sendErr := functions.SendSimpleMessage(params.Email, "OTP code", otp)
			if sendErr != nil {
				log.Error("Failed to send OTP email: ", sendErr)
				errChan <- errors.New("failed to send verification email")
				return
			}
			errChan <- nil
		}()

		go func() {
			if updateErr := (*database).UpdateUserCode(tokenDetails.UserID, hashedOtp); updateErr != nil {
				log.Error("Failed to update OTP in DB: ", updateErr)
				errChan <- errors.New("failed to update verification code")
				return
			}
			errChan <- nil
		}()

		for i := 0; i < 2; i++ {
			if err := <-errChan; err != nil {
				api.RequestErrorHandler(w, err)
				return
			}
		}

		log.Infof("Fresh OTP sent to unverified user: %s", params.Email)
	}

	// ── Respond — same shape regardless of verified status ────────────────
	var response = api.LoginResponse{
		Code:     http.StatusOK,
		Message:  "Login successful",
		UserID:   tokenDetails.UserID,
		Role:     tokenDetails.Role,
		Verified: tokenDetails.Verified,
		Token:    code,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response: ", err)
		api.InternalErrorHandler(w)
		return
	}
}
