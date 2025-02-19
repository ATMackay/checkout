# Checkout 

## Components

* Go HTTP server exposing a RESTful API built with [httprouter](https://github.com/julienschmidt/httprouter).
* Multiple DB support: SQLite (in-memory and file store), Postgres (for production use)
* Dedicated `promotions` engine for applying promotions/deals to purchases
* [Prometheus](https://prometheus.io/) metrics server endpoint.

## System Design 

The Checkout server is stateless and exposes a RESTful http interface.

### Code layout

```
.
├── main.go      // main application entrypoint
├── build        // stores generated binary and code coverage data
├── client       // HTTP client wrappers - Useful for 
├── cmd          // CLI command package (cobra/viper)
├── constants    // Contains embedded version, service name and other global constants
├── database     // Multiple database driver implementations using GORM
├── docs
│   ├── openapi  // OpenAPI/Swagger docs that can be imported into Postman
│   └── markdown // Additional documentation
├── integration  // Integration testing
├── model        // Database and API model.
├── promotions   // Engine with composable promotions strategy implementations
└── server       // HTTP server built with httprouter (high performance, panic recovery, concurrency)
```

## REST API Endpoints

### **1. Status**
- **Endpoint:** `GET /status`
- **Description:** Returns the status of the service.
- **Response:**
  ```json
  {
    "message": "OK",
    "version": "1.0.0",
    "service": "checkout-svc"
  }
  ```

---

### **2. Health**
- **Endpoint:** `GET /health`
- **Description:** Checks the health of the service and its dependencies.
- **Response:**
  ```json
  {
    "version": "1.0.0",
    "service": "checkout-svc",
    "failures": []
  }
  ```

---

### **3. Get Item Price**
- **Endpoint:** `GET /v0/inventory/item/price/:key`
- **Description:** Get price information for a single item by SKU or name.
- **Parameters:**
  - `key` (path): The SKU or name of the item.
- **Response:**
  ```json
  {
    "items": [
      {
        "name": "Item1",
        "sku": "SKU1",
        "price": 10.99,
        "inventory_quantity": 100
      }
    ],
    "total": 10.99,
    "total_with_discount": 10.99
  }
  ```

---

### **4. Get Items Price**
- **Endpoint:** `POST /v0/inventory/items/price`
- **Description:** Get total price for a batch of items by SKUs.
- **Request Body:**
  ```json
  {
    "skus": ["SKU1", "SKU2"]
  }
  ```
- **Response:**
  ```json
  {
    "items": [
      {
        "name": "Item1",
        "sku": "SKU1",
        "price": 10.99,
        "inventory_quantity": 100
      },
      {
        "name": "Item2",
        "sku": "SKU2",
        "price": 20.99,
        "inventory_quantity": 200
      }
    ],
    "total": 31.98,
    "total_with_discount": 31.98
  }
  ```

---

### **5. Purchase Items**
- **Endpoint:** `POST /v0/inventory/items/purchase`
- **Description:** Execute a purchase order for the supplied item list.
- **Request Body:**
  ```json
  {
    "skus": ["SKU1", "SKU2"]
  }
  ```
- **Response:**
  ```json
  {
    "order_reference": "ORD-12345",
    "cost": 31.98
  }
  ```


### **6. Get Orders**
- **Endpoint:** `GET /v0/orders`
- **Description:** List all purchase orders.
- **Authentication:** Required.
- **Response:**
  ```json
  [
    {
      "reference": "ORD-12345",
      "skus": ["SKU1", "SKU2"],
      "cost": 31.98
    }
  ]
  ```

### **7. Add Items**
- **Endpoint:** `POST /v0/inventory/items`
- **Description:** Add new or updated items to the inventory.
- **Authentication:** Required.
- **Request Body:**
  ```json
  {
    "items": [
      {
        "name": "Item1",
        "sku": "SKU1",
        "price": 10.99,
        "inventory_quantity": 100
      }
    ]
  }
  ```
- **Response:**
  ```json
  {
    "items": [
      {
        "id": 1,
        "name": "Item1",
        "sku": "SKU1",
        "price": 10.99,
        "inventory_quantity": 100
      }
    ]
  }
  ```


### **Authentication**
- **Endpoints requiring authentication:**
  - `GET /v0/orders`
  - `POST /v0/inventory/items`
- **Authentication Method:** Include a valid password in the `X-Auth-Password` header.

### **Error Responses**
All endpoints return the following error response in case of failure:
```json
{
  "error": "Error message"
}
```

## Getting started

### Usage
```
$ make build
$ ./build/checkout --help   
checkout server command line interface.

VERSION:
  semver: v0.0.1-1-g0ab807f
  commit: 0ab807f7b986bd8cfa4b593e97b02da62bc4ba92
  compilation date: 2025-02-19 22:08:26

Usage:
  checkout [subcommand] [flags]
  checkout [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  run         Start the checkout server
  version     Print version details

Flags:
  -h, --help   help for checkout

Use "checkout [command] --help" for more information about a command.
```

### Run the checkout server with in-memory DB
```
$ make run
```

### Run with SQLite
```
$ make build
$ ./build/checkout run --sqlite data/db --log-level debug --password 1234
```

### Run with connection to Postgres
Ensure that you have a Postgres instance listening on <DB_HOST>:<DB_PORT> ready to accept connection.
```
$ make build
$ ./build/checkout run --db-host <DB_HOST> --db-port <DB_PORT> --db-user <DB_USER> --db-password <DB_PASSWORD> --password 1234
```

### Run package tests
```
$ make test
```

