package e5

// GetTransactionsInput is the struct used to query transactions by company number
type GetTransactionsInput struct {
	CompanyCode   string `validate:"required"`
	CompanyNumber string `validate:"required"`
	PageNumber    int
}

// GetTransactionsResponse returns the output of a get request for company transactions
type GetTransactionsResponse struct {
	Page         Page          `json:"page"`
	Transactions []Transaction `json:"data"`
}

// Transaction is a representation of a transaction item in E5
type Transaction struct {
	CompanyCode          string  `json:"companyCode"`
	LedgerCode           string  `json:"ledgerCode"`
	CustomerCode         string  `json:"customerCode"`
	TransactionReference string  `json:"transactionReference"`
	TransactionDate      string  `json:"transactionDate"`
	MadeUpDate           string  `json:"madeUpDate"`
	Amount               float64 `json:"amount"`
	OutstandingAmount    float64 `json:"outstandingAmount"`
	IsPaid               bool    `json:"isPaid"`
	TransactionType      string  `json:"transactionType"`
	TransactionSubType   string  `json:"transactionSubType"`
	TypeDescription      string  `json:"typeDescription"`
	DueDate              string  `json:"dueDate"`
	AccountStatus        string  `json:"accountStatus"`
}

// Page is a representation of a Page data block in part of e5 GET request
type Page struct {
	Size          int `json:"size"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
	Number        int `json:"number"`
}

// CreatePaymentInput is the struct needed to send a create payment request to the Client API
type CreatePaymentInput struct {
	CompanyCode   string                      `json:"companyCode" validate:"required"`
	CompanyNumber string                      `json:"customerCode" validate:"required"`
	PaymentID     string                      `json:"paymentId" validate:"required"`
	TotalValue    float64                     `json:"paymentValue" validate:"required"`
	Transactions  []*CreatePaymentTransaction `json:"transactions" validate:"required"`
}

// CreatePaymentTransaction is the struct to define the transactions you want to pay for
type CreatePaymentTransaction struct {
	Reference string  `json:"transactionReference" validate:"required"`
	Value     float64 `json:"allocationValue" validate:"required"`
}

// AuthorisePaymentInput is the struct to authorise payment
type AuthorisePaymentInput struct {
	CompanyCode         string `json:"companyCode" validate:"required"`
	PaymentID           string `json:"paymentId" validate:"required"`
	CardReference       string `json:"paymentCardReference"`
	AuthorisationNumber string `json:"authorisationNumber"`
	CardType            string `json:"cardType"`
	Email               string `json:"emailAddress" validate:"required,email"`
}

// PaymentActionInput input is the struct used for the confirm, reject and timeout actions
type PaymentActionInput struct {
	CompanyCode string `json:"companyCode" validate:"required"`
	PaymentID   string `json:"paymentId" validate:"required"`
}

// PaymentActionResponse is the return value of a successful request to create a payment
type PaymentActionResponse struct {
	Success      bool
	ErrorMessage string
}

// e5ApiErrorResponse is the generic struct used to unmarshal the body of responses that have errored
type apiErrorResponse struct {
	Code         int    `json:"httpStatusCode"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	MessageCode  string `json:"messageCode,omitempty"`
	Message      string `json:"message"`
	DebugMessage string `json:"debugMessage"`
	SubErrors    []struct {
		Object        string `json:"object"`
		Field         string `json:"field"`
		RejectedValue string `json:"rejectedValue"`
		Message       string `json:"message"`
	} `json:"subErrors,omitempty"`
}

// SubErrorMap converts the sub error struct into a map
func (e *apiErrorResponse) SubErrorMap() []map[string]string {
	subErrors := make([]map[string]string, 0, len(e.SubErrors))

	for _, sub := range e.SubErrors {
		subErrors = append(subErrors, map[string]string{
			"field":          sub.Field,
			"rejected_value": sub.RejectedValue,
			"message":        sub.Message,
		})
	}

	return subErrors
}
