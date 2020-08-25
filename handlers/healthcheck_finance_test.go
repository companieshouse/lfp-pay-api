package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/lfp-pay-api/config"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleHealthCheckFinance(t *testing.T) {

	cfg, _ := config.Get()
	now := time.Now()

	Convey("Given I make a request to the healthcheck_finance endpoint", t, func() {

		Convey("When the system is healthy", func() {
			cfg.WeeklyMaintenanceStartTime = fmt.Sprintf("%02d00", now.Hour())
			cfg.WeeklyMaintenanceEndTime = fmt.Sprintf("%02d00", now.Hour()-100)
			cfg.WeeklyMaintenanceDay = now.Weekday()

			req, _ := http.NewRequest("GET", "/healthcheck/finance-system", nil)
			w := httptest.NewRecorder()
			HandleHealthCheckFinanceSystem(w, req)

			Convey("Then the status should return 'OK'", func() {

				So(w.Code, ShouldEqual, http.StatusOK)

				Convey(fmt.Sprintf("And the body of the message should be correct"), func() {

					So(w.Body.String(), ShouldStartWith, `{"message":"HEALTHY"`)
				})
			})
		})

		Convey("When the system is not healthy due to weekly downtime", func() {
			cfg.WeeklyMaintenanceStartTime = fmt.Sprintf("%02d00", now.Hour())
			cfg.WeeklyMaintenanceEndTime = fmt.Sprintf("%02d00", now.Hour()+100)
			cfg.WeeklyMaintenanceDay = now.Weekday()

			req, _ := http.NewRequest("GET", "/healthcheck/finance-system", nil)
			w := httptest.NewRecorder()
			HandleHealthCheckFinanceSystem(w, req)

			Convey("Then the status should return 'Service Unavailable'", func() {

				So(w.Code, ShouldEqual, http.StatusServiceUnavailable)

				Convey("And the body of the message should be correct", func() {

					So(w.Body.String(), ShouldStartWith, `{"message":"UNHEALTHY - PLANNED MAINTENANCE","maintenance_end_time":`)
				})
			})
		})

		Convey("When the system is not healthy due to planned maintenance", func() {
			cfg.WeeklyMaintenanceStartTime = fmt.Sprintf("%02d00", now.Hour())
			cfg.WeeklyMaintenanceEndTime = fmt.Sprintf("%02d00", now.Hour()-100)
			cfg.WeeklyMaintenanceDay = now.Weekday()
			cfg.PlannedMaintenanceStart = (now.AddDate(0, 0, -1)).Format("02 Jan 06 15:04 MST")
			cfg.PlannedMaintenanceEnd = (now.AddDate(0, 0, 1)).Format("02 Jan 06 15:04 MST")

			req, _ := http.NewRequest("GET", "/healthcheck/finance-system", nil)
			w := httptest.NewRecorder()
			HandleHealthCheckFinanceSystem(w, req)

			Convey("Then the status should return 'Service Unavailable'", func() {

				So(w.Code, ShouldEqual, http.StatusServiceUnavailable)

				Convey(fmt.Sprintf("And the body of the message should be correct"), func() {

					So(w.Body.String(), ShouldStartWith, `{"message":"UNHEALTHY - PLANNED MAINTENANCE","maintenance_end_time":`)
				})
			})
		})

		Convey("When the Planned Maintenance Start Config value is invalid", func() {
			cfg.WeeklyMaintenanceStartTime = fmt.Sprintf("%02d00", now.Hour())
			cfg.WeeklyMaintenanceEndTime = fmt.Sprintf("%02d00", now.Hour()-100)
			cfg.WeeklyMaintenanceDay = now.Weekday()
			cfg.PlannedMaintenanceStart = "invalid"

			req, _ := http.NewRequest("GET", "/healthcheck/finance-system", nil)
			w := httptest.NewRecorder()
			HandleHealthCheckFinanceSystem(w, req)

			Convey("Then the status should be 'Internal Server Error'", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)

			})
		})

		Convey("When the Planned Maintenance End Config value is invalid", func() {
			cfg.WeeklyMaintenanceStartTime = fmt.Sprintf("%02d00", now.Hour())
			cfg.WeeklyMaintenanceEndTime = fmt.Sprintf("%02d00", now.Hour()-100)
			cfg.WeeklyMaintenanceDay = now.Weekday()
			cfg.PlannedMaintenanceStart = (now).Format("02 Jan 06 15:04 MST")
			cfg.PlannedMaintenanceEnd = "invalid"

			req, _ := http.NewRequest("GET", "/healthcheck/finance-system", nil)
			w := httptest.NewRecorder()
			HandleHealthCheckFinanceSystem(w, req)

			Convey("Then the status should be 'Internal Server Error'", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)

			})
		})
	})
}
