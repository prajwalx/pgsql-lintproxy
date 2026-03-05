# pgsql-lintproxy

**A high-performance, wire-protocol PostgreSQL proxy that enforces query safety at the network layer.**

`PGSQL Lint proxy` sits between your application (or SQL client) and your PostgreSQL database. It decodes the Postgres wire protocol in real-time, parses incoming SQL using the actual Postgres source-code parser, and blocks dangerous operations (like `DELETE` or `UPDATE` without a `WHERE` clause) before they ever hit your data.



## 🚀 Why it exists
Standard SQL linters only check static code. However, in modern development:
1. **ORMs** generate dynamic queries that are never seen by static analyzers.
2. **Manual mistakes** in GUI clients (like TablePlus or DBeaver) can lead to accidental data loss.
3. **Database triggers** and migrations are often high-risk.

`PGSQL Lint Proxy` provides a **runtime safety net** that works regardless of what language or tool is connecting to the database.

---

## ✨ Features
* **Postgres Native Parsing:** Uses `pg_query_go` (C-bindings to the actual Postgres 16+ parser) for 100% dialect accuracy.
* **Wire Protocol Interception:** Decodes both **Simple Query** and **Extended Query** (Parse/Bind/Execute) modes.
* **Non-Blocking Observability:** Transparently streams responses back from the DB while inspecting requests.
* **Protocol Safety:** Prevents client timeouts by injecting `ReadyForQuery` signals even when a query is blocked.
* **Zero Config:** Just point it at your DB and update your connection port.

---

## 🛠 Technical Challenges & Solutions

### 1. The "ReadyForQuery" Deadlock
**Problem:** Initially, blocking a query caused GUI clients (TablePlus) to hang and timeout.
**Solution:** I implemented a state-reset mechanism. When a query is blocked, the proxy doesn't just send an `ErrorResponse`; it manually injects a `ReadyForQuery` packet. This resets


### 2. Handling Extended Query Mode
Most proxies only handle raw SQL strings. `PGSQL Lint Proxy` intercepts the `Parse` message used by ORMs and GUI clients, ensuring that even prepared statements are linted for safety.

---

## 🚦 Getting Started

### Prerequisites
* Go 1.21+
* A running PostgreSQL instance (Local or Docker)

### Installation
```bash
git clone https://github.com/prajwalx/pgsql-lintproxy.git
cd pgsql-lintproxy
go mod download
```


### Usage

```bash
# Start the proxy on port 5433, pointing to a DB on 5432
go run cmd/main.go -p 5433 -db localhost:5432
```

## Testing the Linter

Connect your favorite SQL client (TablePlus, DBeaver, or psql) and create a new connection with the following details:

1. `Host: 127.0.0.1` (Always use the IP instead of localhost to avoid socket issues).

2. `Port: 5433` (Or whichever port your Go app is listening on).

3. `User/Password/Database`: Use the credentials of your actual Postgres database. Your proxy is transparently forwarding these, so the real DB still needs to authenticate them.

4. `SSL Mode`: Set this to Disable, as the proxy don't have SSL certicates configured yet.

### Allowed

```SQL
SELECT * FROM employee WHERE id = 1; -- PASS
```

### Blocked

```SQL
DELETE FROM employee; -- : ERROR:  psql-lintproxy: delete without where clause blocked
```

## 📂 Project Structure
- `cmd/main.go`: Entry point and TCP listener logic.

- `internal/proxy/server.go`: Core protocol engine. Handles the "Man-in-the-Middle" byte streaming, SSL negotiation, and pgproto3 decoding.

- `internal/linter/rules.go`: The AST-inspection logic using pg_query_go.

## 🛠 Built With
- [Go](https://github.com/golang/go) - Core language.

- [pgx/v5/pgproto3](https://www.github.com/jackc/pgx) - PostgreSQL wire protocol encoding/decoding.

- [pg_query_go/v6](https://www.github.com/pganalyze/pg_query_go) - The actual PostgreSQL parser bindings.