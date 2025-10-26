# Docker Test Containers - Quick Reference

## Container Management (Pure Docker via Taskfile)

### Start/Stop

```bash
# Start all containers
task containers:up

# Stop containers
task containers:down

# Restart all
task containers:restart

# Clean everything (containers + volumes + network)
task containers:clean
```

### Individual Containers

```bash
# Start individual containers
task containers:postgres
task containers:couchdb
task containers:rabbitmq
```

### Monitoring

```bash
# Check status
task containers:status

# View logs (last 50 lines)
task containers:logs

# Follow logs in real-time
task containers:logs:follow

# Individual container logs
docker logs eve-postgres-test
docker logs eve-couchdb-test
docker logs eve-rabbitmq-test

# Follow specific container
docker logs -f eve-postgres-test
```

### Health Checks

```bash
# Wait for all containers to be healthy
task containers:wait

# Check health manually
docker inspect eve-postgres-test --format='{{.State.Health.Status}}'
docker inspect eve-couchdb-test --format='{{.State.Health.Status}}'
docker inspect eve-rabbitmq-test --format='{{.State.Health.Status}}'
```

## Container Details

### PostgreSQL
- **Container**: `eve-postgres-test`
- **Image**: `postgres:16-alpine`
- **Port**: `5433:5432`
- **Volume**: `eve-postgres-test-data`
- **User**: `testuser`
- **Password**: `testpass`
- **Database**: `testdb`
- **Connection**: `host=localhost port=5433 user=testuser password=testpass dbname=testdb sslmode=disable`

### CouchDB
- **Container**: `eve-couchdb-test`
- **Image**: `couchdb:3.5`
- **Port**: `5985:5984`
- **Volume**: `eve-couchdb-test-data`
- **User**: `admin`
- **Password**: `testpass`
- **URL**: `http://admin:testpass@localhost:5985`

### RabbitMQ
- **Container**: `eve-rabbitmq-test`
- **Image**: `rabbitmq:3.13-management-alpine`
- **AMQP Port**: `5673:5672`
- **Management Port**: `15673:15672`
- **Volume**: `eve-rabbitmq-test-data`
- **User**: `guest`
- **Password**: `guest`
- **AMQP URL**: `amqp://guest:guest@localhost:5673/`
- **Management UI**: `http://localhost:15673`

### Network
- **Network**: `eve-test-network`
- **Type**: Bridge
- **Purpose**: Container-to-container communication

## Testing

### Run Tests

```bash
# All tests (unit + integration with containers)
task test:all

# Just integration tests (requires containers running)
task test:integration:local

# Unit tests only
task test

# With coverage
task coverage:integration
```

### Development Workflow

```bash
# 1. Start containers once
task containers:up

# 2. Run tests multiple times (fast)
go test -tags=integration ./db/...
go test -tags=integration ./queue/...

# 3. Check status
task containers:status

# 4. Stop when done
task containers:down
```

## Troubleshooting

### Container Already Exists

```bash
task containers:clean
task containers:up
```

### Port Conflicts

```bash
# Check what's using the ports
lsof -i :5433
lsof -i :5985
lsof -i :5673

# Stop containers
task containers:down
```

### View Detailed Logs

```bash
# Recent logs
docker logs eve-postgres-test --tail 100
docker logs eve-couchdb-test --tail 100
docker logs eve-rabbitmq-test --tail 100

# Follow logs
docker logs -f eve-postgres-test
```

### Container Not Starting

```bash
# Check logs
task containers:logs

# Try clean restart
task containers:clean
task containers:up

# Manual inspection
docker inspect eve-postgres-test
```

### Manual Cleanup

```bash
# Remove containers
docker stop eve-postgres-test eve-couchdb-test eve-rabbitmq-test
docker rm eve-postgres-test eve-couchdb-test eve-rabbitmq-test

# Remove volumes
docker volume rm eve-postgres-test-data
docker volume rm eve-couchdb-test-data
docker volume rm eve-rabbitmq-test-data

# Remove network
docker network rm eve-test-network

# Or use task
task containers:clean
```

## Direct Docker Commands

### Start Containers

```bash
# Network
docker network create eve-test-network

# PostgreSQL
docker run -d \
  --name eve-postgres-test \
  --network eve-test-network \
  -e POSTGRES_USER=testuser \
  -e POSTGRES_PASSWORD=testpass \
  -e POSTGRES_DB=testdb \
  -p 5433:5432 \
  -v eve-postgres-test-data:/var/lib/postgresql/data \
  postgres:16-alpine

# CouchDB
docker run -d \
  --name eve-couchdb-test \
  --network eve-test-network \
  -e COUCHDB_USER=admin \
  -e COUCHDB_PASSWORD=testpass \
  -p 5985:5984 \
  -v eve-couchdb-test-data:/opt/couchdb/data \
  couchdb:3.5

# RabbitMQ
docker run -d \
  --name eve-rabbitmq-test \
  --network eve-test-network \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  -p 5673:5672 \
  -p 15673:15672 \
  -v eve-rabbitmq-test-data:/var/lib/rabbitmq \
  rabbitmq:3.13-management-alpine
```

### Interactive Access

```bash
# PostgreSQL
docker exec -it eve-postgres-test psql -U testuser -d testdb

# CouchDB
curl http://admin:testpass@localhost:5985/_all_dbs

# RabbitMQ
docker exec -it eve-rabbitmq-test rabbitmqctl status
```

## Environment Variables for Tests

```bash
export POSTGRES_URL="host=localhost port=5433 user=testuser password=testpass dbname=testdb sslmode=disable"
export COUCHDB_URL="http://admin:testpass@localhost:5985"
export RABBITMQ_URL="amqp://guest:guest@localhost:5673/"
```

These are automatically set by `task test:integration:local`.

## Benefits Over docker-compose

- ✅ No docker-compose dependency
- ✅ Direct Docker commands (clearer what's happening)
- ✅ Individual container control
- ✅ Easier debugging
- ✅ Same commands everywhere
- ✅ One less tool to install

## Common Tasks

```bash
# Quick test run
task test:all

# Development cycle
task containers:up && go test -tags=integration ./... && task containers:down

# Check everything is healthy
task containers:status

# Fresh start
task containers:clean && task containers:up

# View what's happening
task containers:logs:follow
```
