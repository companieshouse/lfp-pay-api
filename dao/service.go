package dao

import (
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"
	"github.com/companieshouse/lfp-pay-api/e5"
)

// Service interface declares how to interact with the persistence layer regardless of underlying technology
type Service interface {
	// CreatePayableResource will persist a newly created resource
	CreatePayableResource(dao *models.PayableResourceDao) error
	// GetPayableResource will find a single payable resource with the given companyNumber and reference
	GetPayableResource(companyNumber, reference string) (*models.PayableResourceDao, error)
	// UpdatePaymentDetails will update the resource with changed values
	UpdatePaymentDetails(dao *models.PayableResourceDao) error
	// SaveE5Error stored which command to E5 failed e.g. create, authorise or confirm
	SaveE5Error(companyNumber, reference string, action e5.Action) error
	// Shutdown can be called to clean up any open resources that the service may be holding on to.
	Shutdown()
}

// NewDAOService will create a new instance of the Service interface. All details about its implementation and the
// database driver will be hidden from outside of this package
func NewDAOService(cfg *config.Config) Service {
	database := getMongoDatabase(cfg.MongoDBURL, cfg.Database)
	return &MongoService{
		db:             database,
		CollectionName: cfg.MongoCollection,
	}
}
