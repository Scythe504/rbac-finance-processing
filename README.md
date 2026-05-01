# RBAC Finance Processing API

A robust backend system for managing financial records with Role-Based Access Control (RBAC).

**Live Demo:** [https://rbac-finance.onrender.com](https://rbac-finance.onrender.com)

## Architecture (Diagram)

[Excalidraw Link](https://excalidraw.com/#json=6Frc6NKOYznbKHCIzV-gS,bDbkl-4JEcNymaVsEsDPVA)

## Quick Start

Requires Docker and Docker Compose.

```bash
git clone https://github.com/scythe504/rbac-finance-processing
cd rbac-finance-processing
cp .env.example .env
docker compose up
```

The API will be available at `http://localhost:8080/api/v1`. Migrations and seeding run automatically on startup when running via docker compose.

For Detailed Setup check [SETUP](./docs/SETUP.md).

## Test Credentials

Seeded user credentials are available in `USERS.md`. Use these to test different access levels without registering manually.

## API Documentation

Import `docs/openapi.yaml` into Postman for organized routes testing.

## Roles & Access Control

| Role | Dashboard | View Records | Create/Update/Delete Records | Manage Users |
|------|-----------|--------------|------------------------------|--------------|
| Viewer | ✅ | ❌ | ❌ | ❌ |
| Analyst | ✅ | ✅ | ❌ | ❌ |
| Admin | ✅ | ✅ | ✅ | ✅ |

---
## Technical Considerations & Implementation Details

**Single organization** — The system currently manages financial records for a single entity. Scaling to multi-tenancy would involve introducing an `organizations` table and scoping all records and users accordingly.

**Registration** — Default registration assigns the `viewer` role. Administrative users can promote accounts to `analyst` or `admin` as needed.

**Dashboard Scope** — Aggregated financial data represents the entire organization's state. All authenticated roles have access to these high-level metrics.

**Audit Logging** — Records include a `user_id` field specifically for auditing purposes, tracking which administrator or analyst performed the entry.

**Soft Deletes** — Records utilize soft deletion to preserve data integrity. While they are excluded from standard queries, they remain in the database with a `deleted_at` timestamp.

**Currency Handling** — Currently assumes a single uniform currency across all transactions. Multi-currency support is a planned extension involving exchange rate tracking.

**Stateless Authentication** — Uses JWTs for stateless session management. In a high-security production environment, this would be supplemented with a token revocation list or shorter-lived access tokens with refresh rotations.

## Design Decisions & Tradeoffs

**Raw SQL over ORM** — Used `database/sql` with `pgx` driver directly instead of GORM or similar. This gives full control over query construction, makes aggregation queries explicit, and avoids the magic that ORMs introduce around dynamic filtering. The tradeoff is more verbose code for basic CRUD operations.

**Flat RBAC over per-resource ownership** — Roles are system-wide rather than per-record. An admin manages all records regardless of who created them, which fits the use case of a company financial dashboard where a finance team manages shared data. Per-resource ownership (where each user only sees their own records) would require ABAC which is significantly more complex.

**UUID for users, SERIAL for records** — User IDs are UUIDs to prevent enumeration attacks since they appear in JWTs and API responses. Record IDs are SERIAL integers since they are internal data, never exposed in a way that leaks business intelligence, and SERIAL gives natural ordering for free which benefits pagination.

**`date` vs `created_at`** — Records have two timestamp fields. `created_at` is system-generated and tracks when the record was entered into the system. `date` is user-supplied and tracks when the transaction actually occurred, allowing backdating of entries. Dashboard filters operate on `date` so queries like "show me March expenses" return transactions that happened in March, not transactions entered in March.

**Offset pagination over cursor-based** — Simpler to implement and sufficient for this scale. Cursor-based pagination would be more performant on large datasets since offset requires scanning and skipping rows. Noted as a future improvement.

**Concurrent dashboard queries** — The four dashboard sub-queries (balance totals, category totals, trends, recent activity) run concurrently using `errgroup` from `golang.org/x/sync`. Since they are independent read operations with no data dependencies, this reduces dashboard response time from O(sum of query times) to O(max query time).

**Development** — `Dockerfile.dev` includes the full Go toolchain and Air for hot reloading.
