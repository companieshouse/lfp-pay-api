package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api-core/validators"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/e5"
	"github.com/companieshouse/lfp-pay-api/service"
	"github.com/companieshouse/lfp-pay-api/utils"
	"gopkg.in/go-playground/validator.v9"
)

// handleEmailKafkaMessage allows us to mock the call to sendEmailKafkaMessage for unit tests
var handleEmailKafkaMessage = service.SendEmailKafkaMessage

var wg sync.WaitGroup

// PayResourceHandler will update the resource to mark it as paid and also tell the finance system that the
// transaction(s) associated with it are paid.
func PayResourceHandler(svc *service.PayableResourceService, e5Client *e5.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. get the payable resource our of the context. authorisation is already handled in the interceptor
		i := r.Context().Value(config.PayableResource)
		if i == nil {
			err := fmt.Errorf("no payable resource in context. check PayableAuthenticationInterceptor is installed")
			log.ErrorR(r, err)
			m := models.NewMessageResponse("no payable request present in request context")
			utils.WriteJSONWithStatus(w, r, m, http.StatusBadRequest)
			return
		}

		resource := i.(*models.PayableResource)

		log.Info("processing LFP payment", log.Data{
			"lfp_reference":  resource.Reference,
			"company_number": resource.CompanyNumber,
		})

		// 2. validate the request and check the reference number against the payment api to validate that is has
		// actually been paid
		var request models.PatchResourceRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			log.ErrorR(r, err, log.Data{"lfp_reference": resource.Reference})
			m := models.NewMessageResponse("there was a problem reading the request body")
			utils.WriteJSONWithStatus(w, r, m, http.StatusBadRequest)
			return
		}
		v := validator.New()
		err = v.Struct(request)

		if err != nil {
			log.ErrorR(r, err, log.Data{"lfp_reference": resource.Reference, "payment_id": request.Reference})
			m := models.NewMessageResponse("the request contained insufficient data and/or failed validation")
			utils.WriteJSONWithStatus(w, r, m, http.StatusBadRequest)
			return
		}

		payment, err := service.GetPaymentInformation(request.Reference, r)
		if err != nil {
			log.ErrorR(r, err, log.Data{"lfp_reference": resource.Reference, "payment_id": request.Reference})
			m := models.NewMessageResponse("the payable resource does not exist")
			utils.WriteJSONWithStatus(w, r, m, http.StatusBadRequest)
			return
		}

		err = validators.New().ValidateForPayment(*resource, *payment)
		if err != nil {
			m := models.NewMessageResponse("there was a problem validating this payment")
			utils.WriteJSONWithStatus(w, r, m, http.StatusBadRequest)
			return
		}

		wg.Add(3)

		go sendConfirmationEmail(resource, payment, r, w)
		go updateDatabase(resource, payment, svc, r, w)
		go updateE5(e5Client, resource, payment, svc, r, w)

		wg.Wait()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent) // This will not be set if status has already been set
	})
}
func sendConfirmationEmail(resource *models.PayableResource, payment *validators.PaymentInformation, r *http.Request, w http.ResponseWriter) {
	// Send confirmation email
	defer wg.Done()
	err := handleEmailKafkaMessage(*resource, r)
	if err != nil {
		log.ErrorR(r, err, log.Data{"lfp_reference": resource.Reference, "payment_id": payment.Reference})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Info("confirmation email sent to customer", log.Data{
		"lfp_reference":  resource.Reference,
		"company_number": resource.CompanyNumber,
		"email_address":  resource.CreatedBy.Email,
	})
}

func updateDatabase(resource *models.PayableResource, payment *validators.PaymentInformation, svc *service.PayableResourceService, r *http.Request, w http.ResponseWriter) {
	// Update the payable resource in the db
	defer wg.Done()
	err := svc.UpdateAsPaid(*resource, *payment)
	if err != nil {
		log.ErrorR(r, err, log.Data{"lfp_reference": resource.Reference, "payment_id": payment.Reference})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Info("payment resource is now marked as paid in db", log.Data{
		"lfp_reference":  resource.Reference,
		"company_number": resource.CompanyNumber,
	})
}

func updateE5(e5Client *e5.Client, resource *models.PayableResource, payment *validators.PaymentInformation, svc *service.PayableResourceService, r *http.Request, w http.ResponseWriter) {
	// Mark the resource as paid in e5
	defer wg.Done()
	err := service.MarkTransactionsAsPaid(svc, e5Client, *resource, *payment)
	if err != nil {
		log.ErrorR(r, err, log.Data{
			"lfp_reference":  resource.Reference,
			"company_number": resource.CompanyNumber,
		})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
