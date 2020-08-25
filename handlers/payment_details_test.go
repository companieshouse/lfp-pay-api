package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"

	. "github.com/smartystreets/goconvey/convey"
)

func serveGetPaymentDetailsHandler(payableResource *models.PayableResource) *httptest.ResponseRecorder {
	path := "/company/12345/penalties/late-filing/payable/321/penalties/late-filing"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()

	if payableResource != nil {
		ctx := context.WithValue(req.Context(), config.PayableResource, payableResource)
		req = req.WithContext(ctx)
	}

	HandleGetPaymentDetails(res, req)

	return res
}

func TestUnitHandleGetPaymentDetails(t *testing.T) {

	Convey("No payable resource in request context", t, func() {
		res := serveGetPaymentDetailsHandler(nil)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Payment Details not found due to no costs", t, func() {
		t := time.Now().Truncate(time.Millisecond)

		payable := models.PayableResource{
			CompanyNumber: "12345678",
			Reference:     "abcdef",
			Links: models.PayableResourceLinks{
				Self:    "/company/12345678/penalties/late-filing/abcdef",
				Payment: "/company/12345678/penalties/late-filing/abcdef/payment",
			},
			Etag:      "qwertyetag1234",
			CreatedAt: &t,
			CreatedBy: models.CreatedBy{
				Email: "test@user.com",
				ID:    "uz3r1D_H3r3",
			},
			Payment: models.Payment{
				Amount: "5",
				Status: "pending",
			},
		}

		res := serveGetPaymentDetailsHandler(&payable)
		So(res.Code, ShouldEqual, http.StatusNotFound)
	})

	Convey("Payment Details success", t, func() {
		t := time.Now().Truncate(time.Millisecond)

		payable := models.PayableResource{
			CompanyNumber: "12345678",
			Reference:     "abcdef",
			Links: models.PayableResourceLinks{
				Self:    "/company/12345678/penalties/late-filing/abcdef",
				Payment: "/company/12345678/penalties/late-filing/abcdef/payment",
			},
			Etag:      "qwertyetag1234",
			CreatedAt: &t,
			CreatedBy: models.CreatedBy{
				Email: "test@user.com",
				ID:    "uz3r1D_H3r3",
			},
			Transactions: []models.TransactionItem{
				models.TransactionItem{
					Amount:        5,
					Type:          "penalty",
					TransactionID: "0987654321",
				},
			},
			Payment: models.Payment{
				Amount: "5",
				Status: "pending",
			},
		}

		res := serveGetPaymentDetailsHandler(&payable)
		So(res.Code, ShouldEqual, http.StatusOK)
	})

}
