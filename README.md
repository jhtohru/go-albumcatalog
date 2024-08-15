# albumcatalog

**albumcatalog** is a RESTful API server that CRUDs music albums.

The application HTTP endpoints are described at the [docs/oas.yaml](docs/oas.yaml) Open API Specification file.

This is a personal project aimed to practice and study the development of HTTP services following [Mat Ryer](https://github.com/matryer)'s post [How I write HTTP services in Go after 13 years](https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/).

## Running the application

### Starting a Postgres instance

Since the application uses Postgres as its data storage, it expects a running Postgres instance to start.

You can spin up a Docker container running a Postgres instance with the following command:

```console
go-albumcatalog$ docker-compose -f ./dev/docker-compose.yaml up -d
```

### Setting the required environment variables

Once a Postgres instance is running, you may set the required environment variables to your current shell so you will not need to set them when running integration tests or starting the application.
Run the following commands to set the required enviroment variables with the values defined in the [dev/docker-compose.yaml](dev/docker-compose.yaml) file.

```console
go-albumcatalog$ export POSTGRES_ADDR=127.0.0.1:5432
go-albumcatalog$ export DSN=postgresql://postgres@${POSTGRES_ADDR}/postgres?sslmode=disable
```

### Testing the source code
To run the tests, both unit and database integration tests, run the following command:

```console
go-albumcatalog$ go test ./...
```
If the `POSTGRES_ADDR` environment variable is not set, the integration tests will fail.

### Migrating the database

[Goose](https://github.com/pressly/goose) is used to migrate the database. Run the following command to migrate the Postgres main database, which will be used by the application. If `$DSN` is not set, replace it with the DSN of the Postgres database to which the application will connect.

```console
go-albumcatalog$ goose -dir ./migrations/ postgres ${DSN} up
```

### Starting the application
To start the application, run the following command:

```console
go-albumcatalog$ go run ./cmd/albumcatalog/main.go
```

If the `DSN` environment variable is not set, the application will not start and exit with a non-zero value.

If no errors occur, the application will listen to 127.0.0.1:8080. To customize the server hostname and port, set the `SERVER_HOST` and `SERVER_PORT` environment variables.

### Compiling the application
To compile the application, run the following command:

```console
go-albumcatalog$ go build -o albumcatalog ./cmd/albumcatalog/
```
