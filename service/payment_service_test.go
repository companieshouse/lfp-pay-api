package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/companieshouse/go-session-handler/httpsession"
	"github.com/companieshouse/go-session-handler/session"
	"github.com/companieshouse/lfp-pay-api-core/constants"
	"github.com/companieshouse/lfp-pay-api-core/validators"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

var paymentsResourceResponse = `
{
  "amount": "150",
  "reference": "123",
  "status": "paid",
  "created_by": {
    "email": "test@example.com",
    "forename": "test",
    "surname": "user",
    "id": "123"
  }
}
`

var paymentDetailsResponse = `
{
  "card_type": "VISA",
  "external_payment_id": "1234567890",
  "transaction_id": "1234",
  "payment_status": "paid"
}
`

func TestUnitPaymentInformation(t *testing.T) {
	Convey("IsPaid()", t, func() {
		Convey("is not paid if status is not 'paid'", func() {
			p := &validators.PaymentInformation{}
			So(p.IsPaid(), ShouldBeFalse)
		})

		Convey("is paid is status is 'paid", func() {
			p := &validators.PaymentInformation{Status: constants.Paid.String()}
			So(p.IsPaid(), ShouldBeTrue)
		})
	})

	Convey("PaymentAmount()", t, func() {
		Convey("parses string float successfully", func() {
			amounts := make(map[string]float64, 2)
			amounts["150"] = 150
			amounts["150.50"] = 150.50

			p := validators.PaymentInformation{}

			for s, f := range amounts {
				p.Amount = s
				So(p.PaymentAmount(), ShouldEqual, f)
			}
		})

		Convey("negative number returned when failing to parse float", func() {
			p := validators.PaymentInformation{Amount: "foo"}
			So(p.PaymentAmount(), ShouldEqual, -1)
		})
	})
}

func TestUnitGetPaymentInformation(t *testing.T) {
	paymentsAPIURL := "https://api-payments.companieshouse.gov.uk"

	Convey("GetPaymentResourceFromPaymentAPI", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		ctx := context.WithValue(context.Background(), httpsession.ContextKeySession, &session.Session{})
		r := &http.Request{}
		r = r.WithContext(ctx)

		Convey("payment resource error", func() {
			defer httpmock.Reset()
			httpmock.RegisterResponder(http.MethodGet, paymentsAPIURL+"/payments/123", httpmock.NewStringResponder(http.StatusTeapot, ""))

			resp, err := GetPaymentInformation("123", &http.Request{})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, `ch-api: got HTTP response code 418 with body: `)
		})

		Convey("payment details error", func() {
			defer httpmock.Reset()
			httpmock.RegisterResponder(http.MethodGet, paymentsAPIURL+"/payments/123", httpmock.NewStringResponder(http.StatusOK, paymentsResourceResponse))
			httpmock.RegisterResponder(http.MethodGet, paymentsAPIURL+"/private/payments/123/payment-details", httpmock.NewStringResponder(http.StatusTeapot, ""))

			resp, err := GetPaymentInformation("123", r)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, `ch-api: got HTTP response code 418 with body: `)
		})

		Convey("it returns a serialised version of the response", func() {
			defer httpmock.Reset()
			httpmock.RegisterResponder(http.MethodGet, paymentsAPIURL+"/payments/123", httpmock.NewStringResponder(http.StatusOK, paymentsResourceResponse))
			httpmock.RegisterResponder(http.MethodGet, paymentsAPIURL+"/private/payments/123/payment-details", httpmock.NewStringResponder(http.StatusOK, paymentDetailsResponse))

			resp, err := GetPaymentInformation("123", r)

			So(err, ShouldBeNil)
			So(resp.Amount, ShouldEqual, "150")
			So(resp.CreatedBy, ShouldEqual, "test@example.com")
			So(resp.Status, ShouldEqual, "paid")
			So(resp.Reference, ShouldEqual, "123")
			So(resp.CardType, ShouldEqual, "VISA")
			So(resp.ExternalPaymentID, ShouldEqual, "1234567890")
		})
	})
}
