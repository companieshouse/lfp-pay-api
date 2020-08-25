package transformers

import (
	"fmt"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/constants"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/utils"
)

// PayableResourceRequestToDB will take the input request from the REST call and transform it to a dao ready for
// insertion into the database
func PayableResourceRequestToDB(req *models.PayableRequest) *models.PayableResourceDao {
	transactionsDAO := map[string]models.TransactionDao{}
	for _, tx := range req.Transactions {
		transactionsDAO[tx.TransactionID] = models.TransactionDao{
			Amount:     tx.Amount,
			MadeUpDate: tx.MadeUpDate,
			Type:       tx.Type,
		}
	}

	reference := utils.GenerateReferenceNumber()
	etag, err := utils.GenerateEtag()
	if err != nil {
		log.Error(fmt.Errorf("error generating etag: [%s]", err))
	}
	format := "/company/%s/penalties/late-filing/payable/%s"

	self := fmt.Sprintf(format, req.CompanyNumber, reference)

	paymentLinkFormat := "%s/payment"
	paymentLink := fmt.Sprintf(paymentLinkFormat, self)

	resumeJourneyLinkFormat := "/late-filing-penalty/company/%s/penalty/%s/view-penalties"
	resumeJourneyLink := fmt.Sprintf(resumeJourneyLinkFormat, req.CompanyNumber, req.Transactions[0].TransactionID) // Assumes there is only one transaction

	createdAt := time.Now().Truncate(time.Millisecond)
	dao := &models.PayableResourceDao{
		CompanyNumber: req.CompanyNumber,
		Reference:     reference,
		Data: models.PayableResourceDataDao{
			Etag:         etag,
			Transactions: transactionsDAO,
			Payment: models.PaymentDao{
				Status: constants.Pending.String(),
			},
			CreatedAt: &createdAt,
			CreatedBy: models.CreatedByDao{
				Email:    req.CreatedBy.Email,
				ID:       req.CreatedBy.ID,
				Forename: req.CreatedBy.Forename,
				Surname:  req.CreatedBy.Surname,
			},
			Links: models.PayableResourceLinksDao{
				Self:          self,
				Payment:       paymentLink,
				ResumeJourney: resumeJourneyLink,
			},
		},
	}

	return dao
}

// PayableResourceDaoToCreatedResponse will transform a payable resource dao that has successfully been created into
// a http response entity
func PayableResourceDaoToCreatedResponse(model *models.PayableResourceDao) *models.CreatedPayableResource {
	return &models.CreatedPayableResource{
		ID: model.Reference,
		Links: models.CreatedPayableResourceLinks{
			Self: model.Data.Links.Self,
		},
	}
}

// PayableResourceDBToRequest will take the Dao version of a payable resource and convert to an request version
func PayableResourceDBToRequest(payableDao *models.PayableResourceDao) *models.PayableResource {
	transactions := []models.TransactionItem{}
	for key, val := range payableDao.Data.Transactions {
		tx := models.TransactionItem{
			TransactionID: key,
			Amount:        val.Amount,
			MadeUpDate:    val.MadeUpDate,
			Type:          val.Type,
		}
		transactions = append(transactions, tx)
	}

	payable := models.PayableResource{
		CompanyNumber: payableDao.CompanyNumber,
		Reference:     payableDao.Reference,
		Transactions:  transactions,
		Etag:          payableDao.Data.Etag,
		CreatedAt:     payableDao.Data.CreatedAt,
		CreatedBy:     models.CreatedBy(payableDao.Data.CreatedBy),
		Links:         models.PayableResourceLinks(payableDao.Data.Links),
		Payment:       models.Payment(payableDao.Data.Payment),
	}

	return &payable
}

// PayableResourceToPaymentDetails will create a PaymentDetails resource (for integrating into payment service) from an LFP PayableResource
func PayableResourceToPaymentDetails(payable *models.PayableResource) *models.PaymentDetails {
	costs := []models.Cost{}
	for _, tx := range payable.Transactions {
		cost := models.Cost{
			Amount:                  fmt.Sprintf("%g", tx.Amount),
			AvailablePaymentMethods: []string{"credit-card"},
			ClassOfPayment:          []string{"penalty"},
			Description:             "Late Filing Penalty",
			DescriptionIdentifier:   "late-filing-penalty",
			Kind:                    "cost#cost",
			ResourceKind:            "late-filing-penalty#late-filing-penalty",
			ProductType:             "late-filing-penalty",
		}
		costs = append(costs, cost)
	}

	payment := models.PaymentDetails{
		Description: "Late Filing Penalty",
		Etag:        payable.Etag, // use the same Etag as PayableResource its built from - if PayableResource changes PaymentDetails may change too
		Kind:        "payment-details#payment-details",
		Links: models.PaymentDetailsLinks{
			Self:     payable.Links.Payment, // this is the payment details resource so should use payment link from PayableResource
			Resource: payable.Links.Self,    // PayableResources Self link is the resource this PaymentDetails is paying for
		},
		PaidAt:           payable.Payment.PaidAt,
		PaymentReference: payable.Payment.Reference,
		Status:           payable.Payment.Status,
		CompanyNumber:    payable.CompanyNumber,
		Items:            costs,
	}

	return &payment
}
