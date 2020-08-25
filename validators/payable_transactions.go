package validators

import (
	"errors"
	"fmt"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/service"
)

var (
	ErrTransactionDoesNotExist   = errors.New("invalid transaction")
	ErrTransactionNotPayable     = errors.New("you cannot pay for this type of transaction")
	ErrTransactionDCA            = errors.New("the transaction is with a debt collecting agency")
	ErrTransactionIsPaid         = errors.New("this transaction is already paid")
	ErrTransactionIsPartPaid     = errors.New("the transaction is already part paid")
	ErrTransactionAmountMismatch = errors.New("you can only pay off the full amount of the transaction")
	ErrMultiplePenalties         = errors.New("the company has more than one outstanding penalty")
)

// TransactionsArePayable validator will verify the transaction in a request do exist for the company. It will also update the
// type and made up date fields to match what is in E5.
func TransactionsArePayable(companyNumber string, txs []models.TransactionItem) ([]models.TransactionItem, error) {
	response, _, err := service.GetPenalties(companyNumber)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// rule for first release, a company must only have one outstanding penalty that they can pay for
	payablePenaltyCount := 0

	// create and cache a map of the transaction to make it easier to lookup each one
	itemMap := map[string]models.TransactionListItem{}
	for _, tx := range response.Items {
		itemMap[tx.ID] = tx
		if !tx.IsPaid && tx.Type == "penalty" {
			payablePenaltyCount++
		}
	}

	var validTxs []models.TransactionItem

	// for the first release, the company must only have one outstanding penalty
	if payablePenaltyCount > 1 {
		log.Info("company has more than one outstanding penalty", log.Data{
			"company_number": companyNumber,
			"penalty_count":  payablePenaltyCount,
		})
		return validTxs, ErrMultiplePenalties
	}

	for _, t := range txs {
		val, ok := itemMap[t.TransactionID]
		data := map[string]interface{}{
			"transaction_ref": t.TransactionID,
			"company_number":  companyNumber,
		}
		if !ok {
			log.Info("disallowing paying for a transaction that does not exist in E5", data)
			return nil, ErrTransactionDoesNotExist
		}

		if val.IsPartPaid() {
			log.Info("the penalty that is trying to be paid is already part paid", data)
			return nil, ErrTransactionIsPartPaid
		}

		if val.IsPaid {
			log.Info("disallowing paying for a transaction that is already paid", data)
			return nil, ErrTransactionIsPaid
		}

		if val.Type != "penalty" {
			log.Info("disallowing paying for a transaction that is not a penalty", data)
			return nil, ErrTransactionNotPayable
		}

		if val.Outstanding != t.Amount {
			data["attempted_amount"] = fmt.Sprintf("%f", t.Amount)
			data["outstanding_amount"] = fmt.Sprintf("%f", val.Outstanding)
			log.Info("disallowing paying for transaction as attempting to pay off partial balance", data)
			return nil, ErrTransactionAmountMismatch
		}

		if val.IsDCA {
			log.Info("the transaction that is trying to be paid is with a debt collecting agency", data)
			return nil, ErrTransactionDCA
		}

		validTx := models.TransactionItem{
			TransactionID: t.TransactionID,
			Amount:        t.Amount,
			Type:          val.Type,
			MadeUpDate:    val.MadeUpDate,
			IsDCA:         val.IsDCA,
			IsPaid:        val.IsPaid,
		}
		validTxs = append(validTxs, validTx)
	}

	return validTxs, nil
}
