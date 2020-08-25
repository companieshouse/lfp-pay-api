package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/utils"
)

// HandleGetPayableResource retrieves the payable resource from request context
func HandleGetPayableResource(w http.ResponseWriter, req *http.Request) {

	// get payable resource from context, put there by PayableResourceAuthenticationInterceptor
	payableResource, ok := req.Context().Value(config.PayableResource).(*models.PayableResource)

	if !ok {
		log.ErrorR(req, fmt.Errorf("invalid PayableResource in request context"))
		m := models.NewMessageResponse("the payable resource is not present in the request context")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, req, payableResource)
}
