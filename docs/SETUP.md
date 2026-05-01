### Manual Setup

Requires Go version 1.25.2

**Prerequisites:**
> Requires Go installed on your system.  
>
> [![Go](https://img.shields.io/badge/Go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/dl/)

```sh
$ git clone https://github.com/scythe504/rbac-finance-processing.git
$ cd rbac-finance-processing
$ cp .env.example .env
```

```sh
# install dev dependencies
go install github.com/air-verse/air@latest # Hot Reloading
go install github.com/pressly/goose/v3/cmd/goose@latest # Database Schema Migration

```
---
### Quick Setup

**Prerequisite:**  
> Docker **must** be installed and running on your system.  
>
> [![Install Docker](https://img.shields.io/badge/Install%20Docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)](https://docs.docker.com/get-started/get-docker/)

### Starting the Backend

Server will be up on http://localhost:8080 by default.

#### Makefile

Refer to [Makefile](./Makefile) for generating openapi.yaml or instantly running the application with docker or other options.

```sh
make <makefile_cmd>
```

or 
#### Running Via Docker Compose
```sh
docker compose down -v
docker compose up --build
```

#### Running Manually
```sh
# for default config, replace with your database connection string
goose -dir migrations postgres "postgresql://melkey:password1234@localhost:5432/blueprint" up
```

```sh
# Seeds admin, analyst, and viewer accounts
go run cmd/seed/main.go
# Runs the backend
go run cmd/api/main.go
# or with Hot-reloading
air
```

Once you are up and running you can test the endpoints more details on that in [TESTING](./TESTING.md)