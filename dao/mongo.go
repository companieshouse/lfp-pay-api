package dao

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/e5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func getMongoClient(mongoDBURL string) *mongo.Client {
	if client != nil {
		return client
	}

	ctx := context.Background()

	clientOptions := options.Client().ApplyURI(mongoDBURL)
	client, err := mongo.Connect(ctx, clientOptions)

	// assume the caller of this func cannot handle the case where there is no database connection so the prog must
	// crash here as the service cannot continue.
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// check we can connect to the mongodb instance. failure here should result in a crash.
	pingContext, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer cancel()
	err = client.Ping(pingContext, nil)
	if err != nil {
		log.Error(errors.New("ping to mongodb timed out. please check the connection to mongodb and that it is running"))
		os.Exit(1)
	}

	log.Info("connected to mongodb successfully")

	return client
}

// MongoDatabaseInterface is an interface that describes the mongodb driver
type MongoDatabaseInterface interface {
	Collection(name string, opts ...*options.CollectionOptions) *mongo.Collection
}

func getMongoDatabase(mongoDBURL, databaseName string) MongoDatabaseInterface {
	return getMongoClient(mongoDBURL).Database(databaseName)
}

// MongoService is an implementation of the Service interface using MongoDB as the backend driver.
type MongoService struct {
	db             MongoDatabaseInterface
	CollectionName string
}

// SaveE5Error will update the resource by flagging an error in e5 for a particular action
func (m *MongoService) SaveE5Error(companyNumber, reference string, action e5.Action) error {
	dao, err := m.GetPayableResource(companyNumber, reference)
	if err != nil {
		log.Error(err, log.Data{"company_number": companyNumber, "lfp_reference": reference})
		return err
	}

	filter := bson.M{"_id": dao.ID}
	update := bson.D{
		{
			"$set", bson.D{
				{"e5_command_error", string(action)},
			},
		},
	}

	collection := m.db.Collection(m.CollectionName)

	log.Debug("updating e5 command error in mongo document", log.Data{"_id": dao.ID})

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err, log.Data{"_id": dao.ID, "company_number": dao.CompanyNumber, "reference": dao.Reference})
		return err
	}

	return nil
}

// CreatePayableResource will store the payable request into the database
func (m *MongoService) CreatePayableResource(dao *models.PayableResourceDao) error {

	dao.ID = primitive.NewObjectID()

	collection := m.db.Collection(m.CollectionName)
	_, err := collection.InsertOne(context.Background(), dao)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// GetPayableResource gets the payable request from the database
func (m *MongoService) GetPayableResource(companyNumber, reference string) (*models.PayableResourceDao, error) {
	var resource models.PayableResourceDao

	collection := m.db.Collection(m.CollectionName)
	dbResource := collection.FindOne(context.Background(), bson.M{"reference": reference, "company_number": companyNumber})

	err := dbResource.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Debug("no payable resource found", log.Data{"company_number": companyNumber, "reference": reference})
			return nil, nil
		}
		log.Error(err, log.Data{"company_number": companyNumber, "reference": reference})
		return nil, err
	}

	err = dbResource.Decode(&resource)

	if err != nil {
		log.Error(err, log.Data{"company_number": companyNumber, "reference": reference})
		return nil, err
	}

	return &resource, nil
}

// UpdatePaymentDetails will save the document back to Mongo
func (m *MongoService) UpdatePaymentDetails(dao *models.PayableResourceDao) error {
	filter := bson.M{"_id": dao.ID}

	update := bson.D{
		{
			"$set", bson.D{
				{"data.payment.status", dao.Data.Payment.Status},
				{"data.payment.reference", dao.Data.Payment.Reference},
				{"data.payment.paid_at", dao.Data.Payment.PaidAt},
				{"data.payment.amount", dao.Data.Payment.Amount},
			},
		},
	}

	collection := m.db.Collection(m.CollectionName)

	log.Debug("updating payment details in mongo document", log.Data{"_id": dao.ID})

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err, log.Data{"_id": dao.ID, "company_number": dao.CompanyNumber, "reference": dao.Reference})
		return err
	}

	log.Debug("updated payment details in mongo document", log.Data{"_id": dao.ID})

	return nil
}

// Shutdown is a hook that can be used to clean up db resources
func (m *MongoService) Shutdown() {
	if client != nil {
		err := client.Disconnect(context.Background())
		if err != nil {
			log.Error(err)
			return
		}
		log.Info("disconnected from mongodb successfully")
	}
}
