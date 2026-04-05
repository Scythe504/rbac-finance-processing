# Testing Guide

## Prerequisites

Before testing, ensure the application is running. Follow the setup instructions in [SETUP](./SETUP.md) to get the API up and running.

## Importing the API Collection into Postman

1. Open Postman
2. Navigate to **Workspaces → Collections**
3. Click **Import**
4. Drag and drop `docs/openapi.yaml` from the project root into the import dialog, or click **Choose Files** and select it manually
5. Postman will generate a collection with all endpoints pre-configured

## Setting Up Authentication

1. First, login using the seeded credentials from `USERS.md`:
   - `POST /api/v1/auth/login`
   - Copy the `token` value from the response
   - or simply copy paste the token from [USERS](../USERS.md) once the database has been seeded.

2. In the imported collection, click the collection name in the sidebar
3. Go to the **Variables** tab
4. Set the `apiKey` variable value to `Bearer <your-token>` (replace `<your-token>` with the copied token)
5. Save the collection

All protected routes will now automatically use this token.

## Testing Different Roles

Repeat the login step with each seeded user from `USERS.md` to test role-based access:

- **Admin** — full access to all endpoints
- **Analyst** — read-only access to records and dashboard
- **Viewer** — dashboard access only, expects `403 Forbidden` on record endpoints

## Recommended Test Order

1. `POST /api/v1/auth/login` — login as admin, copy token
2. `GET /api/v1/me` — verify session
3. `GET /api/v1/dashboard?period=weekly` — verify seeded data appears
4. `POST /api/v1/records` — create a new record
5. `GET /api/v1/records` — verify record appears, test filters
6. `GET /api/v1/records?txn_type=income&page=1&limit=10` — test pagination and filtering
7. `PATCH /api/v1/records/{id}` — update a record
8. `DELETE /api/v1/records/{id}` — soft delete a record
9. `GET /api/v1/records?show_deleted=true` — verify deleted record appears
10. `PATCH /api/v1/users/{id}/role` — change a user's role
11. `PATCH /api/v1/users/{id}/status` — toggle user active status
12. Login as analyst → verify `403` on `POST /api/v1/records`
13. Login as viewer → verify `403` on `GET /api/v1/records`