# LFP PAY API

LFP PAY API which provides an interface for Creating, Getting, and Patching LFPs.

## Requirements
In order to run this API locally you will need to install the following:

- [Go](https://golang.org/doc/install)
- [Git](https://git-scm.com/downloads)

## Getting Started
1. Clone this repository: `go get github.com/companieshouse/lfp-pay-api`
1. Build the executable: `make build`

## Configuration
Variable                         | Default | Description
:--------------------------------|:-------:|:-----------
`E5_API_URL`                     | `-`     | E5 API Address
`E5_USERNAME`                    | `-`     | E5 API Username
`BIND_ADDR`                      | `-`     | The host:port to bind to
`MONGODB_URL`                    | `-`     | The mongo db connection string
`LFP_MONGODB_DATABASE`           | `-`     | The database name to connect to e.g. `late_filing_penalties`
`LFP_MONGODB_COLLECTION`         | `-`     | The collection name e.g. `payable_resources`
`KAFKA_BROKER_ADDR`              | `_`     | Kafka Broker Address
`SCHEMA_REGISTRY_URL`            | `_`     | Schema Registry URL
`CHS_URL`                        | `_`     | CHS URL
`WEEKLY_MAINTENANCE_START_TIME`  | `_`     | Start time of weekly maintenance e.g. `0700`
`WEEKLY_MAINTENANCE_END_TIME`    | `_`     | End time of weekly maintenance e.g. `0730`
`WEEKLY_MAINTENANCE_DAY`         | `_`     | Day of weekly maintenance e.g. `0` (zero for Sunday)
`PLANNED_MAINTENANCE_START_TIME` | `_`     | Start time and date of planned maintenance e.g. `01 Jan 19 15:04 BST`
`PLANNED_MAINTENANCE_END_TIME`   | `_`     | End time and date of planned maintenance e.g. `31 Jan 19 16:59 BST`

## Endpoints
Method    | Path                                                                   | Description
:---------|:-----------------------------------------------------------------------|:-----------
**GET**   | `/healthcheck`                                                         | Standard healthcheck endpoint
**GET**   | `/healthcheck/finance-system`                                          | Healthcheck endpoint to check whether the finance system is available
**GET**   | `/company/{company_number}/penalties/late-filing`                      | List the Late Filing Penalties for a company
**POST**  | `/company/{company_number}/penalties/late-filing/payable`              | Create a payable penalty resource
**GET**   | `/company/{company_number}/penalties/late-filing/payable/{id}`         | Get a payable resource
**GET**   | `/company/{company_number}/penalties/late-filing/payable/{id}/payment` | List the cost items related to the penalty resource
**PATCH** | `/company/{company_number}/penalties/late-filing/payable/{id}/payment` | Mark the resource as paid

## External Finance Systems
The only external finance system currently supported is E5.

