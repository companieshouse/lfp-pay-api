package e5

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/go-playground/validator.v9"
)

func hasFieldError(field, tag string, errs validator.ValidationErrors) bool {
	for _, e := range errs {
		f := e.Field()
		t := e.Tag()
		if f == field && t == tag {
			return true
		}
	}
	return false
}

func TestUnitClient_CreatePayment(t *testing.T) {
	e5 := NewClient("foo", "https://e5")
	url := "https://e5/arTransactions/payment?ADV_userName=foo"

	Convey("creating a payment", t, func() {
		input := &CreatePaymentInput{
			CompanyCode:   "LP",
			CompanyNumber: "1000024",
			PaymentID:     "1234",
			TotalValue:    100,
			Transactions: []*CreatePaymentTransaction{
				{
					Reference: "1234",
					Value:     100,
				},
			},
		}

		Convey("response should be unsuccessful when there is a 500 error from E5", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpErr := &apiErrorResponse{Code: 500, Message: "test error"}
			responder, _ := httpmock.NewJsonResponder(http.StatusInternalServerError, httpErr)
			httpmock.RegisterResponder(http.MethodPost, url, responder)

			err := e5.CreatePayment(input)

			So(err, ShouldBeError, ErrE5InternalServer)
		})

		Convey("response should be unsuccessful when the company does not exist", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpErr := &apiErrorResponse{Code: 404, Message: "company not found"}
			responder, _ := httpmock.NewJsonResponder(http.StatusNotFound, httpErr)
			httpmock.RegisterResponder(http.MethodPost, url, responder)

			err := e5.CreatePayment(input)

			So(err, ShouldBeError, ErrE5NotFound)
		})

		Convey("response should be successful if a 200 is returned from E5", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			responder := httpmock.NewBytesResponder(http.StatusOK, nil)
			httpmock.RegisterResponder(http.MethodPost, url, responder)

			err := e5.CreatePayment(input)

			So(err, ShouldBeNil)
		})
	})
}

// this is the response returned by e5 when the company number is incorrect i.e. no transactions exist
var e5EmptyResponse = `
{
  "page" : {
    "size" : 0,
    "totalElements" : 0,
    "totalPages" : 1,
    "number" : 0
  },
  "data" : [ ]
}`

var e5TransactionResponse = `
{
  "page" : {
    "size" : 1,
    "totalElements" : 1,
    "totalPages" : 1,
    "number" : 0
  },
  "data" : [ {
    "companyCode" : "LP",
    "ledgerCode" : "EW",
    "customerCode" : "10000024",
    "transactionReference" : "00378420",
    "transactionDate" : "2017-11-28",
    "madeUpDate" : "2017-02-28",
    "amount" : 150,
    "outstandingAmount" : 150,
    "isPaid" : false,
    "transactionType" : "1",
    "transactionSubType" : "EU",
    "typeDescription" : "Penalty Ltd Wel & Eng <=1m     LTDWA    ",
    "dueDate" : "2017-12-12"
  }]
}
`

var e5ValidationError = `
{
  "httpStatusCode" : 400,
  "status" : "BAD_REQUEST",
  "timestamp" : "2019-07-07T18:40:07Z",
  "messageCode" : null,
  "message" : "Constraint Validation error",
  "debugMessage" : null,
  "subErrors" : [ {
    "object" : "String",
    "field" : "companyCode",
    "rejectedValue" : "LPs",
    "message" : "size must be between 0 and 2"
  } ]
}
`

func TestUnitClient_GetTransactions(t *testing.T) {
	Convey("getting a list of transactions for a company", t, func() {
		e5 := NewClient("foo", "https://e5")
		url := "https://e5/arTransactions/10000024?ADV_userName=foo&companyCode=LP&fromDate=1990-01-01"

		Convey("company does not exist or no transactions returned", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			responder := httpmock.NewStringResponder(http.StatusOK, e5EmptyResponse)
			httpmock.RegisterResponder(http.MethodGet, url, responder)

			r, err := e5.GetTransactions(&GetTransactionsInput{CompanyNumber: "10000024", CompanyCode: "LP"})

			So(err, ShouldBeNil)
			So(r.Transactions, ShouldBeEmpty)
			So(r.Page.Size, ShouldEqual, 0)
		})

		Convey("should return a list of transactions", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			responder := httpmock.NewStringResponder(http.StatusOK, e5TransactionResponse)
			httpmock.RegisterResponder(http.MethodGet, url, responder)

			r, err := e5.GetTransactions(&GetTransactionsInput{CompanyNumber: "10000024", CompanyCode: "LP"})

			So(err, ShouldBeNil)
			So(r.Transactions, ShouldHaveLength, 1)
		})

		Convey("using an incorrect company code", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			responder := httpmock.NewStringResponder(http.StatusBadRequest, e5ValidationError)
			httpmock.RegisterResponder(http.MethodGet, url, responder)

			r, err := e5.GetTransactions(&GetTransactionsInput{CompanyNumber: "10000024", CompanyCode: "LP"})

			So(r, ShouldBeNil)
			So(err, ShouldBeError, ErrE5BadRequest)
		})

	})
}

func TestUnitClient_AuthorisePayment(t *testing.T) {
	e5 := NewClient("foo", "https://e5")
	url := "https://e5/arTransactions/payment/authorise?ADV_userName=foo"

	Convey("email, paymentId are required parameters", t, func() {
		input := &AuthorisePaymentInput{}

		err := e5.AuthorisePayment(input)

		So(err, ShouldNotBeNil)

		errors := err.(validator.ValidationErrors)

		So(errors, ShouldHaveLength, 3)
		So(hasFieldError("Email", "required", errors), ShouldBeTrue)
		So(hasFieldError("PaymentID", "required", errors), ShouldBeTrue)
		So(hasFieldError("CompanyCode", "required", errors), ShouldBeTrue)
	})

	Convey("500 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusInternalServerError, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.AuthorisePayment(&AuthorisePaymentInput{PaymentID: "123", Email: "test@example.com", CompanyCode: "LP"})

		So(err, ShouldBeError, ErrE5InternalServer)
	})

	Convey("400 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusBadRequest, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.AuthorisePayment(&AuthorisePaymentInput{PaymentID: "123", Email: "test@example.com", CompanyCode: "LP"})

		So(err, ShouldBeError, ErrE5BadRequest)
	})

	Convey("404 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusNotFound, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.AuthorisePayment(&AuthorisePaymentInput{PaymentID: "123", Email: "test@example.com", CompanyCode: "LP"})

		So(err, ShouldBeError, ErrE5NotFound)
	})

	Convey("403 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusForbidden, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.AuthorisePayment(&AuthorisePaymentInput{PaymentID: "123", Email: "test@example.com", CompanyCode: "LP"})

		So(err, ShouldBeError, ErrUnexpectedServerError)
	})

	Convey("everything okay when there are not errors", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusOK, "")
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		input := &AuthorisePaymentInput{PaymentID: "123", Email: "test@example.com", CompanyCode: "LP"}
		err := e5.AuthorisePayment(input)

		So(err, ShouldBeNil)
	})
}

func TestUnitClient_Confirm(t *testing.T) {
	e5 := NewClient("foo", "https://e5")
	url := "https://e5/arTransactions/payment/confirm?ADV_userName=foo"
	input := &PaymentActionInput{PaymentID: "123", CompanyCode: "LP"}

	Convey("paymentId is required", t, func() {
		err := e5.ConfirmPayment(&PaymentActionInput{})

		errors := err.(validator.ValidationErrors)

		So(err, ShouldNotBeNil)
		So(hasFieldError("PaymentID", "required", errors), ShouldBeTrue)
	})

	Convey("500 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusInternalServerError, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.ConfirmPayment(input)

		So(err, ShouldBeError, ErrE5InternalServer)
	})

	Convey("400 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusBadRequest, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.ConfirmPayment(input)

		So(err, ShouldBeError, ErrE5BadRequest)
	})

	Convey("404 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusNotFound, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.ConfirmPayment(input)

		So(err, ShouldBeError, ErrE5NotFound)
	})

	Convey("403 error from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusForbidden, e5ValidationError)
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.ConfirmPayment(input)

		So(err, ShouldBeError, ErrUnexpectedServerError)
	})

	Convey("successful confirmation", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		responder := httpmock.NewStringResponder(http.StatusOK, "")
		httpmock.RegisterResponder(http.MethodPost, url, responder)

		err := e5.ConfirmPayment(input)

		So(err, ShouldBeNil)
	})
}
