package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/service"
	"github.com/companieshouse/lfp-pay-api/utils"
)

// HandleGetPaymentDetails retrieves costs for a supplied company number and reference.
func HandleGetPaymentDetails(w http.ResponseWriter, req *http.Request) {

	// get payable resource from context, put there by PayableResourceAuthenticationInterceptor
	payableResource, ok := req.Context().Value(config.PayableResource).(*models.PayableResource)

	if !ok {
		log.ErrorR(req, fmt.Errorf("invalid PayableResource in request context"))
		m := models.NewMessageResponse("the payable resource is not present in the request context")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	// Get the payment details from the payable resource
	paymentDetails, responseType, err := paymentDetailsService.GetPaymentDetailsFromPayableResource(req, payableResource)
	logData := log.Data{"company_number": payableResource.CompanyNumber, "reference": payableResource.Reference}
	if err != nil {
		switch responseType {
		case service.InvalidData:
			log.DebugR(req, fmt.Sprintf("invalid data getting payment details from payable resource so returning not found [%s]", err.Error()), logData)
			m := models.NewMessageResponse("payable resource does not exist or has insufficient data")
			utils.WriteJSONWithStatus(w, req, m, http.StatusNotFound)
			return
		default:
			log.ErrorR(req, fmt.Errorf("error when getting payment details from PayableResource: [%v]", err), logData)
			m := models.NewMessageResponse("payable resource does not exist or has insufficient data")
			utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
			return
		}
	}

	utils.WriteJSON(w, req, paymentDetails)

	log.InfoR(req, "Successful GET request for payment details", logData)

}
