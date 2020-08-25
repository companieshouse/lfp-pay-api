package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitRegisterRoutes(t *testing.T) {
	Convey("Register routes", t, func() {
		router := mux.NewRouter()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockService := mocks.NewMockService(mockCtrl)
		Register(router, &config.Config{}, mockService)

		So(router.GetRoute("healthcheck"), ShouldNotBeNil)
		So(router.GetRoute("healthcheck-finance-system"), ShouldNotBeNil)
		So(router.GetRoute("get-penalties"), ShouldNotBeNil)
		So(router.GetRoute("create-payable"), ShouldNotBeNil)
		So(router.GetRoute("get-payable"), ShouldNotBeNil)
		So(router.GetRoute("get-payment-details"), ShouldNotBeNil)
		So(router.GetRoute("mark-as-paid"), ShouldNotBeNil)
	})
}

func TestUnitGetHealthCheck(t *testing.T) {
	Convey("Get HealthCheck", t, func() {
		req := httptest.NewRequest("GET", "/healthcheck", nil)
		w := httptest.NewRecorder()
		healthCheck(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}
