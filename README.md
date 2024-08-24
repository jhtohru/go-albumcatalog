# go-album-catalog

**go-album-catalog** is a RESTful API server that CRUDs music albums.

The application HTTP endpoints are described at the [docs/oas.yaml](docs/oas.yaml) Open API Specification file.

This is a personal project aimed to practice and study the development of HTTP services following [Mat Ryer](https://github.com/matryer)'s post [How I write HTTP services in Go after 13 years](https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/).

## Running the application

Since the application stores data in a Postgres database, it requires the `DSN` environment variable to be set with a running Postgres instance DSN and it will panic if not set.

```console
$ DSN=<POSTGRES_DSN> go run ./cmd/catalog/main.go
```

### Environment variables

The server hostname can be defined setting the `SERVER_HOST` environment variable.
The server port can be defined setting the `SERVER_PORT` environment variable, and defaults to **8080** if not set.
If the `MIGRATE_DB` environment variable is set as `"true"`, the database is migrated before the application starts.

## Testing the source code

The application source code is covered by both unit and integration tests.
When the integration tests run, a container running a Postgres instance is started automatically.

### Environment variables

To run the integration tests with a custom Postgres instance, set the `POSTGRES_ADDR` environment variable with the Postgres instance address.
The following environment variables can be used to allow the integration tests to connect to the custom Postgres instance:
| Env var name | Default value |
| - | - |
| `POSTGRES_USER` | `"postgres"` |
| `POSTGRES_PASSWORD` | `"password"` |
| `POSTGRES_DEFAULT_DB` | `"postgres"` |

## Migrating the database

[Goose](https://github.com/pressly/goose) is used to migrate the database. Run the following command to migrate the Postgres main database, which will be used by the application. Replace `<DSN>` with the DSN of the Postgres database to be migrated.

```console
$ goose -dir ./migrations/ postgres <DSN> up
```

## Local development

Having local Postgres instance can help the development because it enables starting the application locally and also makes the integration tests more responsive.

Starting a local Postgres instance:
```console
$ docker-compose -f dev/docker-compose.yaml up -d
```

Starting the application connected to the local Postgres instance:
```console
$ DSN='postgresql://127.0.0.1:5432/postgres?user=postgres&password=password&sslmode=disable' MIGRATE_DB='true' go run cmd/catalog/main.go
```

Running the integration tests connected to the local Postgres instance:
```console
$ POSTGRES_ADDR='127.0.0.1:5432' go test ./...
```
