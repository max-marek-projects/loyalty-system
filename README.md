# Loyalty System

Loyalty points service for an online store.  
It allows users to register, submit order numbers, earn loyalty points, check balances, and withdraw points to pay for new orders.  
The system integrates with an external accrual calculator (simulated or real) that determines how many points each order earns.

---

## Features

- **User registration & authentication** (JWT stored in HTTP‑only cookies).
- **Order submission** – users upload order numbers, validated by the Luhn algorithm.
- **Order status tracking** – NEW → PROCESSING → PROCESSED / INVALID.
- **Balance management** – current points and total withdrawn.
- **Points withdrawal** – use points to pay for orders (subject to sufficient balance).
- **Background order processor** – polls the accrual system (or mock) to update order status and balances.
- **Structured logging** – `zap` with configurable log level.
- **PostgreSQL** persistence with migrations.
- **Graceful shutdown** and concurrency control (worker pool).

---

## Tech Stack

| Component       | Technology                                 |
|----------------|--------------------------------------------|
| Language       | Go 1.21+                                    |
| HTTP router    | `go-chi/chi`                    |
| Database       | PostgreSQL (pgx driver)                     |
| Migrations     | `golang-migrate`                |
| JWT handling   | `golang-jwt/jwt`                |
| Password hashing | `bcrypt`                       |
| Logging        | `uber-go/zap`                   |
| Configuration  | `caarlos0/env` + `flag` + `.env` file |
| Testing        | `testify` + `mockery`|

---

## API Endpoints

All endpoints are served under `http://<host>:<port>/api/user`.

| Method | Endpoint                 | Description                                     | Auth required |
|--------|--------------------------|-------------------------------------------------|---------------|
| POST   | `/register`              | Register a new user (login/password JSON)      | ❌            |
| POST   | `/login`                 | Authenticate and receive auth cookie           | ❌            |
| POST   | `/orders`                | Submit an order number (plain text body)       | ✅            |
| GET    | `/orders`                | List all user’s orders with statuses & accrual | ✅            |
| GET    | `/balance`               | Get current balance and total withdrawn        | ✅            |
| POST   | `/balance/withdraw`      | Withdraw points (JSON: `order`, `sum`)         | ✅            |
| GET    | `/withdrawals`           | List all withdrawal transactions               | ✅            |

---

## Configuration

The service can be configured via **environment variables**, **flags**, or a **`.env`** file (flags take precedence).

| Env variable               | Flag  | Default      | Description                              |
|----------------------------|-------|--------------|------------------------------------------|
| `RUN_ADDRESS`              | `-a`  | `:8080`      | Address to listen on                     |
| `DATABASE_URI`             | `-d`  | (none)       | PostgreSQL connection string             |
| `ACCRUAL_SYSTEM_ADDRESS`   | `-r`  | (none)       | Base URL of external accrual service     |
| `COOKIE_SECRET`            | `-s`  | (none)       | Secret key for JWT signing               |
| `LOGGER_LEVEL`             | `-l`  | `INFO`       | Log level (DEBUG/INFO/WARN/ERROR/FATAL)  |
| `MAX_PARALLEL_WORKERS`     | `-p`  | `5`          | Number of background order processors    |
| `POLL_INTERVAL`            | `-i`  | `5`          | Seconds between polling for new orders   |
| `MOCK_EXTERNAL_STORAGE`    | `-m`  | `false`      | Use mock accrual (random points)         |
| `READ_TIMEOUT`             | `-t`  | `30`         | Server read timeout (seconds)            |
| `WRITE_TIMEOUT`            | `-w`  | `30`         | Server write timeout (seconds)           |

Example `.env` file:

```bash
RUN_ADDRESS=:8080
DATABASE_URI=postgres://user:pass@localhost:5432/gophermart?sslmode=disable
ACCRUAL_SYSTEM_ADDRESS=http://accrual:8080
COOKIE_SECRET=supersecretkey
LOGGER_LEVEL=INFO
MAX_PARALLEL_WORKERS=10
POLL_INTERVAL=10
MOCK_EXTERNAL_STORAGE=false
READ_TIMEOUT=60
WRITE_TIMEOUT=60

```

---

## Running the Project

### Prerequisites

- Go 1.21+
- PostgreSQL (running and accessible)
- (Optional) External accrual service – or enable `MOCK_EXTERNAL_STORAGE`

### Steps

1. **Clone the repository**  

```bash
git clone https://github.com/yourusername/loyalty-system.git

cd loyalty-system
```

1. **Configure** – create a `.env` file or set environment variables.

2. **Run database migrations** (automatically executed on startup, but you can run manually)

```bash
go run cmd/gophermart/main.go
```

1. **Build and run**

```bash
go build -o gophermart ./cmd/gophermart

./gophermart -a :8080 -d "postgres://..." -r "http://..." -s "mysecret"
```

   Or using Docker (if you provide a `Dockerfile`):

```bash
docker build -t gophermart .
docker run -p 8080:8080 --env-file .env gophermart
```

---

## Background Order Processing

Once started, the service launches a pool of workers (`MAX_PARALLEL_WORKERS`) that:

- Periodically (every `POLL_INTERVAL` seconds) fetch `NEW` orders from the database.
- Mark them as `PROCESSING` to avoid duplicate work.
- For each order, call the external accrual system (`GET /api/orders/{number}`) or use mock logic.
- Update order status and user balance accordingly.
- Retry on `429` (rate limit) or temporary errors using the `Retry-After` header.

This ensures that order processing continues in the background without blocking HTTP handlers.

---

## Example API Calls

### Register

```bash
curl -X POST http://localhost:8080/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"login":"alice","password":"secret"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"login":"alice","password":"secret"}' \
  -c cookies.txt
```

### Submit order (plain text)

```bash
curl -X POST http://localhost:8080/api/user/orders \
  -H "Content-Type: text/plain" \
  -b cookies.txt \
  -d "49927398716"
```

### Get orders

```bash
curl -X GET http://localhost:8080/api/user/orders \
  -b cookies.txt
```

### Get balance

```bash
curl -X GET http://localhost:8080/api/user/balance \
  -b cookies.txt
```

### Withdraw

```bash
curl -X POST http://localhost:8080/api/user/balance/withdraw \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"order":"49927398716","sum":150.5}'
```

### Get withdrawals

```bash
curl -X GET http://localhost:8080/api/user/withdrawals \
  -b cookies.txt
```

---

## License

This project is for educational/portfolio purposes.  
Free to use and modify under the MIT License.

---

## Author

Max Marek – [GitHub](https://github.com/max-marek)
