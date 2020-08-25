package validators

import (
	"fmt"
	"os"
	"testing"

	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

func createE5Response(txType string, isPaid bool, isDCA bool, outstandingAmount int) string {
	isPaidString := "true"
	if !isPaid {
		isPaidString = "false"
	}

	accountStatus := ""
	if isDCA {
		accountStatus = "DCA"
	}

	return fmt.Sprintf(`
		{
			"page": {
			"size": 4,
				"totalElements": 4,
				"totalPages": 1,
				"number": 0
			},
			"data": [
			{
				"companyCode": "LP",
				"ledgerCode": "EW",
				"customerCode": "10000024",
				"transactionReference": "00378420",
				"transactionDate": "2017-11-28",
				"madeUpDate": "2017-02-28",
				"amount": 150,
				"outstandingAmount": %d,
				"isPaid": %s,
				"transactionType": "%s",
				"transactionSubType": "EU",
				"typeDescription": "Penalty Ltd Wel & Eng <=1m     LTDWA    ",
				"dueDate": "2017-12-12",
				"accountStatus": "%s"
			}
		]
	}
	`, outstandingAmount, isPaidString, txType, accountStatus)
}

const multiplePenalties = `
{
  "page" : {
    "size" : 2,
    "totalElements" : 2,
    "totalPages" : 1,
    "number" : 0
  },
  "data" : [ {
    "companyCode" : "LP",
    "ledgerCode" : "EW",
    "customerCode" : "05838070",
    "transactionReference" : "00482774",
    "transactionDate" : "2018-04-30",
    "madeUpDate" : "2017-06-30",
    "amount" : 150,
    "outstandingAmount" : 150,
    "isPaid" : false,
    "transactionType" : "1",
    "transactionSubType" : "EU",
    "typeDescription" : "Penalty Ltd Wel & Eng <=1m     LTDWA    ",
    "dueDate" : "2018-05-14"
  }, {
    "companyCode" : "LP",
    "ledgerCode" : "EW",
    "customerCode" : "05838070",
    "transactionReference" : "00556352",
    "transactionDate" : "2019-06-27",
    "madeUpDate" : "2018-06-30",
    "amount" : 750,
    "outstandingAmount" : 750,
    "isPaid" : false,
    "transactionType" : "1",
    "transactionSubType" : "EJ",
    "typeDescription" : "Double DBL LTD E&W>1<3 MNTH   DLTWB     ",
    "dueDate" : "2019-07-11"
  } ]
}
`

func TestUnitPayableTransactions(t *testing.T) {
	os.Chdir("..")
	cfg, _ := config.Get()
	cfg.E5APIURL = "https://e5"
	cfg.E5Username = "SYSTEM"

	url := "https://e5/arTransactions/10000024?ADV_userName=SYSTEM&companyCode=LP&fromDate=1990-01-01"

	Convey("error is returned when transaction does not exist", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		e5Response := createE5Response("1", false, false, 150)
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, e5Response))

		txs := []models.TransactionItem{
			models.TransactionItem{TransactionID: "123"},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(validTxs, ShouldBeNil)
		So(err, ShouldBeError, ErrTransactionDoesNotExist)
	})

	Convey("error is returned if type is not a penalty", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		e5Response := createE5Response("2", false, false, 150)
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, e5Response))

		txs := []models.TransactionItem{
			models.TransactionItem{TransactionID: "00378420"},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(validTxs, ShouldBeNil)
		So(err, ShouldBeError, ErrTransactionNotPayable)
	})

	Convey("error is returned if transaction is already paid", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		e5Response := createE5Response("1", true, false, 150)
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, e5Response))

		txs := []models.TransactionItem{
			models.TransactionItem{TransactionID: "00378420"},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(validTxs, ShouldBeNil)
		So(err, ShouldBeError, ErrTransactionIsPaid)
	})

	Convey("type and made up date are taken from E5", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		e5Response := createE5Response("1", false, false, 150)
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, e5Response))

		txs := []models.TransactionItem{
			models.TransactionItem{TransactionID: "00378420", Amount: 150},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(err, ShouldBeNil)
		So(validTxs[0].MadeUpDate, ShouldEqual, "2017-02-28")
		So(validTxs[0].Type, ShouldEqual, "penalty")
		So(validTxs[0].Amount, ShouldEqual, 150)
		So(validTxs[0].IsPaid, ShouldEqual, false)
		So(validTxs[0].IsDCA, ShouldEqual, false)
	})

	Convey("error is returned if trying to part pay a transaction", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		e5Response := createE5Response("1", false, false, 150)
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, e5Response))

		txs := []models.TransactionItem{
			models.TransactionItem{TransactionID: "00378420", Amount: 100},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(validTxs, ShouldBeNil)
		So(err, ShouldBeError, ErrTransactionAmountMismatch)
	})

	Convey("error when company has multiple penalties", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, multiplePenalties))

		txs := []models.TransactionItem{
			{TransactionID: "00378420", Amount: 100},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(validTxs, ShouldBeNil)
		So(err, ShouldBeError, ErrMultiplePenalties)
	})

	Convey("error is returned if transaction is in DCA status", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		e5Response := createE5Response("1", false, true, 150)
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, e5Response))

		txs := []models.TransactionItem{
			{TransactionID: "00378420", Amount: 150},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(validTxs, ShouldBeNil)
		So(err, ShouldBeError, ErrTransactionDCA)
	})

	Convey("error is returned if penalty is part paid", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		e5Response := createE5Response("1", false, true, 50)
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, e5Response))

		txs := []models.TransactionItem{
			{TransactionID: "00378420", Amount: 50},
		}

		validTxs, err := TransactionsArePayable("10000024", txs)

		So(validTxs, ShouldBeNil)
		So(err, ShouldBeError, ErrTransactionIsPartPaid)
	})
}
