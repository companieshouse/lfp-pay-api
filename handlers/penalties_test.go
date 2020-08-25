package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleGetPenalties(t *testing.T) {
	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("GET", "/company/NI038379/penalties/late-filing", nil)
		w := httptest.NewRecorder()
		HandleGetPenalties(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Request Body Invalid", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleGetPenalties(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

}
