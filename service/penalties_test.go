package service

import (
	j "encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api-core/validators"
	"github.com/companieshouse/lfp-pay-api/e5"
	"github.com/companieshouse/lfp-pay-api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

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

func TestUnitMarkTransactionsAsPaid(t *testing.T) {
	Convey("amount must be okay to parse as float", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockService := mocks.NewMockService(mockCtrl)
		svc := &PayableResourceService{DAO: mockService}

		c := &e5.Client{}
		r := models.PayableResource{}
		p := validators.PaymentInformation{Amount: "foo"}

		err := MarkTransactionsAsPaid(svc, c, r, p)
		So(err, ShouldNotBeNil)
	})

	Convey("E5 request errors", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockService := mocks.NewMockService(mockCtrl)
		svc := &PayableResourceService{DAO: mockService}

		Convey("failure in creating a new payment", func() {
			defer httpmock.Reset()
			e5Responder := httpmock.NewStringResponder(http.StatusBadRequest, e5ValidationError)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment", e5Responder)

			mockService.EXPECT().SaveE5Error("10000024", "123", e5.CreateAction).Return(errors.New(""))

			c := &e5.Client{}
			p := validators.PaymentInformation{Amount: "150", PaymentID: "123"}
			r := models.PayableResource{
				Reference:     "123",
				CompanyNumber: "10000024",
				Transactions: []models.TransactionItem{
					{TransactionID: "123", Amount: 150},
				},
			}

			err := MarkTransactionsAsPaid(svc, c, r, p)

			So(err, ShouldBeError, e5.ErrE5BadRequest)
		})

		Convey("failure in authorising a payment", func() {
			defer httpmock.Reset()
			e5Responder := httpmock.NewStringResponder(http.StatusBadRequest, e5ValidationError)
			okResponder := httpmock.NewBytesResponder(http.StatusOK, nil)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment", okResponder)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment/authorise", e5Responder)

			mockService.EXPECT().SaveE5Error("10000024", "123", e5.AuthoriseAction).Return(errors.New(""))

			c := &e5.Client{}
			p := validators.PaymentInformation{
				Amount:    "150",
				PaymentID: "123",
				CreatedBy: "test@example.com",
			}

			r := models.PayableResource{
				Reference:     "123",
				CompanyNumber: "10000024",
				Transactions: []models.TransactionItem{
					{TransactionID: "123", Amount: 150},
				},
			}

			err := MarkTransactionsAsPaid(svc, c, r, p)

			So(err, ShouldBeError, e5.ErrE5BadRequest)
		})

		Convey("failure in confirming a payment", func() {
			defer httpmock.Reset()
			e5Responder := httpmock.NewStringResponder(http.StatusBadRequest, e5ValidationError)
			okResponder := httpmock.NewBytesResponder(http.StatusOK, nil)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment", okResponder)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment/authorise", okResponder)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment/confirm", e5Responder)

			mockService.EXPECT().SaveE5Error("10000024", "123", e5.ConfirmAction).Return(errors.New(""))

			c := &e5.Client{}
			p := validators.PaymentInformation{
				Amount:    "150",
				PaymentID: "123",
				CreatedBy: "test@example.com",
			}

			r := models.PayableResource{
				Reference:     "123",
				CompanyNumber: "10000024",
				Transactions: []models.TransactionItem{
					{TransactionID: "123", Amount: 150},
				},
			}

			err := MarkTransactionsAsPaid(svc, c, r, p)

			So(err, ShouldBeError, e5.ErrE5BadRequest)
		})

		Convey("no errors when all 3 calls to E5 succeed", func() {
			defer httpmock.Reset()
			okResponder := httpmock.NewBytesResponder(http.StatusOK, nil)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment", okResponder)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment/authorise", okResponder)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment/confirm", okResponder)

			c := &e5.Client{}
			p := validators.PaymentInformation{
				Amount:    "150",
				PaymentID: "123",
				CreatedBy: "test@example.com",
			}

			r := models.PayableResource{
				Reference:     "123",
				CompanyNumber: "10000024",
				Transactions: []models.TransactionItem{
					{TransactionID: "123", Amount: 150},
				},
			}

			err := MarkTransactionsAsPaid(svc, c, r, p)

			So(err, ShouldBeNil)
		})

		Convey("paymentId (PUON) is prefixed with 'X'", func() {
			defer httpmock.Reset()

			// struct to decode the request body
			type body struct {
				PaymentID string `json:"paymentId"`
			}

			// check the payment id value before responding.
			paymentIDResponder := func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewBytesResponse(http.StatusOK, nil)
				defer req.Body.Close()
				var b body
				err := j.NewDecoder(req.Body).Decode(&b)
				if err != nil {
					return nil, errors.New("failed to read request body")
				}

				if b.PaymentID[0] != 'X' {
					return nil, errors.New("paymentId does not begin with an X")
				}
				return resp, nil
			}

			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment", paymentIDResponder)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment/authorise", paymentIDResponder)
			httpmock.RegisterResponder(http.MethodPost, "/arTransactions/payment/confirm", paymentIDResponder)

			c := &e5.Client{}
			p := validators.PaymentInformation{
				Amount:    "150",
				PaymentID: "123",
				CreatedBy: "test@example.com",
			}

			r := models.PayableResource{
				Reference:     "123",
				CompanyNumber: "10000024",
				Transactions: []models.TransactionItem{
					{TransactionID: "123", Amount: 150},
				},
			}

			err := MarkTransactionsAsPaid(svc, c, r, p)
			So(err, ShouldBeNil)

		})
	})
}
