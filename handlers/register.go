package handlers

import (
	"net/http"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/dao"
	"github.com/companieshouse/lfp-pay-api/e5"
	"github.com/companieshouse/lfp-pay-api/interceptors"
	"github.com/companieshouse/lfp-pay-api/middleware"
	"github.com/companieshouse/lfp-pay-api/service"
	"github.com/gorilla/mux"
)

var payableResourceService *service.PayableResourceService
var paymentDetailsService *service.PaymentDetailsService

// Register defines the route mappings for the main router and it's subrouters
func Register(mainRouter *mux.Router, cfg *config.Config, svc dao.Service) {

	payableResourceService = &service.PayableResourceService{
		Config: cfg,
		DAO:    svc,
	}

	paymentDetailsService = &service.PaymentDetailsService{
		PayableResourceService: payableResourceService,
	}

	payableAuthInterceptor := interceptors.PayableAuthenticationInterceptor{
		Service: *payableResourceService,
	}

	// only oauth2 users can create payable resources
	oauth2OnlyInterceptor := &authentication.OAuth2OnlyAuthenticationInterceptor{
		StrictPaths: map[string][]string{
			"/company/{company_number}/penalties/late-filing/payable": []string{http.MethodPost},
		},
	}

	e5Client := e5.NewClient(cfg.E5Username, cfg.E5APIURL)

	userAuthInterceptor := &authentication.UserAuthenticationInterceptor{
		AllowAPIKeyUser:                true,
		RequireElevatedAPIKeyPrivilege: true,
	}

	mainRouter.HandleFunc("/healthcheck", healthCheck).Methods(http.MethodGet).Name("healthcheck")
	mainRouter.HandleFunc("/healthcheck/finance-system", HandleHealthCheckFinanceSystem).Methods(http.MethodGet).Name("healthcheck-finance-system")

	appRouter := mainRouter.PathPrefix("/company/{company_number}/penalties/late-filing").Subrouter()
	appRouter.HandleFunc("", HandleGetPenalties).Methods(http.MethodGet).Name("get-penalties")
	appRouter.Handle("/payable", CreatePayableResourceHandler(svc)).Methods(http.MethodPost).Name("create-payable")
	appRouter.Use(
		oauth2OnlyInterceptor.OAuth2OnlyAuthenticationIntercept,
		userAuthInterceptor.UserAuthenticationIntercept,
		middleware.CompanyMiddleware,
	)

	// sub router for handling interactions with existing payable resources to apply relevant
	// PayableAuthenticationInterceptor
	existingPayableRouter := appRouter.PathPrefix("/payable/{payable_id}").Subrouter()
	existingPayableRouter.HandleFunc("", HandleGetPayableResource).Name("get-payable").Methods(http.MethodGet)
	existingPayableRouter.HandleFunc("/payment", HandleGetPaymentDetails).Methods(http.MethodGet).Name("get-payment-details")
	existingPayableRouter.Use(payableAuthInterceptor.PayableAuthenticationIntercept)

	// separate router for the patch request so that we can apply the interceptor to it without interfering with
	// other routes
	payResourceRouter := appRouter.PathPrefix("/payable/{payable_id}/payment").Methods(http.MethodPatch).Subrouter()
	payResourceRouter.Use(payableAuthInterceptor.PayableAuthenticationIntercept, authentication.ElevatedPrivilegesInterceptor)
	payResourceRouter.Handle("", PayResourceHandler(payableResourceService, e5Client)).Name("mark-as-paid")

	// Set middleware across all routers and sub routers
	mainRouter.Use(log.Handler)
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
