package authhandlers

import (
	"errors"
	"math"
	"time"

	"github.com/Olayori-X/stock-control-backend/functions"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// checkSalesAssociateResumption enforces the daily resumption geofence for
// Sales Associates. Non-sales roles and associates with nothing planned
// today are waved through. Returns an error only when the associate should
// be blocked from logging in (or on an internal failure) — callers surface
// that error via the standard handlers.
//
// Shared between LoginHandler (email/password) and PINLoginHandler, since
// both need identical resumption enforcement and duplicating this logic
// risks the two drifting out of sync.
func checkSalesAssociateResumption(database *sqltools.DatabaseInterface, userID, role string, lat, lon *float64, deviceRef string) error {
	if role != "sales" {
		return nil
	}

	if lat == nil || lon == nil {
		return errors.New("location is required to log in")
	}

	routeDay := time.Now().Weekday().String()

	plannedOutlets, err := (*database).GetPlannedOutletsForDay(userID, routeDay)
	if err != nil {
		log.Error("Failed to fetch planned outlets: ", err)
		return errors.New("internal error")
	}

	if len(plannedOutlets) == 0 {
		log.Infof("No planned outlets for %s on %s, skipping resumption check", userID, routeDay)
		return nil
	}

	radiusKM, err := (*database).GetSettingFloat("resumption_radius_km")
	if err != nil {
		log.Error("Failed to read resumption_radius_km setting: ", err)
		return errors.New("internal error")
	}
	radiusMeters := radiusKM * 1000

	nearestDistance := math.MaxFloat64
	for _, outlet := range plannedOutlets {
		d := functions.HaversineMeters(*lat, *lon, outlet.Latitude, outlet.Longitude)
		if d < nearestDistance {
			nearestDistance = d
		}
	}

	result := "FAIL"
	if nearestDistance <= radiusMeters {
		result = "PASS"
	}

	if err := (*database).RecordResumption(userID, routeDay, *lat, *lon, nearestDistance, result, deviceRef); err != nil {
		log.Error("Failed to record resumption: ", err)
		return errors.New("internal error")
	}

	if result == "FAIL" {
		log.Warnf("Resumption FAIL for %s: %.0fm from nearest planned outlet (radius %.0fm)",
			userID, nearestDistance, radiusMeters)
		return errors.New("you are not near your planned route for today")
	}

	return nil
}
