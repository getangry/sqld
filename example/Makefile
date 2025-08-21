# SQLd Example Makefile

.PHONY: help up down logs test clean reset generate

# Default target
help:
	@echo "Available commands:"
	@echo "  make up          - Start the application and database"
	@echo "  make down        - Stop all services"
	@echo "  make logs        - View application logs"
	@echo "  make test        - Run API tests"
	@echo "  make clean       - Stop and remove all containers and volumes"
	@echo "  make reset       - Reset database with fresh data"
	@echo "  make generate    - Regenerate SQLc code"
	@echo "  make psql        - Connect to PostgreSQL"
	@echo "  make simple      - Run simple_usage.go example"
	@echo "  make integration - Run integration.go example (default)"

# Start services
up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Application ready at http://localhost:8080"
	@echo "Try: curl http://localhost:8080/users"

# Stop services
down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f app

# Run API tests
test:
	@echo "Testing API endpoints..."
	@echo "\n1. List all users:"
	@curl -s "http://localhost:8080/users" | jq '.[0:3]' || curl -s "http://localhost:8080/users"
	
	@echo "\n\n2. Search by name containing 'john':"
	@curl -s "http://localhost:8080/users?name[contains]=john" | jq '.' || curl -s "http://localhost:8080/users?name[contains]=john"
	
	@echo "\n\n3. Search active users over 25:"
	@curl -s "http://localhost:8080/users?status=active&age[gt]=25" | jq '.[0:2]' || curl -s "http://localhost:8080/users?status=active&age[gt]=25"
	
	@echo "\n\n4. Search US admins and managers:"
	@curl -s "http://localhost:8080/users?role[in]=admin,manager&country=US" | jq '.' || curl -s "http://localhost:8080/users?role[in]=admin,manager&country=US"
	
	@echo "\n\n5. Get user by ID:"
	@curl -s "http://localhost:8080/users/1" | jq '.' || curl -s "http://localhost:8080/users/1"

# Clean up everything
clean:
	docker-compose down -v --rmi all

# Reset database
reset:
	docker-compose down -v
	docker-compose up -d postgres
	@echo "Database reset. Starting application..."
	@sleep 5
	docker-compose up -d app

# Regenerate SQLc code
generate:
	cd sqlc && sqlc generate
	@echo "SQLc code regenerated"

# Connect to PostgreSQL
psql:
	docker-compose exec postgres psql -U sqld_user -d sqld_db

# Run simple usage example
simple:
	@echo "Rebuilding with simple_usage.go..."
	@sed -i.bak 's/integration.go/simple_usage.go/g' Dockerfile
	docker-compose up --build app
	@mv Dockerfile.bak Dockerfile

# Run integration example (default)
integration:
	@echo "Rebuilding with integration.go..."
	@sed -i.bak 's/simple_usage.go/integration.go/g' Dockerfile
	docker-compose up --build app
	@mv Dockerfile.bak Dockerfile

# Quick development cycle
dev: down up logs

# Health check
health:
	@echo "Checking service health..."
	@docker-compose ps
	@echo "\nTesting database connection:"
	@docker-compose exec postgres pg_isready -U sqld_user -d sqld_db
	@echo "\nTesting application:"
	@curl -s -f http://localhost:8080/users >/dev/null && echo "✓ Application is responding" || echo "✗ Application is not responding"

# Show useful URLs
urls:
	@echo "Useful URLs for testing:"
	@echo "  http://localhost:8080/users"
	@echo "  http://localhost:8080/users?name=john"
	@echo "  http://localhost:8080/users?name[contains]=smith"
	@echo "  http://localhost:8080/users?age[gt]=25"
	@echo "  http://localhost:8080/users?status=active&age[gte]=30"
	@echo "  http://localhost:8080/users?role[in]=admin,manager&country=US"
	@echo "  http://localhost:8080/users?age[between]=25,35"
	@echo "  http://localhost:8080/users?verified=true&country[in]=US,CA"
	@echo "  http://localhost:8080/users/1"
	@echo "  http://localhost:8080/users/john@example.com"