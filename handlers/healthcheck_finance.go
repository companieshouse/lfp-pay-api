package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/utils"
)

// HandleHealthCheckFinanceSystem checks whether the e5 system is available to take requests
func HandleHealthCheckFinanceSystem(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Get()
	if err != nil {
		log.ErrorR(r, fmt.Errorf("error returning config: [%v]", err))
		m := models.NewMessageResponse("failed to get maintenance times from config")
		utils.WriteJSONWithStatus(w, r, m, http.StatusInternalServerError)
		return
	}

	currentTime := time.Now()
	var systemUnavailable bool
	var systemAvailableTime time.Time

	// Check for weekly downtime
	systemAvailableTime, systemUnavailable = checkWeeklyDownTime(cfg,
		currentTime, systemAvailableTime, systemUnavailable)

	// Check for planned maintenance
	systemAvailableTime, systemUnavailable, parseError := checkPlannedMaintenance(w, r, cfg, currentTime, systemAvailableTime, systemUnavailable)
	if parseError {
		return
	}

	if systemUnavailable {
		m := models.NewMessageTimeResponse("UNHEALTHY - PLANNED MAINTENANCE", systemAvailableTime)
		utils.WriteJSONWithStatus(w, r, m, http.StatusServiceUnavailable)
		log.TraceR(r, "Planned maintenance")
		return
	}

	m := models.NewMessageResponse("HEALTHY")
	utils.WriteJSON(w, r, m)
}

func checkPlannedMaintenance(w http.ResponseWriter,
	r *http.Request,
	cfg *config.Config,
	currentTime time.Time,
	systemAvailableTime time.Time,
	systemUnavailable bool) (time.Time, bool, bool) {
	if isPlannedMaintenanceCheckRequired(cfg) {
		timeDateLayout := "02 Jan 06 15:04 MST"
		maintenanceStart, err := time.Parse(timeDateLayout, cfg.PlannedMaintenanceStart)
		if err != nil {
			log.ErrorR(r, fmt.Errorf("error parsing Maintenance Start time: [%v]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return time.Time{}, false, true
		}
		maintenanceEnd, err := time.Parse(timeDateLayout, cfg.PlannedMaintenanceEnd)
		if err != nil {
			log.ErrorR(r, fmt.Errorf("error parsing Maintenance End time: [%v]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return time.Time{}, false, true
		}

		if maintenanceEnd.After(currentTime) && maintenanceStart.Before(currentTime) && maintenanceEnd.After(systemAvailableTime) {
			systemAvailableTime = maintenanceEnd
			systemUnavailable = true
		}
	}
	return systemAvailableTime, systemUnavailable, false
}

func checkWeeklyDownTime(cfg *config.Config,
	currentTime time.Time,
	systemAvailableTime time.Time,
	systemUnavailable bool) (time.Time, bool) {
	if isWeeklyMaintenanceTimeCheckRequired(cfg) {
		// If the weekday is maintenance day
		if currentTime.Weekday() == cfg.WeeklyMaintenanceDay {

			weeklyMaintenanceStartTime := returnWeeklyMaintenanceTime(cfg.WeeklyMaintenanceStartTime[:2], cfg.WeeklyMaintenanceStartTime[2:])

			weeklyMaintenanceEndTime := returnWeeklyMaintenanceTime(cfg.WeeklyMaintenanceEndTime[:2], cfg.WeeklyMaintenanceEndTime[2:])

			// Check if time is within maintenance time
			if isWithinMaintenanceTime(weeklyMaintenanceEndTime, currentTime, weeklyMaintenanceStartTime) {
				systemAvailableTime = weeklyMaintenanceEndTime
				systemUnavailable = true
			}
		}
	}
	return systemAvailableTime, systemUnavailable
}

func isPlannedMaintenanceCheckRequired(cfg *config.Config) bool {
	return cfg.PlannedMaintenanceStart != "" && cfg.PlannedMaintenanceEnd != ""
}

func isWithinMaintenanceTime(weeklyMaintenanceEndTime time.Time, currentTime time.Time, weeklyMaintenanceStartTime time.Time) bool {
	return weeklyMaintenanceEndTime.After(currentTime) && weeklyMaintenanceStartTime.Before(currentTime)
}

func isWeeklyMaintenanceTimeCheckRequired(cfg *config.Config) bool {
	return cfg.WeeklyMaintenanceStartTime != "" && cfg.WeeklyMaintenanceEndTime != ""
}

// returnWeeklyMaintenanceTime returns a time.Time format for the current date with the hour and minute set to the arguments passed
func returnWeeklyMaintenanceTime(hour, minute string) time.Time {
	currentTime := time.Now()

	intHour, _ := strconv.Atoi(hour)
	timeDifferenceInHours := time.Duration(intHour - currentTime.Hour())

	intMinute, _ := strconv.Atoi(minute)
	timeDifferenceInMinutes := time.Duration(intMinute - currentTime.Minute())

	return currentTime.Add(time.Hour*timeDifferenceInHours + time.Minute*timeDifferenceInMinutes).Round(time.Second)
}
