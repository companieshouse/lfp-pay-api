package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/service"
	"github.com/companieshouse/lfp-pay-api/utils"
	"github.com/gorilla/mux"
)

// HandleGetPenalties retrieves the penalty details for the supplied company number from e5
func HandleGetPenalties(w http.ResponseWriter, req *http.Request) {
	log.InfoR(req, "start GET penalties request from e5")

	// Check for a company number in request
	vars := mux.Vars(req)
	companyNumber, err := utils.GetCompanyNumberFromVars(vars)
	if err != nil {
		log.ErrorR(req, err)
		m := models.NewMessageResponse("company number is not in request context")
		utils.WriteJSONWithStatus(w, req, m, http.StatusBadRequest)
		return
	}

	companyNumber = strings.ToUpper(companyNumber)

	// Call service layer to handle request to E5
	transactionListResponse, responseType, err := service.GetPenalties(companyNumber)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error calling e5 to get transactions: %v", err))
		switch responseType {
		case service.InvalidData:
			m := models.NewMessageResponse("failed to read finance transactions")
			utils.WriteJSONWithStatus(w, req, m, http.StatusBadRequest)
			return
		case service.Error:
		default:
			m := models.NewMessageResponse("there was a problem communicating with the finance backend")
			utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
			return
		}
	}

	// response body contains fully decorated REST model
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(transactionListResponse)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		return
	}

	log.InfoR(req, "Successfully GET penalties from e5", log.Data{"company_number": companyNumber})
}
