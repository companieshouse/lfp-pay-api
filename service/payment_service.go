package service

import (
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/go-sdk-manager/manager"
	"github.com/companieshouse/lfp-pay-api-core/validators"
)

// GetPaymentInformation will attempt to get the payment resource from the payment platform.
// this can then be used to validate the state of a payment.
func GetPaymentInformation(id string, req *http.Request) (*validators.PaymentInformation, error) {
	publicSDK, err := manager.GetSDK(req)
	if err != nil {
		log.Error(err, log.Data{"payment_id": id})
		return nil, err
	}

	paymentResource, err := publicSDK.Payments.Get(id).Do()
	if err != nil {
		log.Error(err, log.Data{"payment_id": id})
		return nil, err
	}

	privateSDK, err := manager.GetPrivateSDK(req)
	if err != nil {
		log.Error(err, log.Data{"payment_id": id})
		return nil, err
	}

	paymentDetails, err := privateSDK.Payments.GetPaymentDetails(id).Do()
	if err != nil {
		log.Error(err, log.Data{"payment_id": id})
		return nil, err
	}

	paymentInformation := &validators.PaymentInformation{
		PaymentID:         id,
		CompletedAt:       paymentResource.CompletedAt,
		CreatedBy:         paymentResource.CreatedBy.Email,
		Amount:            paymentResource.Amount,
		Status:            paymentResource.Status,
		Reference:         paymentResource.Reference,
		ExternalPaymentID: paymentDetails.ExternalPaymentID,
		CardType:          paymentDetails.CardType,
	}

	return paymentInformation, nil
}
