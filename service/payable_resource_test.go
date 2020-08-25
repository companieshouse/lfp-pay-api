package service

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/lfp-pay-api-core/constants"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api-core/validators"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

func createMockPayableResourceService(mockDAO *mocks.MockService, cfg *config.Config) PayableResourceService {
	return PayableResourceService{
		DAO:    mockDAO,
		Config: cfg,
	}
}

func TestUnitGetPayableResource(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	cfg, _ := config.Get()

	Convey("Error getting payable resource from DB", t, func() {
		mock := mocks.NewMockService(mockCtrl)
		mockPayableService := createMockPayableResourceService(mock, cfg)
		mock.EXPECT().GetPayableResource("12345678", gomock.Any()).Return(&models.PayableResourceDao{}, fmt.Errorf("error"))

		req := httptest.NewRequest("Get", "/test", nil)

		payableResource, status, err := mockPayableService.GetPayableResource(req, "12345678", "1234")
		So(payableResource, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting payable resource from db: [error]")
	})

	Convey("Payable resource not found", t, func() {
		mock := mocks.NewMockService(mockCtrl)
		mockPayableService := createMockPayableResourceService(mock, cfg)
		mock.EXPECT().GetPayableResource("12345678", "invalid").Return(nil, nil)

		req := httptest.NewRequest("Get", "/test", nil)

		payableResource, status, err := mockPayableService.GetPayableResource(req, "12345678", "invalid")
		So(payableResource, ShouldBeNil)
		So(status, ShouldEqual, NotFound)
		So(err, ShouldBeNil)
	})

	Convey("Get Payable resource - success - Single transaction", t, func() {
		mock := mocks.NewMockService(mockCtrl)
		mockPayableService := createMockPayableResourceService(mock, cfg)

		txs := map[string]models.TransactionDao{
			"abcd": models.TransactionDao{Amount: 5},
		}
		t := time.Now().Truncate(time.Millisecond)
		mock.EXPECT().GetPayableResource("12345678", gomock.Any()).Return(
			&models.PayableResourceDao{
				CompanyNumber: "12345678",
				Reference:     "1234",
				Data: models.PayableResourceDataDao{
					Etag:      "qwertyetag1234",
					CreatedAt: &t,
					CreatedBy: models.CreatedByDao{
						ID:       "identity",
						Email:    "test@user.com",
						Forename: "some",
						Surname:  "body",
					},
					Links: models.PayableResourceLinksDao{
						Self:    "/company/12345678/penalties/late-filing/payable/1234",
						Payment: "/company/12345678/penalties/late-filing/payable/1234/payment",
					},
					Transactions: txs,
					Payment: models.PaymentDao{
						Status: constants.Pending.String(),
						Amount: "5",
					},
				},
			},
			nil,
		)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		payableResource, status, err := mockPayableService.GetPayableResource(req, "12345678", "1234")

		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)
		So(payableResource.CompanyNumber, ShouldEqual, "12345678")
		So(payableResource.Reference, ShouldEqual, "1234")
		So(payableResource.Etag, ShouldEqual, "qwertyetag1234")
		So(payableResource.CreatedAt, ShouldEqual, &t)
		So(payableResource.CreatedBy.ID, ShouldEqual, "identity")
		So(payableResource.CreatedBy.Email, ShouldEqual, "test@user.com")
		So(payableResource.CreatedBy.Forename, ShouldEqual, "some")
		So(payableResource.CreatedBy.Surname, ShouldEqual, "body")
		So(payableResource.Links.Self, ShouldEqual, "/company/12345678/penalties/late-filing/payable/1234")
		So(payableResource.Links.Payment, ShouldEqual, "/company/12345678/penalties/late-filing/payable/1234/payment")
		So(payableResource.Payment.Amount, ShouldEqual, "5")
		So(payableResource.Payment.Status, ShouldEqual, "pending")
		So(len(payableResource.Transactions), ShouldEqual, 1)
		So(payableResource.Transactions[0].Amount, ShouldEqual, 5)
	})

	Convey("Get Payable resource - success - Multiple transactions", t, func() {
		mock := mocks.NewMockService(mockCtrl)
		mockPayableService := createMockPayableResourceService(mock, cfg)

		txs := map[string]models.TransactionDao{
			"abcd": models.TransactionDao{Amount: 5},
			"wxyz": models.TransactionDao{Amount: 10},
		}
		t := time.Now().Truncate(time.Millisecond)
		mock.EXPECT().GetPayableResource("12345678", gomock.Any()).Return(
			&models.PayableResourceDao{
				CompanyNumber: "12345678",
				Reference:     "1234",
				Data: models.PayableResourceDataDao{
					Etag:      "qwertyetag1234",
					CreatedAt: &t,
					CreatedBy: models.CreatedByDao{
						ID:       "identity",
						Email:    "test@user.com",
						Forename: "some",
						Surname:  "body",
					},
					Links: models.PayableResourceLinksDao{
						Self:    "/company/12345678/penalties/late-filing/payable/1234",
						Payment: "/company/12345678/penalties/late-filing/payable/1234/payment",
					},
					Transactions: txs,
					Payment: models.PaymentDao{
						Status:    constants.Paid.String(),
						Amount:    "15",
						Reference: "payref",
						PaidAt:    &t,
					},
				},
			},
			nil,
		)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		payableResource, status, err := mockPayableService.GetPayableResource(req, "12345678", "1234")

		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)
		So(payableResource.CompanyNumber, ShouldEqual, "12345678")
		So(payableResource.Reference, ShouldEqual, "1234")
		So(payableResource.Etag, ShouldEqual, "qwertyetag1234")
		So(payableResource.CreatedAt, ShouldEqual, &t)
		So(payableResource.CreatedBy.ID, ShouldEqual, "identity")
		So(payableResource.CreatedBy.Email, ShouldEqual, "test@user.com")
		So(payableResource.CreatedBy.Forename, ShouldEqual, "some")
		So(payableResource.CreatedBy.Surname, ShouldEqual, "body")
		So(payableResource.Links.Self, ShouldEqual, "/company/12345678/penalties/late-filing/payable/1234")
		So(payableResource.Links.Payment, ShouldEqual, "/company/12345678/penalties/late-filing/payable/1234/payment")
		So(payableResource.Payment.Amount, ShouldEqual, "15")
		So(payableResource.Payment.Status, ShouldEqual, "paid")
		So(payableResource.Payment.Reference, ShouldEqual, "payref")
		So(payableResource.Payment.PaidAt, ShouldEqual, &t)
		So(len(payableResource.Transactions), ShouldEqual, 2)
		So(payableResource.Transactions[0].Amount+payableResource.Transactions[1].Amount, ShouldEqual, 15) // array order can change - sum can't
	})
}

func TestUnitPayableResourceService_UpdateAsPaid(t *testing.T) {
	Convey("PayableResourceService.UpdateAsPaid", t, func() {
		Convey("Payable resource must exist", func() {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockDaoService := mocks.NewMockService(mockCtrl)
			mockDaoService.EXPECT().GetPayableResource(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))
			svc := PayableResourceService{DAO: mockDaoService}

			err := svc.UpdateAsPaid(models.PayableResource{}, validators.PaymentInformation{})

			So(err, ShouldBeError, ErrLFPNotFound)
		})

		Convey("LFP payable resource must not have already been paid", func() {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			dataModel := &models.PayableResourceDao{
				Data: models.PayableResourceDataDao{
					Payment: models.PaymentDao{
						Status: constants.Paid.String(),
					},
				},
			}
			mockDaoService := mocks.NewMockService(mockCtrl)
			mockDaoService.EXPECT().GetPayableResource(gomock.Any(), gomock.Any()).Return(dataModel, nil)
			svc := PayableResourceService{DAO: mockDaoService}

			err := svc.UpdateAsPaid(models.PayableResource{}, validators.PaymentInformation{Status: constants.Paid.String()})

			So(err, ShouldBeError, ErrAlreadyPaid)
		})

		Convey("payment details are saved to db", func() {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			dataModel := &models.PayableResourceDao{
				Data: models.PayableResourceDataDao{
					Payment: models.PaymentDao{},
				},
			}
			mockDaoService := mocks.NewMockService(mockCtrl)
			mockDaoService.EXPECT().GetPayableResource(gomock.Any(), gomock.Any()).Return(dataModel, nil)
			mockDaoService.EXPECT().UpdatePaymentDetails(gomock.Any()).Times(1)
			svc := PayableResourceService{DAO: mockDaoService}

			layout := "2006-01-02T15:04:05.000Z"
			str := "2014-11-12T11:45:26.371Z"
			completedAt, _ := time.Parse(layout, str)

			paymentResponse := validators.PaymentInformation{
				Reference:   "123",
				Amount:      "150",
				Status:      "paid",
				CompletedAt: completedAt,
				CreatedBy:   "test@example.com",
			}

			err := svc.UpdateAsPaid(models.PayableResource{}, paymentResponse)

			So(err, ShouldBeNil)
			So(dataModel.Data.Payment.Status, ShouldEqual, paymentResponse.Status)
			So(dataModel.Data.Payment.PaidAt, ShouldNotBeNil)
			So(dataModel.Data.Payment.Amount, ShouldEqual, paymentResponse.Amount)
			So(dataModel.Data.Payment.Reference, ShouldEqual, paymentResponse.Reference)
		})
	})
}
