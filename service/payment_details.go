package service

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/transformers"
)

// PaymentDetailsService contains the PayableResourceService for updating the resource with payment details after a successful/failed payment
type PaymentDetailsService struct {
	PayableResourceService *PayableResourceService // needed when implemnting PATCH endpoint from payment-processed-consumer
}

// GetPaymentDetailsFromPayableResource transforms a PayableResource into its corresponding Payment details resource
func (service *PaymentDetailsService) GetPaymentDetailsFromPayableResource(req *http.Request, payable *models.PayableResource) (*models.PaymentDetails, ResponseType, error) {
	paymentDetails := transformers.PayableResourceToPaymentDetails(payable)

	if len(paymentDetails.Items) == 0 {
		err := fmt.Errorf("no items in payment details transformed from payable resource [%s]", payable.Reference)
		log.ErrorR(req, err, log.Data{"company_number": payable.CompanyNumber, "reference": payable.Reference, "payable_transactions": payable.Transactions})
		return nil, InvalidData, err
	}

	return paymentDetails, Success, nil
}
