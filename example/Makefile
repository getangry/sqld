# SQLd Example Makefile

.PHONY: help up down logs test clean reset generate

# Default target
help:
	@echo "Available commands:"
	@echo "  make up          - Start the application and database"
	@echo "  make down        - Stop all services"
	@echo "  make logs        - View application logs"
	@echo "  make test        - Run API tests"
	@echo "  make test-cursor - Test cursor-based pagination"
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
	@echo "Application ready at http://localhost:8081"
	@echo "Try: curl http://localhost:8081/users"

# Stop services
down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f app

# Run API tests
test:
	@echo "Testing API endpoints..."
	@echo "\n1. List first 3 users (cursor pagination):"
	@curl -s "http://localhost:8081/users?limit=3" | jq '{users_count: (.users | length), has_more, first_user: .users[0].name}' || curl -s "http://localhost:8081/users?limit=3"
	
	@echo "\n\n2. Search by name containing 'john':"
	@curl -s "http://localhost:8081/users?name[contains]=john" | jq '{users_count: (.users | length), has_more}' || curl -s "http://localhost:8081/users?name[contains]=john"
	
	@echo "\n\n3. Search active users over 25 (with cursor):"
	@curl -s "http://localhost:8081/users?status=active&age[gt]=25&limit=3" | jq '{users_count: (.users | length), has_more, users: [.users[].name]}' || curl -s "http://localhost:8081/users?status=active&age[gt]=25&limit=3"
	
	@echo "\n\n4. Search US admins and managers (SQLc optimized):"
	@curl -s "http://localhost:8081/users?status=active&limit=5" | jq '{users_count: (.users | length), has_more, users: [.users[].name]}' || curl -s "http://localhost:8081/users?status=active&limit=5"
	
	@echo "\n\n5. Get user by ID:"
	@curl -s "http://localhost:8081/users/1" | jq '{name, email, status}' || curl -s "http://localhost:8081/users/1"

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
	@curl -s -f http://localhost:8081/users >/dev/null && echo "✓ Application is responding" || echo "✗ Application is not responding"

# Test cursor-based pagination
test-cursor:
	@echo "Testing cursor-based pagination..."
	@echo "\n1. First page (limit 3):"
	@RESPONSE1=$$(curl -s "http://localhost:8081/users?limit=3") && echo "$$RESPONSE1" | jq '{users_count: (.users | length), has_more, first_user: .users[0].name}'
	@echo "\n2. Get cursor for next page:"
	@CURSOR=$$(curl -s "http://localhost:8081/users?limit=3" | jq -r '.next_cursor') && echo "Cursor: $$CURSOR"
	@echo "\n3. Second page using cursor:"
	@CURSOR=$$(curl -s "http://localhost:8081/users?limit=3" | jq -r '.next_cursor') && curl -s "http://localhost:8081/users?limit=3&cursor=$$CURSOR" | jq '{users_count: (.users | length), has_more, first_user: .users[0].name}'
	@echo "\n4. Status filter with cursor:"
	@curl -s "http://localhost:8081/users?status=active&limit=2" | jq '{users_count: (.users | length), has_more, users: [.users[].name]}'
	@echo "\n5. Complex filter with cursor:"
	@curl -s "http://localhost:8081/users?status=active&country=US&limit=2" | jq '{users_count: (.users | length), has_more, users: [.users[].name]}'

# Show useful URLs demonstrating SQLc + sqld annotation system
urls:
	@echo "SQLc + sqld Annotation System Demo URLs:"
	@echo ""
	@echo "Basic queries (uses SQLc SearchUsers with annotations):"
	@echo "  http://localhost:8081/users?limit=5"
	@echo "  http://localhost:8081/users?limit=3&cursor=..."
	@echo ""
	@echo "Status filter (uses SQLc SearchUsersByStatus with annotations):"
	@echo "  http://localhost:8081/users?status=active&limit=3"
	@echo ""
	@echo "Complex filters (uses SQLc SearchUsers + dynamic filtering):"
	@echo "  http://localhost:8081/users?status=active&country=US&limit=2"
	@echo "  http://localhost:8081/users?name[contains]=smith&limit=3"
	@echo "  http://localhost:8081/users?age[gt]=25&verified=true&limit=5"
	@echo "  http://localhost:8081/users?role[in]=admin,manager&country=US&limit=3"
	@echo ""
	@echo "Static queries (uses original SQLc queries):"
	@echo "  http://localhost:8081/users/1"
	@echo "  http://localhost:8081/users/john@example.com"