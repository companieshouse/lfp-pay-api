package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/lfp-pay-api-core/models"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitWriteJSON(t *testing.T) {
	Convey("Failure to marshal json", t, func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		// causes an UnsupportedTypeError
		WriteJSON(w, r, make(chan int))

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "application/json")
		So(w.Body.String(), ShouldEqual, "")
	})

	Convey("contents are written as json", t, func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		WriteJSON(w, r, &models.CreatedPayableResourceLinks{})

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "application/json")
		So(w.Body.String(), ShouldEqual, "{\"self\":\"\"}\n")
	})
}

func TestUnitGetCompanyNumber(t *testing.T) {
	Convey("Get Company Number", t, func() {
		vars := map[string]string{
			"company_number": "12345",
		}
		companyNumber, err := GetCompanyNumberFromVars(vars)
		So(companyNumber, ShouldEqual, "12345")
		So(err, ShouldBeNil)
	})

	Convey("No Company Number", t, func() {
		vars := map[string]string{}
		companyNumber, err := GetCompanyNumberFromVars(vars)
		So(companyNumber, ShouldBeEmpty)
		So(err.Error(), ShouldEqual, "company number not supplied")
	})
}
