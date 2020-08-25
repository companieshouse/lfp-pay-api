package middleware

import (
	"context"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/utils"
	"github.com/gorilla/mux"
)

// CompanyMiddleware will intercept the company number in the path and stick it into the context
func CompanyMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		companyNumber, err := utils.GetCompanyNumberFromVars(vars)

		// if there is no company number in the path, then it is likely that this middleware has been used on an
		// incorrect route.
		if err != nil {
			log.ErrorR(r, err)
		}
		ctx := context.WithValue(r.Context(), config.CompanyNumber, companyNumber)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
