package service

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api-core/validators"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/e5"
	"github.com/companieshouse/lfp-pay-api/utils"

	"gopkg.in/yaml.v2"
)

// TransactionType Enum Type
type TransactionType int

// Enumeration containing all possible types when mapping e5 transactions
const (
	Penalty TransactionType = 1 + iota
	Other
)

// String representation of transaction types
var transactionTypes = [...]string{
	"penalty",
	"other",
}

func (transactionType TransactionType) String() string {
	return transactionTypes[transactionType-1]
}

// GetPenalties is a function that:
// 1. makes a request to e5 to get a list of transactions for the specified company
// 2. takes the results of this request and maps them to a format that the lfp-pay-web can consume
func GetPenalties(companyNumber string) (*models.TransactionListResponse, ResponseType, error) {
	cfg, err := config.Get()
	if err != nil {
		return nil, Error, nil
	}
	client := e5.NewClient(cfg.E5Username, cfg.E5APIURL)
	e5Response, err := client.GetTransactions(&e5.GetTransactionsInput{CompanyNumber: companyNumber, CompanyCode: "LP"})

	if err != nil {
		log.Error(fmt.Errorf("error getting transaction list: [%v]", err))
		return nil, Error, err
	}

	// Generate the CH preferred format of the results i.e. classify the transactions into payable "penalty" types or
	// non-payable "other" types
	generatedTransactionListFromE5Response, err := generateTransactionListFromE5Response(e5Response)
	if err != nil {
		err = fmt.Errorf("error generating transaction list from the e5 response: [%v]", err)
		log.Error(err)
		return nil, Error, err
	}

	log.Info("Completed GetPenalties request and mapped to CH LFP transactions", log.Data{"company_number": companyNumber})
	return generatedTransactionListFromE5Response, Success, nil
}

// GetTransactionForPenalty returns a single, specified, transaction from e5 for a specific company
func GetTransactionForPenalty(companyNumber, penaltyNumber string) (*models.TransactionListItem, error) {
	response, _, err := GetPenalties(companyNumber)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, transaction := range response.Items {
		if transaction.ID == penaltyNumber {
			return &transaction, nil
		}
	}

	return nil, fmt.Errorf("cannot find lfp transaction for penalty number [%v]", penaltyNumber)
}

func generateTransactionListFromE5Response(e5Response *e5.GetTransactionsResponse) (*models.TransactionListResponse, error) {
	// Next, map results to a format that can be used by LFP web
	payableTransactionList := models.TransactionListResponse{}
	etag, err := utils.GenerateEtag()
	if err != nil {
		err = fmt.Errorf("error generating etag: [%v]", err)
		log.Error(err)
		return nil, err
	}

	payableTransactionList.Etag = etag
	payableTransactionList.TotalResults = e5Response.Page.TotalElements
	// Each transaction needs to be checked and identified as a 'penalty' or 'other'. This allows lfp-web to determine
	// which transactions are payable. This is done using a yaml file to map payable transactions
	yamlFile, err := ioutil.ReadFile("assets/penalty_types.yml")
	if err != nil {
		err = fmt.Errorf("error reading penalty types yaml file: [%v]", err)
		log.Error(err)
		return nil, err
	}

	allowedTransactions := models.AllowedTransactionMap{}
	err = yaml.Unmarshal(yamlFile, &allowedTransactions)
	if err != nil {
		err = fmt.Errorf("error unmarshalling yaml file: [%v]", err)
		log.Error(err)
		return nil, err
	}

	// Loop through e5 response and construct CH resources
	for _, e5Transaction := range e5Response.Transactions {
		listItem := models.TransactionListItem{}
		listItem.ID = e5Transaction.TransactionReference
		listItem.Etag, err = utils.GenerateEtag()
		if err != nil {
			err = fmt.Errorf("error generating etag: [%v]", err)
			log.Error(err)
			return nil, err
		}
		listItem.IsPaid = e5Transaction.IsPaid
		listItem.Kind = "late-filing-penalty#late-filing-penalty"
		listItem.IsDCA = e5Transaction.AccountStatus == "DCA"
		listItem.DueDate = e5Transaction.DueDate
		listItem.MadeUpDate = e5Transaction.MadeUpDate
		listItem.TransactionDate = e5Transaction.TransactionDate
		listItem.OriginalAmount = e5Transaction.Amount
		listItem.Outstanding = e5Transaction.OutstandingAmount
		// Check if the transaction is allowed and set to 'penalty' if it is
		if _, ok := allowedTransactions.Types[e5Transaction.TransactionType][e5Transaction.TransactionSubType]; ok {
			listItem.Type = Penalty.String()
		} else {
			listItem.Type = Other.String()
		}
		payableTransactionList.Items = append(payableTransactionList.Items, listItem)
	}
	return &payableTransactionList, nil
}

// MarkTransactionsAsPaid will update the transactions in E5 as paid.
// resource - is the payable resource from the db representing the late filing penalty(ies)
// payment - is the information about the payment session
func MarkTransactionsAsPaid(svc *PayableResourceService, client *e5.Client, resource models.PayableResource, payment validators.PaymentInformation) error {
	amountPaid, err := strconv.ParseFloat(payment.Amount, 32)
	if err != nil {
		log.Error(err, log.Data{"payment_id": payment.Reference, "amount": payment.Amount})
		return err
	}

	var transactions []*e5.CreatePaymentTransaction

	for _, t := range resource.Transactions {
		transactions = append(transactions, &e5.CreatePaymentTransaction{
			Reference: t.TransactionID,
			Value:     t.Amount,
		})
	}

	// this will be used for the PUON value in E5. it is referred to as paymentId in their spec. X is prefixed to it
	// so that it doesn't clash with other PUON's from different sources when finance produce their reports - namely
	// ones that begin with 'LP' which signify penalties that have been paid outside of the digital service.
	paymentID := "X" + payment.PaymentID

	// three http requests are needed to mark a transactions as paid. The process is 1) create the payment, 2) authorise
	// the payments and finally 3) confirm the payment. if anyone of these fails, the company account will be locked in
	// E5. Finance have confirmed that it is better to keep these locked as a cleanup process will happen naturally in
	// the working day.
	err = client.CreatePayment(&e5.CreatePaymentInput{
		CompanyCode:   "LP",
		CompanyNumber: resource.CompanyNumber,
		PaymentID:     paymentID,
		TotalValue:    amountPaid,
		Transactions:  transactions,
	})

	if err != nil {
		if svcErr := svc.RecordE5CommandError(resource, e5.CreateAction); svcErr != nil {
			log.Error(svcErr, log.Data{"payment_id": payment.PaymentID, "lfp_reference": resource.Reference})
			return err
		}
		logE5Error("failed to create payment in E5", err, resource, payment)
		return err
	}

	err = client.AuthorisePayment(&e5.AuthorisePaymentInput{
		CompanyCode:   "LP",
		PaymentID:     paymentID,
		CardReference: payment.ExternalPaymentID,
		CardType:      payment.CardType,
		Email:         payment.CreatedBy,
	})

	if err != nil {
		if svcErr := svc.RecordE5CommandError(resource, e5.AuthoriseAction); svcErr != nil {
			log.Error(svcErr, log.Data{"payment_id": payment.PaymentID, "lfp_reference": resource.Reference})
			return err
		}
		logE5Error("failed to authorise payment in E5", err, resource, payment)
		return err
	}

	err = client.ConfirmPayment(&e5.PaymentActionInput{
		CompanyCode: "LP",
		PaymentID:   paymentID,
	})

	if err != nil {
		if svcErr := svc.RecordE5CommandError(resource, e5.ConfirmAction); svcErr != nil {
			log.Error(svcErr, log.Data{"payment_id": payment.PaymentID, "lfp_reference": resource.Reference})
			return err
		}
		logE5Error("failed to confirm payment in E5", err, resource, payment)
		return err
	}

	log.Info("marked LFP transaction(s) as paid in E5", log.Data{
		"lfp_reference": resource.Reference,
		"payment_id":    payment.PaymentID,
		"e5_puon":       payment.PaymentID,
	})

	return nil
}

func logE5Error(message string, originalError error, resource models.PayableResource, payment validators.PaymentInformation) {
	log.Error(errors.New(message), log.Data{
		"lfp_reference": resource.Reference,
		"payment_id":    payment.PaymentID,
		"amount":        payment.Amount,
		"error":         originalError,
	})
}
