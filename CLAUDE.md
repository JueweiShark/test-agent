# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WWUC is a user and access control management microservice built with the Go Kratos framework. It provides user authentication, role-based access control (RBAC), group/team management, and permission management with multi-application support.

**Framework**: Kratos v2 (cloud-native Go microservice framework)
**Language**: Go 1.20
**Current Branch**: release-v0.2.0

## Common Commands

### Initial Setup
```bash
# Install all required protoc generators and development tools
make init

# Generate all code (API, config, and Wire DI)
make all

# Or run individually:
make api       # Generate Go code from proto files
make config    # Generate internal configuration code
make generate  # Run go generate (Wire DI)
```

### Building
```bash
# Build the application binary (outputs to ./bin/)
make build

# Clean generated code
make clean
```

### Running the Application
```bash
# Run with default config
./bin/wwuc -conf ./conf/wwuc.yaml -log ./log/wwuc.log

# Database management (initialize/update schema)
./bin/wwuc -conf ./conf/wwuc.yaml -method <db-operation>
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/service/user/...

# Run a single test
go test ./internal/service/user/ -run TestUserInfo

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Development Workflow
```bash
# After modifying .proto files:
make api       # Regenerate API code
make config    # Regenerate config code if conf.proto changed
make generate  # Regenerate Wire DI if dependencies changed

# Complete regeneration:
make all
```

## Architecture

### Layered Architecture

The codebase follows clean architecture with distinct layers:

```
Transport (HTTP/gRPC) → Service → Business Logic (BIZ) → Data (Repository) → Database
```

**Key Principle**: Dependencies flow inward. Business logic never depends on transport or data implementation details.

### Directory Structure

- **`api/`**: Protocol Buffer definitions (API contracts)
  - `user/v1/`: User, role, group, permission APIs
  - `common/`, `errors/`: Shared types and error definitions
  - `xviz/`: External API definitions
- **`api_gen/`**: Generated code from .proto files (never edit manually)
- **`cmd/wwuc/`**: Application entry point
  - `main.go`: Bootstrap and server initialization
  - `wire.go`: Dependency injection wiring
  - `db_manage.go`: Database migration utilities
- **`internal/`**: Private application code
  - **`server/`**: HTTP/gRPC server setup with middleware
    - `http_middleware.go`: JWT auth, permission validation, logging
  - **`service/`**: Service handlers (thin layer mapping API to use cases)
  - **`biz/`**: Business logic layer (use cases, domain logic)
    - Core business rules and orchestration
  - **`data/`**: Data access layer (repositories)
    - `schema/`: GORM database models
    - Repository implementations
  - **`conf/`**: Configuration structures (from conf.proto)
- **`pkg/`**: Shared packages
  - `bootstrap/`: Configuration loading (local files and Nacos)
- **`util/`**: Utility packages
- **`constants/`**: Application-wide constants
- **`conf/`**: Configuration YAML files

### Key Components

**Authentication & Authorization**:
- JWT tokens signed with HS512 algorithm
- Middleware validates tokens and extracts user context
- Casbin enforces RBAC permissions
- Permission middleware checks access on protected endpoints

**Dependency Injection**:
- Google Wire for compile-time DI
- Dependencies wired in `cmd/wwuc/wire.go`
- Changes to dependencies require running `make generate`

**Database**:
- GORM ORM with MySQL
- Models in `internal/data/schema/`
- Core entities: User, Role, Group, Permission, and their relationships
- Transaction management in data layer

**Configuration**:
- Defined in protobuf (`internal/conf/conf.proto`)
- Supports local YAML files and Nacos remote configuration
- Bootstrap package handles loading strategy

**Service Discovery** (Optional):
- Nacos for service registry
- Polaris for service mesh support

## Code Generation Workflow

This project heavily uses code generation. Understanding the generation flow is critical:

1. **Define API in .proto files** (`api/`)
2. **Run `make api`** to generate:
   - Go structs (`*.pb.go`)
   - gRPC service definitions (`*_grpc.pb.go`)
   - HTTP handlers (`*_http.pb.go`)
   - Validation code (`*.pb.validate.go`)
   - Error definitions (`*_errors.pb.go`)
   - OpenAPI specs
3. **Implement service handlers** in `internal/service/`
4. **Update Wire providers** in `cmd/wwuc/wire.go` if new dependencies added
5. **Run `make generate`** to regenerate DI code

**Important**: Never manually edit files in `api_gen/` or `wire_gen.go` - they are regenerated.

## Testing Patterns

Tests typically:
1. Load configuration from `conf/wwuc.yaml`
2. Manually initialize dependencies (database, repos, use cases)
3. Create service instances
4. Execute test cases

See `internal/service/user/user_test.go` for the standard initialization pattern.

## Middleware Stack

HTTP requests flow through this middleware chain:
1. Recovery (panic handling)
2. Tracing (request ID)
3. Logging (request/response)
4. JWT Authentication (token validation)
5. Permission Validation (Casbin RBAC)
6. Service Handler

Middleware is configured in `internal/server/http.go`.

## Configuration Files

- `conf/wwuc.yaml`: Main configuration template
- `conf/wwuc_local.yaml`: Developer-specific config (not in version control)
- `conf/wwuc_jinshan_online.yaml`: Production configuration

Configuration includes:
- Server ports (HTTP: 8922, gRPC: 9101)
- Database connection strings
- Redis configuration
- JWT signing keys
- Service discovery settings (Nacos/Polaris)

## Common Patterns

**Adding a new API endpoint**:
1. Define in appropriate `.proto` file under `api/`
2. Run `make api`
3. Implement handler in `internal/service/`
4. Implement use case in `internal/biz/`
5. Implement repository method in `internal/data/` if needed
6. Update Wire providers if new dependencies required
7. Run `make generate`

**Database schema changes**:
- Modify models in `internal/data/schema/`
- Update corresponding repository interfaces in `internal/biz/`
- Implement repository methods in `internal/data/`
- Use `db_manage.go` for migrations

**Adding middleware**:
- Implement in `internal/server/http_middleware.go`
- Register in server chain in `internal/server/http.go`

## Current Development Focus (v0.2.0)

Recent work has focused on:
- Empty/null permission handling improvements
- Role permission constraint relaxation
- Team management enhancements with roleInfo
- Multi-application support (Simulation/Embodied apps)
- Password change fixes

## Important Notes

- **Wire regeneration**: After modifying `wire.go`, always run `make generate`
- **Proto changes**: After modifying any `.proto` file, run `make api` before building
- **Configuration schema**: Changes to `internal/conf/conf.proto` require `make config`
- **Module path**: All imports use `wuwen.com/wwuc` as the module prefix
- **Branch strategy**: Currently on `release-v0.2.0`, main branch is `master`
