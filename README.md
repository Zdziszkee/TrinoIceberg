This is my implementation of the Swift Codes Parquet Data Lake using Tring + Iceberg + Hive Metastore for efficient read queries.
The credentials, password are sample ones pushed for the ease of running the project. They are not meant for production use.
Project can be run through the following steps:
1. Clone the repository
2. Install dependencies go mod tidy
3. Run the project docker-compose up -d
4. Wait for the containers to start up, you can view state of them through docker-compose logs -f
5. Endpoints should be avaiable on http://localhost:8081/v1/swiftCodes

Example usages:
GET http://127.0.0.1:8081/v1/swiftCodes/BSZLPLP1XXX
GET http://127.0.0.1:8081/v1/swiftCodes/country/MT
POST http://127.0.0.1:8081/v1/swiftCodes
DELETE http://127.0.0.1:8081/v1/swiftCodes/BSZLPLP1XXXA


Access to trino container for running queries:
-> docker exec -it swift-codes-trino-1  bash
-> trino
-> trino:default_schema> use swift_catalog.default_schema;
-> select * FROM swift_banks limit 10;

Running tests:
➜  swift-codes git:(main) export PATH=$PATH:$HOME/go/bin
-> ginkgo -r
