# Zorvyn RBAC Finance Data Processing API

A backend system for managing financial records with Role-Based Access Control, built as a screening assignment for a backend internship role from [zorvyn.io](https://zorvyn.io).

## Architecture (Diagram)

[Excalidraw Link](https://excalidraw.com/#json=6Frc6NKOYznbKHCIzV-gS,bDbkl-4JEcNymaVsEsDPVA)

## Quick Start

Requires Docker and Docker Compose.

```bash
git clone https://github.com/scythe504/zorvyn-rbac-finance
cd zorvyn-rbac-finance
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
## Assumptions

**Single organization** — The system assumes all financial records belong to a single company. A multi-tenant implementation would require an `organizations` table, with users belonging to an organization and records scoped to it via a foreign key. This was out of scope for the assignment.

**Registration** — The user registration only supports registering with viewer role, an admin can promote a user to the desired role.

**Dashboard is company-wide** — Aggregated financial data represents the entire organization, not per-user summaries. All authenticated roles see the same dashboard numbers.

**`user_id` on records is audit-only** — Records are not ownership-scoped to the user who created them. `user_id` tracks who entered the record for audit purposes, not for access control.

**Soft delete is one-way via API** — Deleted records can be restored by directly setting `deleted_at` to NULL in the database, but no restore endpoint was implemented as it was not in the requirements.

**Single currency** — No currency field on records. All amounts are assumed to be in the same currency. Multi-currency support would require exchange rate handling.

**No token invalidation** — JWTs are stateless. Logout is handled client-side by discarding the token. A compromised token remains valid until expiry (30 days). Production would require a token blacklist or short expiry with refresh tokens.

## Design Decisions & Tradeoffs

**Raw SQL over ORM** — Used `database/sql` with `pgx` driver directly instead of GORM or similar. This gives full control over query construction, makes aggregation queries explicit, and avoids the magic that ORMs introduce around dynamic filtering. The tradeoff is more verbose code for basic CRUD operations.

**Flat RBAC over per-resource ownership** — Roles are system-wide rather than per-record. An admin manages all records regardless of who created them, which fits the use case of a company financial dashboard where a finance team manages shared data. Per-resource ownership (where each user only sees their own records) would require ABAC which is significantly more complex.

**UUID for users, SERIAL for records** — User IDs are UUIDs to prevent enumeration attacks since they appear in JWTs and API responses. Record IDs are SERIAL integers since they are internal data, never exposed in a way that leaks business intelligence, and SERIAL gives natural ordering for free which benefits pagination.

**`date` vs `created_at`** — Records have two timestamp fields. `created_at` is system-generated and tracks when the record was entered into the system. `date` is user-supplied and tracks when the transaction actually occurred, allowing backdating of entries. Dashboard filters operate on `date` so queries like "show me March expenses" return transactions that happened in March, not transactions entered in March.

**Offset pagination over cursor-based** — Simpler to implement and sufficient for this scale. Cursor-based pagination would be more performant on large datasets since offset requires scanning and skipping rows. Noted as a future improvement.

**Concurrent dashboard queries** — The four dashboard sub-queries (balance totals, category totals, trends, recent activity) run concurrently using `errgroup` from `golang.org/x/sync`. Since they are independent read operations with no data dependencies, this reduces dashboard response time from O(sum of query times) to O(max query time).

**Development** — `Dockerfile.dev` includes the full Go toolchain and Air for hot reloading.
