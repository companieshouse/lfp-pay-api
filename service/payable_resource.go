package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api-core/validators"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/dao"
	"github.com/companieshouse/lfp-pay-api/e5"
	"github.com/companieshouse/lfp-pay-api/transformers"
)

var (
	// ErrPaymentNotFulfilled represents the scenario that the payment resource itself is not paid
	ErrPaymentNotFulfilled = errors.New("the resource you are trying to pay for has not been paid")
	// ErrAlreadyPaid represents when the LFP payable resource is already paid
	ErrAlreadyPaid = errors.New("the LFP has already been paid")
	// ErrLFPNotFound represents when the payable resource does not exist in the db
	ErrLFPNotFound = errors.New("the LFP does not exist")
	// ErrPayment represents an error when the payable resource amount does not match the amount in the payment resource
	ErrPayment = errors.New("there was a problem validating the payment")
)

// PayableResourceService contains the DAO for db access
type PayableResourceService struct {
	DAO    dao.Service
	Config *config.Config
}

// GetPayableResource retrieves the payable resource with the given company number and reference from the database
func (s *PayableResourceService) GetPayableResource(req *http.Request, companyNumber string, reference string) (*models.PayableResource, ResponseType, error) {
	payable, err := s.DAO.GetPayableResource(companyNumber, reference)
	if err != nil {
		err = fmt.Errorf("error getting payable resource from db: [%v]", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}
	if payable == nil {
		log.TraceR(req, "payable resource not found", log.Data{"company_number": companyNumber, "reference": reference})
		return nil, NotFound, nil
	}

	payableRest := transformers.PayableResourceDBToRequest(payable)

	return payableRest, Success, nil
}

// UpdateAsPaid will update the resource as paid and persist the changes in the database
func (s *PayableResourceService) UpdateAsPaid(resource models.PayableResource, payment validators.PaymentInformation) error {
	model, err := s.DAO.GetPayableResource(resource.CompanyNumber, resource.Reference)
	if err != nil {
		err = fmt.Errorf("error getting payable resource from db: [%v]", err)
		log.Error(err, log.Data{
			"lfp_reference":  resource.Reference,
			"company_number": resource.CompanyNumber,
		})
		return ErrLFPNotFound
	}

	// check if this resource has already been paid
	if model.IsPaid() {
		err = errors.New("this LFP has already been paid")
		log.Error(err, log.Data{
			"lfp_reference":  model.Reference,
			"company_number": model.CompanyNumber,
			"payment_id":     model.Data.Payment.Reference,
		})
		return ErrAlreadyPaid
	}

	model.Data.Payment.Reference = payment.Reference
	model.Data.Payment.Status = payment.Status
	model.Data.Payment.PaidAt = &payment.CompletedAt
	model.Data.Payment.Amount = payment.Amount

	return s.DAO.UpdatePaymentDetails(model)
}

// RecordE5CommandError will mark the resource as having failed to update E5.
func (s *PayableResourceService) RecordE5CommandError(resource models.PayableResource, action e5.Action) error {
	return s.DAO.SaveE5Error(resource.CompanyNumber, resource.Reference, action)
}
