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
	if cfg.WeeklyMaintenanceStartTime != "" && cfg.WeeklyMaintenanceEndTime != "" {
		// If the weekday is maintenance day
		if currentTime.Weekday() == cfg.WeeklyMaintenanceDay {

			weeklyMaintenanceStartTime := returnWeeklyMaintenanceTime(cfg.WeeklyMaintenanceStartTime[:2], cfg.WeeklyMaintenanceStartTime[2:])

			weeklyMaintenanceEndTime := returnWeeklyMaintenanceTime(cfg.WeeklyMaintenanceEndTime[:2], cfg.WeeklyMaintenanceEndTime[2:])

			// Check if time is within maintenance time
			if weeklyMaintenanceEndTime.After(currentTime) && weeklyMaintenanceStartTime.Before(currentTime) {
				systemAvailableTime = weeklyMaintenanceEndTime
				systemUnavailable = true
			}
		}
	}

	// Check for planned maintenance
	if cfg.PlannedMaintenanceStart != "" && cfg.PlannedMaintenanceEnd != "" {
		timeDateLayout := "02 Jan 06 15:04 MST"
		maintenanceStart, err := time.Parse(timeDateLayout, cfg.PlannedMaintenanceStart)
		if err != nil {
			log.ErrorR(r, fmt.Errorf("error parsing Maintenance Start time: [%v]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		maintenanceEnd, err := time.Parse(timeDateLayout, cfg.PlannedMaintenanceEnd)
		if err != nil {
			log.ErrorR(r, fmt.Errorf("error parsing Maintenance End time: [%v]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if maintenanceEnd.After(currentTime) && maintenanceStart.Before(currentTime) && maintenanceEnd.After(systemAvailableTime) {
			systemAvailableTime = maintenanceEnd
			systemUnavailable = true
		}
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

// returnWeeklyMaintenanceTime returns a time.Time format for the current date with the hour and minute set to the arguments passed
func returnWeeklyMaintenanceTime(hour, minute string) time.Time {
	currentTime := time.Now()

	intHour, _ := strconv.Atoi(hour)
	timeDifferenceInHours := time.Duration(intHour - currentTime.Hour())

	intMinute, _ := strconv.Atoi(minute)
	timeDifferenceInMinutes := time.Duration(intMinute - currentTime.Minute())

	return currentTime.Add(time.Hour*timeDifferenceInHours + time.Minute*timeDifferenceInMinutes).Round(time.Second)
}
