package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/companieshouse/chs.go/avro"
	"github.com/companieshouse/chs.go/avro/schema"
	"github.com/companieshouse/chs.go/kafka/producer"
	"github.com/companieshouse/filing-notification-sender/util"
	"github.com/companieshouse/lfp-pay-api-core/models"
	"github.com/companieshouse/lfp-pay-api/config"
)

const lfpReceivedAppID = "lfp-pay-api.late_filing_penalty_received_email"
const lfpFilingDescription = "Late Filing Penalty"
const lfpMessageType = "late_filing_penalty_received_email"

// ProducerTopic is the topic to which the email-send kafka message is sent
const ProducerTopic = "email-send"

// ProducerSchemaName is the schema which will be used to send the email-send kafka message with
const ProducerSchemaName = "email-send"

// SendEmailKafkaMessage sends a kafka message to the email-sender to send an email
func SendEmailKafkaMessage(payableResource models.PayableResource, req *http.Request) error {
	cfg, err := config.Get()
	if err != nil {
		err = fmt.Errorf("error getting config for kafka message production: [%v]", err)
		return err
	}

	// Get a producer
	kafkaProducer, err := producer.New(&producer.Config{Acks: &producer.WaitForAll, BrokerAddrs: cfg.BrokerAddr})
	if err != nil {
		err = fmt.Errorf("error creating kafka producer: [%v]", err)
		return err
	}
	emailSendSchema, err := schema.Get(cfg.SchemaRegistryURL, ProducerSchemaName)
	if err != nil {
		err = fmt.Errorf("error getting schema from schema registry: [%v]", err)
		return err
	}
	producerSchema := &avro.Schema{
		Definition: emailSendSchema,
	}

	// Prepare a message with the avro schema
	message, err := prepareKafkaMessage(*producerSchema, payableResource, req)
	if err != nil {
		err = fmt.Errorf("error preparing kafka message with schema: [%v]", err)
		return err
	}

	// Send the message
	partition, offset, err := kafkaProducer.Send(message)
	if err != nil {
		err = fmt.Errorf("failed to send message in partition: %d at offset %d", partition, offset)
		return err
	}
	return nil
}

// prepareKafkaMessage generates the kafka message that is to be sent
func prepareKafkaMessage(emailSendSchema avro.Schema, payableResource models.PayableResource, req *http.Request) (*producer.Message, error) {
	cfg, err := config.Get()
	if err != nil {
		err = fmt.Errorf("error getting config: [%v]", err)
		return nil, err
	}

	// Access Company Name to be included in the email
	companyName, err := GetCompanyName(payableResource.CompanyNumber, req)
	if err != nil {
		err = fmt.Errorf("error getting company name: [%v]", err)
		return nil, err
	}

	// Access specific transaction that was paid for
	payedTransaction, err := GetTransactionForPenalty(payableResource.CompanyNumber, payableResource.Transactions[0].TransactionID)
	if err != nil {
		err = fmt.Errorf("error getting transaction for LFP: [%v]", err)
		return nil, err
	}

	// Convert madeUpDate and transactionDate to readable format for email
	madeUpDate, err := time.Parse("2006-01-02", payedTransaction.MadeUpDate)
	if err != nil {
		err = fmt.Errorf("error parsing made up date: [%v]", err)
		return nil, err
	}
	transactionDate, err := time.Parse("2006-01-02", payedTransaction.TransactionDate)
	if err != nil {
		err = fmt.Errorf("error parsing penalty date: [%v]", err)
		return nil, err
	}

	// Set dataField to be used in the avro schema.
	dataFieldMessage := models.DataField{
		PayableResource:   payableResource,
		TransactionID:     payableResource.Transactions[0].TransactionID,
		MadeUpDate:        madeUpDate.Format("2 January 2006"),
		TransactionDate:   transactionDate.Format("2 January 2006"),
		Amount:            fmt.Sprintf("%g", payedTransaction.OriginalAmount),
		CompanyName:       companyName,
		FilingDescription: lfpFilingDescription,
		To:                payableResource.CreatedBy.Email,
		Subject:           fmt.Sprintf("Confirmation of your Companies House penalty payment"),
		CHSURL:            cfg.CHSURL,
	}

	dataBytes, err := json.Marshal(dataFieldMessage)
	if err != nil {
		err = fmt.Errorf("error marshalling dataFieldMessage: [%v]", err)
		return nil, err
	}

	messageID := "<" + payableResource.Reference + "." + strconv.Itoa(util.Random(0, 100000)) + "@companieshouse.gov.uk>"

	emailSendMessage := models.EmailSend{
		AppID:        lfpReceivedAppID,
		MessageID:    messageID,
		MessageType:  lfpMessageType,
		Data:         string(dataBytes),
		EmailAddress: payableResource.CreatedBy.Email,
		CreatedAt:    time.Now().String(),
	}

	messageBytes, err := emailSendSchema.Marshal(emailSendMessage)
	if err != nil {
		err = fmt.Errorf("error marshalling email send message: [%v]", err)
		return nil, err
	}

	producerMessage := &producer.Message{
		Value: messageBytes,
		Topic: ProducerTopic,
	}
	return producerMessage, nil
}
