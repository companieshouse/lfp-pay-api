package interceptors

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/service"
	"github.com/companieshouse/lfp-pay-api/utils"
	"github.com/gorilla/mux"
)

// PayableAuthenticationInterceptor contains the payable_resource service used in the interceptor
type PayableAuthenticationInterceptor struct {
	Service service.PayableResourceService
}

// PayableAuthenticationIntercept checks that the user is authenticated for the payable_resource
func (payableAuthInterceptor *PayableAuthenticationInterceptor) PayableAuthenticationIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		companyNumber, payableID, identityType, err := preCheckRequest(w, r)
		if err {
			return
		}

		authorisedUser := ""

		if identityType == authentication.Oauth2IdentityType {
			// Get user details from context, passed in by UserAuthenticationInterceptor
			userDetails, ok := r.Context().Value(authentication.ContextKeyUserDetails).(authentication.AuthUserDetails)
			if !ok {
				log.ErrorR(r, fmt.Errorf("PayableAuthenticationInterceptor error: invalid AuthUserDetails from UserAuthenticationInterceptor"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Get user details from request
			authorisedUser = userDetails.ID
			if authorisedUser == "" {
				log.ErrorR(r, fmt.Errorf("PayableAuthenticationInterceptor unauthorised: no authorised identity"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		payableResource, ret := writeHeader(w, r, payableAuthInterceptor,
			companyNumber, payableID)
		if ret {
			return
		}

		// Store payable_resource in context to use later in the handler
		ctx := context.WithValue(r.Context(), config.PayableResource, payableResource)

		// Set up variables that are used to determine authorisation below
		isGetRequest := http.MethodGet == r.Method
		authUserIsPayableResourceCreator := authorisedUser == payableResource.CreatedBy.ID
		authUserHasPenaltyLookupRole := authentication.IsRoleAuthorised(r, utils.AdminPenaltyLookupRole)
		isAPIKeyRequest := identityType == authentication.APIKeyIdentityType
		apiKeyHasElevatedPrivileges := authentication.IsKeyElevatedPrivilegesAuthorised(r)

		// Set up debug map for logging at each exit point
		debugMap := log.Data{
			"company_number":                             companyNumber,
			"payable_resource_id":                        payableID,
			"auth_user_is_payable_resource_creator":      authUserIsPayableResourceCreator,
			"auth_user_has_payable_resource_lookup_role": authUserHasPenaltyLookupRole,
			"api_key_has_elevated_privileges":            apiKeyHasElevatedPrivileges,
			"request_method":                             r.Method,
		}

		booleans := booleanParameters{authUserIsPayableResourceCreator,
			authUserHasPenaltyLookupRole, isGetRequest,
			isAPIKeyRequest, apiKeyHasElevatedPrivileges}

		checkAllowedThrough(w, r, debugMap, next, ctx, booleans)
	})
}

func preCheckRequest(w http.ResponseWriter, r *http.Request) (string, string, string, bool) {
	// Check for a company_number and payable_id in request
	vars := mux.Vars(r)
	companyNumber := strings.ToUpper(vars["company_number"])
	if companyNumber == "" {
		log.InfoR(r, "PayableAuthenticationInterceptor error: no company_number")
		w.WriteHeader(http.StatusBadRequest)
		return "", "", "", true
	}
	payableID := vars["payable_id"]
	if payableID == "" {
		log.InfoR(r, "PayableAuthenticationInterceptor error: no payable_id")
		w.WriteHeader(http.StatusBadRequest)
		return "", "", "", true
	}

	// Get identity type from request
	identityType := authentication.GetAuthorisedIdentityType(r)
	if isUnauthorizedIdentityType(identityType) {
		log.InfoR(r, "PayableAuthenticationInterceptor unauthorised: not oauth2 or API key identity type")
		w.WriteHeader(http.StatusUnauthorized)
		return "", "", "", true
	}
	return companyNumber, payableID, identityType, false
}

func checkAllowedThrough(w http.ResponseWriter,
	r *http.Request,
	debugMap log.Data,
	next http.Handler,
	ctx context.Context,
	b booleanParameters) {
	// Now that we have the payable resource data and authorized user there are
	// multiple cases that can be allowed through:
	switch {
	case b.authUserIsPayableResourceCreator:
		// 1) Authorized user created the payable_resource
		log.InfoR(r, "PayableAuthenticationInterceptor authorised as creator", debugMap)
		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	case isAuthorizedGetRequest(b.authUserHasPenaltyLookupRole, b.isGetRequest):
		// 2) Authorized user has permission to lookup any payable_resource and
		// request is a GET i.e. to see payable_resource data but not modify/delete
		log.InfoR(r, "PayableAuthenticationInterceptor authorised as admin penalty lookup role on GET", debugMap)
		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	case isAuthorizedApiKeyRequest(b.isAPIKeyRequest, b.apiKeyHasElevatedPrivileges):
		// 3) Authorized API key with elevated privileges is an internal API key
		// that we trust
		log.InfoR(r, "PayableAuthenticationInterceptor authorised as api key elevated user", debugMap)
		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	default:
		// If none of the above conditions above are met then the request is
		// unauthorized
		w.WriteHeader(http.StatusUnauthorized)
		log.InfoR(r, "PayableAuthenticationInterceptor unauthorised", debugMap)
	}
}

func writeHeader(w http.ResponseWriter,
	r *http.Request,
	payableAuthInterceptor *PayableAuthenticationInterceptor,
	companyNumber string,
	payableID string) (*models.PayableResource, bool) {
	// Get the payable resource from the ID in request
	payableResource, responseType, err := payableAuthInterceptor.Service.GetPayableResource(r, companyNumber, payableID)
	if err != nil {
		log.ErrorR(r, fmt.Errorf("PayableAuthenticationInterceptor error when retrieving payable_resource: [%v]", err), log.Data{"service_response_type": responseType.String()})
		switch responseType {
		case service.Forbidden:
			w.WriteHeader(http.StatusForbidden)
			return nil, true
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return nil, true
		}
	}

	if responseType == service.NotFound {
		log.InfoR(r, "PayableAuthenticationInterceptor not found", log.Data{"payable_id": payableID, "company_number": companyNumber})
		w.WriteHeader(http.StatusNotFound)
		return nil, true
	}

	if responseType != service.Success {
		log.ErrorR(r, fmt.Errorf("PayableAuthenticationInterceptor error when retrieving payable_resource. Status: [%s]", responseType.String()))
		w.WriteHeader(http.StatusInternalServerError)
		return nil, true
	}
	return payableResource, false
}

func isUnauthorizedIdentityType(identityType string) bool {
	return !(identityType == authentication.Oauth2IdentityType ||
		identityType == authentication.APIKeyIdentityType)
}

func isAuthorizedApiKeyRequest(isAPIKeyRequest bool,
	apiKeyHasElevatedPrivileges bool) bool {
	return isAPIKeyRequest && apiKeyHasElevatedPrivileges
}

func isAuthorizedGetRequest(authUserHasPenaltyLookupRole bool,
	isGetRequest bool) bool {
	return authUserHasPenaltyLookupRole && isGetRequest
}

type booleanParameters struct {
	authUserIsPayableResourceCreator bool
	authUserHasPenaltyLookupRole     bool
	isGetRequest                     bool
	isAPIKeyRequest                  bool
	apiKeyHasElevatedPrivileges      bool
}
