This is my implementation of the Swift Codes Parquet Data Lake using Tring + Iceberg + Hive Metastore for efficient read queries.
The credentials, password are sample ones pushed for the ease of running the project. They are not meant for production use.
Project can be run through the following steps:
1. Clone the repository
2. Install dependencies go mod tidy
3. Run the project docker-compose up -d
4. Wait for the containers to start up, you can view state of them through docker-compose logs -f
5. Endpoints should be avaiable on http://localhost:8081/v1/swiftCodes/
