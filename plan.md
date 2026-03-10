API GATEWAY V1 – DESIGN AND DEVELOPMENT PLAN

--------------------------------
1. GOAL
--------------------------------

Build a simple API Gateway implemented as a modular monolith in Go.

Main responsibilities:

- Route incoming requests to backend services
- Provide optional security features
- Allow each backend to enable/disable gateway features
- Work with monolith backends or microservices
- Be runnable via Docker and docker-compose

The gateway should stay simple for V1 and avoid unnecessary complexity.



--------------------------------
2. HIGH LEVEL ARCHITECTURE
--------------------------------

Client
  |
  v
API Gateway
  |
  v
Backend Service (monolith or microservice)

The gateway receives requests, performs security checks, and forwards the request to the correct backend.



--------------------------------
3. SERVICE REGISTRATION
--------------------------------

Each backend registers itself when it starts.

Endpoint:

POST /register

Example payload:

{
  "name": "monolith-api",
  "url": "http://monolith:8080",
  "routes": ["/api"],
  "features": {
    "cors": true,
    "ratelimiter": true,
    "injection": true,
    "cache": false,
    "auth": false
  }
}

Meaning:

- Requests starting with /api go to this service
- Gateway features are enabled/disabled per service



--------------------------------
4. SERVICE STRUCTURE
--------------------------------

Service model inside gateway:

type Service struct {
    Name string
    URL string
    Routes []string
    Features Features
}

type Features struct {
    RateLimiter bool
    Injection bool
    CORS bool
    Auth bool
    Cache bool
}



--------------------------------
5. REQUEST FLOW
--------------------------------

1. Request arrives at gateway
2. Gateway finds the service based on route prefix
3. Gateway checks enabled features for that service
4. Security features are applied
5. Request is proxied to backend
6. Response returned to client



--------------------------------
6. INJECTION SECURITY DESIGN
--------------------------------

Problem:

If the gateway blindly forwards requests, malicious inputs could reach the backend.

Examples:

SQL Injection:
  ' OR 1=1

Command injection:
  ; rm -rf /

XSS:
  <script>alert(1)</script>



--------------------------------
7. FIELD VALIDATION APPROACH
--------------------------------

Instead of guessing inputs, the backend informs the gateway what inputs it expects.

The gateway supports predefined validation types.

Allowed validation types:

email
password
username
uuid
integer
string
file_png
file_jpg
file_pdf

Example endpoint registration:

POST /register/endpoint

{
  "route": "/api/login",
  "method": "POST",
  "validation": {
    "email": "email",
    "password": "password"
  }
}

Another example:

{
  "route": "/api/upload-avatar",
  "method": "POST",
  "validation": {
    "file": "file_png"
  }
}



--------------------------------
8. HOW VALIDATION WORKS
--------------------------------

When request arrives:

1. Gateway checks if validation exists for this endpoint
2. If yes, validate each field
3. If validation fails → reject request
4. Otherwise forward request

Example logic:

if field type == email
  validate email format

if field type == integer
  ensure numeric

if field type == file_png
  verify MIME type



--------------------------------
9. SIMPLE INJECTION PROTECTION
--------------------------------

Gateway also performs global filtering.

Reject requests containing dangerous patterns like:

<script
DROP TABLE
--
; rm



--------------------------------
10. CACHING STRATEGY
--------------------------------

Cache only safe operations.

Safe methods:

GET
HEAD

Unsafe methods (never cache):

POST
PUT
PATCH
DELETE

Cache key:

METHOD + URL + QUERY

Example:

GET:/products?page=1

Cache Flow:

request
  |
  v
is GET?
  |
  v
check cache
  |
  |--- hit → return cached response
  |
  |--- miss → call backend
                  |
                  v
             store response

Cache expiration uses TTL.

Recommended TTL:

30 seconds



--------------------------------
11. PROJECT STRUCTURE
--------------------------------

api-gateway

cmd/server
  main.go

internal

  registry
    registry.go
    service.go

  gateway
    router.go
    handler.go
    proxy.go

  features

    ratelimiter
      limiter.go

    cors
      cors.go

    injection
      filter.go

    cache
      cache.go

  validation
    validator.go
    types.go

  config
    config.go



--------------------------------
12. DOCKERFILE
--------------------------------

FROM golang:1.23-alpine

WORKDIR /app

COPY . .

RUN go build -o gateway ./cmd/server

EXPOSE 8080

CMD ["./gateway"]



--------------------------------
13. DOCKER COMPOSE EXAMPLE
--------------------------------

version: "3.9"

services:

  gateway:
    build: .
    ports:
      - "8080:8080"

  monolith:
    image: my-monolith
    ports:
      - "9000:9000"



--------------------------------
14. DEVELOPMENT PLAN
--------------------------------

STEP 1
Core Gateway

Implement:
- HTTP server
- registry
- route resolution
- reverse proxy


STEP 2
Security Modules

Implement:

- CORS
- rate limiter
- injection filter


STEP 3
Feature Flags

Allow enabling/disabling features per service


STEP 4
Endpoint Validation

Allow services to declare expected input types


STEP 5
Caching

Implement:

- in-memory cache
- TTL expiration
- GET-only caching


STEP 6
Docker Support

Provide:

Dockerfile
docker-compose example



--------------------------------
15. FINAL V1 FEATURES
--------------------------------

Routing
Service registry
Feature flags per service
Rate limiting
CORS
Injection filtering
Input validation
GET response caching
Docker deployment

This design intentionally avoids advanced features like:

- load balancing
- service discovery
- circuit breakers
- distributed caching
- metrics

to keep the first version simple and maintainable.
