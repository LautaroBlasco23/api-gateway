# Backend Registration Guide

This guide explains how to register your backend service with the API Gateway and configure it in Docker Compose.

## Quick Start

Your backend needs to make two HTTP requests to the gateway during startup:

1. **Register the service** (required)
2. **Register endpoint validations** (optional)

---

## 1. Register Your Service

### Endpoint
```
POST http://api-gateway:8080/register
```

### Request Body (JSON)

```json
{
  "name": "my-service",
  "url": "http://backend:3000",
  "routes": [
    "/api/users",
    "/api/users/{id}",
    "/api/products",
    "/api/products/{id}",
    "/health"
  ],
  "features": {
    "ratelimiter": true,
    "cors": true,
    "injection": true,
    "cache": true
  }
}
```

### Field Descriptions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ Yes | Unique service identifier. Cannot duplicate existing service names. |
| `url` | string | ✅ Yes | Your service's internal URL (where requests will be proxied). Must be unique. |
| `routes` | string[] | ✅ Yes | Array of route patterns your service handles. Use `{param}` syntax for dynamic segments (e.g., `/api/users/{id}` matches `/api/users/123`). Static routes and parameterized routes can coexist. |
| `features.ratelimiter` | boolean | ❌ No | Enable rate limiting (10 req/s, burst 20 per IP). Default: `false` |
| `features.cors` | boolean | ❌ No | Enable CORS preflight request handling. Default: `false` |
| `features.injection` | boolean | ❌ No | Enable SQL/XSS injection filtering. Default: `false` |
| `features.cache` | boolean | ❌ No | Enable response caching (30s TTL, GET/HEAD only). Default: `false` |

### Response

**Success (201 Created):**
```json
{
  "status": "registered",
  "service": "my-service"
}
```

**Error (409 Conflict):**
```json
{
  "error": "service already registered"
}
```
(Occurs if a service with the same name or URL is already registered)

**Error (400 Bad Request):**
```json
{
  "error": "name, url and routes are required"
}
```

---

## 2. Register Endpoint Validations (Optional)

After registering your service, you can optionally define input validation rules for specific routes.

### Endpoint
```
POST http://api-gateway:8080/register/endpoint
```

### Request Body (JSON)

```json
{
  "route": "/api/users",
  "method": "POST",
  "validation": {
    "email": "email",
    "age": "number",
    "name": "string",
    "profile_picture": "file_image"
  }
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `route` | string | A route pattern (may include `{param}` segments). Must match one of the routes registered for your service. |
| `method` | string | HTTP method (GET, POST, PUT, DELETE, PATCH, etc.) |
| `validation` | object | Field-level validation rules. Key = field name, Value = validation type. |

### Validation Types

| Type | Description |
|------|-------------|
| `string` | Must be a string value in JSON body |
| `number` | Must be a valid number (integer or float) |
| `email` | Must be a valid email format |
| `file_*` | File upload fields (use with multipart/form-data). Types: `file_image`, `file_pdf`, `file_text`, `file_document` |

### Examples

**JSON-based validation:**
```json
{
  "route": "/api/products",
  "method": "POST",
  "validation": {
    "name": "string",
    "price": "number",
    "email": "email"
  }
}
```

**Multipart file upload:**
```json
{
  "route": "/api/upload",
  "method": "POST",
  "validation": {
    "document": "file_pdf",
    "thumbnail": "file_image"
  }
}
```

### Response

**Success (201 Created):**
```json
{
  "status": "registered",
  "route": "/api/users"
}
```

---

## 3. Docker Compose Setup

### Option A: Add API Gateway to Your Project

Add the gateway service to your `docker-compose.yml`:

```yaml
version: '3.9'

services:
  api-gateway:
    image: api-gateway:latest  # Build locally or pull from registry
    container_name: api-gateway
    ports:
      - "8080:8080"
    environment:
      LOG_LEVEL: info
    volumes:
      - ./registry.json:/app/registry.json  # Persist service registrations
    networks:
      - backend-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  # Your backend service
  my-service:
    build: .
    container_name: my-service
    ports:
      - "3000:3000"
    depends_on:
      api-gateway:
        condition: service_healthy
    networks:
      - backend-network
    environment:
      GATEWAY_URL: http://api-gateway:8080

networks:
  backend-network:
    driver: bridge
```

### Option B: Reference Existing Gateway

If the gateway is running externally or in a separate compose file:

```yaml
version: '3.9'

services:
  my-service:
    build: .
    container_name: my-service
    ports:
      - "3000:3000"
    networks:
      - backend-network
    environment:
      GATEWAY_URL: http://api-gateway:8080

networks:
  backend-network:
    external: true  # Reference an external network
    name: api-gateway_backend-network  # Or adjust to match your setup
```

---

## 4. Service Registration Code Example

Add this to your service's startup code:

### Node.js / TypeScript

```javascript
import fetch from 'node-fetch';

async function registerWithGateway() {
  const gatewayUrl = process.env.GATEWAY_URL || 'http://api-gateway:8080';
  const serviceName = 'my-service';
  const serviceUrl = 'http://my-service:3000';

  const serviceData = {
    name: serviceName,
    url: serviceUrl,
    routes: [
      '/api/users',
      '/api/users/{id}',
      '/api/products',
      '/api/products/{id}',
      '/health'
    ],
    features: {
      ratelimiter: true,
      cors: true,
      injection: true,
      cache: true
    }
  };

  try {
    const response = await fetch(`${gatewayUrl}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(serviceData)
    });

    if (!response.ok) {
      const error = await response.json();
      console.error('Registration failed:', error);
      process.exit(1);
    }

    const result = await response.json();
    console.log('✅ Registered with gateway:', result);

    // Optional: Register endpoint validations
    await registerEndpointValidations(gatewayUrl);
  } catch (error) {
    console.error('Failed to register with gateway:', error);
    process.exit(1);
  }
}

async function registerEndpointValidations(gatewayUrl) {
  const validations = [
    {
      route: '/api/users',
      method: 'POST',
      validation: {
        email: 'email',
        name: 'string'
      }
    },
    {
      route: '/api/products',
      method: 'POST',
      validation: {
        name: 'string',
        price: 'number'
      }
    }
  ];

  for (const validation of validations) {
    try {
      const response = await fetch(`${gatewayUrl}/register/endpoint`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(validation)
      });

      if (!response.ok) {
        console.warn(`Failed to register validation for ${validation.route}`);
        continue;
      }

      const result = await response.json();
      console.log('✅ Registered endpoint validation:', result);
    } catch (error) {
      console.warn(`Error registering endpoint validation: ${error.message}`);
    }
  }
}

// Call during server startup
registerWithGateway();
```

### Python

```python
import requests
import json
import os
import sys

def register_with_gateway():
    gateway_url = os.getenv('GATEWAY_URL', 'http://api-gateway:8080')
    service_name = 'my-service'
    service_url = 'http://my-service:5000'

    service_data = {
        'name': service_name,
        'url': service_url,
        'routes': [
            '/api/users',
            '/api/users/{id}',
            '/api/products',
            '/api/products/{id}',
            '/health'
        ],
        'features': {
            'ratelimiter': True,
            'cors': True,
            'injection': True,
            'cache': True
        }
    }

    try:
        response = requests.post(
            f'{gateway_url}/register',
            json=service_data,
            timeout=5
        )

        if response.status_code != 201:
            print(f'Registration failed: {response.json()}', file=sys.stderr)
            sys.exit(1)

        result = response.json()
        print(f'✅ Registered with gateway: {result}')

        # Optional: Register endpoint validations
        register_endpoint_validations(gateway_url)

    except requests.RequestException as e:
        print(f'Failed to register with gateway: {e}', file=sys.stderr)
        sys.exit(1)

def register_endpoint_validations(gateway_url):
    validations = [
        {
            'route': '/api/users',
            'method': 'POST',
            'validation': {
                'email': 'email',
                'name': 'string'
            }
        }
    ]

    for validation in validations:
        try:
            response = requests.post(
                f'{gateway_url}/register/endpoint',
                json=validation,
                timeout=5
            )

            if response.status_code == 201:
                print(f'✅ Registered endpoint validation: {response.json()}')
            else:
                print(f'⚠️ Failed to register validation for {validation["route"]}')

        except requests.RequestException as e:
            print(f'⚠️ Error registering endpoint validation: {e}')

# Call during server startup
register_with_gateway()
```

### Go

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Service struct {
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Routes   []string `json:"routes"`
	Features Features `json:"features"`
}

type Features struct {
	RateLimiter bool `json:"ratelimiter"`
	CORS        bool `json:"cors"`
	Injection   bool `json:"injection"`
	Cache       bool `json:"cache"`
}

func registerWithGateway() error {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://api-gateway:8080"
	}

	svc := Service{
		Name: "my-service",
		URL:  "http://my-service:8000",
		Routes: []string{
			"/api/users",
			"/api/users/{id}",
			"/api/products",
			"/api/products/{id}",
			"/health",
		},
		Features: Features{
			RateLimiter: true,
			CORS:        true,
			Injection:   true,
			Cache:       true,
		},
	}

	body, err := json.Marshal(svc)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("%s/register", gatewayURL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %s", string(respBody))
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("✅ Registered with gateway: %+v\n", result)

	return nil
}

// Call during server initialization
func init() {
	if err := registerWithGateway(); err != nil {
		fmt.Printf("❌ Failed to register with gateway: %v\n", err)
		os.Exit(1)
	}
}
```

---

## 5. Important Notes

### Route Matching

- Routes use **pattern matching**: segments wrapped in `{braces}` match any non-empty path segment.
  - `/api/users/{id}` matches `/api/users/123`, `/api/users/abc`, etc.
  - `/api/users` only matches `/api/users` exactly (no dynamic segment).
- **Specificity wins**: more literal (non-parameterized) segments beat less specific patterns.
  - If both `/api/users/admin` and `/api/users/{id}` are registered, a request to `/api/users/admin` matches the first one.
- Multiple parameters are supported: `/api/orgs/{org}/repos/{repo}` is valid.
- **Cache keys use the matched pattern**, not the literal path. Requests to `/api/users/123` and `/api/users/456` share the same cache entry when both match `/api/users/{id}`.
- The gateway finds the service and proxies to `Service.URL`.

### Service Deregistration

There is no deregistration endpoint. To remove a service:
1. Delete the entry from `registry.json` (if persisted)
2. Restart the gateway

### Feature Flags

All features are optional and default to `false`. Enable only what you need:
- **Rate Limiter**: Prevents abuse (10 req/s per IP)
- **CORS**: Handles cross-origin requests
- **Injection**: Filters SQL/XSS attacks
- **Cache**: Caches GET/HEAD responses (30s TTL)

### Persistence

Service registrations are saved to `registry.json` in the gateway's container. To persist across restarts, mount this file as a volume:

```yaml
volumes:
  - ./registry.json:/app/registry.json
```

### Gateway Health Check

Test if the gateway is ready:
```bash
curl -i http://api-gateway:8080/health
```

---

## Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| "service already registered" (409) | Duplicate service name or URL | Use a unique name and URL |
| "name, url and routes are required" (400) | Missing required fields | Provide all three fields |
| "no service registered for this route" (400) | No pattern matched the request path | Check that the route pattern is registered and the path structure matches |
| Connection refused | Gateway not running/not reachable | Check gateway is running and network connectivity |

---

## Testing Your Registration

```bash
# Test service registration
curl -X POST http://api-gateway:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-service",
    "url": "http://test-service:3000",
    "routes": ["/api/test"],
    "features": {"cache": true}
  }'

# Test endpoint validation registration
curl -X POST http://api-gateway:8080/register/endpoint \
  -H "Content-Type: application/json" \
  -d '{
    "route": "/api/test",
    "method": "POST",
    "validation": {"name": "string"}
  }'

# Test proxying through the gateway
curl http://api-gateway:8080/api/test
```
