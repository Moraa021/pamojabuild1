# Manual Lightning Testing Guide

This guide tests the Lightning donation flow.

The goal is to prove, manually, that:

1. The backend can create a real LND invoice.
2. A payer can pay that invoice.
3. The backend hears about the payment from LND.
4. The backend marks the invoice as `settled`.
5. The ledger is credited once.
6. Old unpaid invoices become `expired`.
7. Restart recovery catches payments that happened while the backend was offline.

Use regtest only. Do not use mainnet funds.

## What You Need

- Go installed.
- `curl`.
- `jq`.
- Docker running.
- Polar running with a regtest Lightning network.
- At least two LND nodes in Polar:
  - One node for the backend. This guide calls it `alice`.
  - One node that pays invoices. This guide calls it `bob`.
- A working payment route from `bob` to `alice`.

In Polar terms, `alice` is the LND node our backend talks to. `bob` is the wallet/node we use to pay the invoice.

If Bob cannot pay Alice invoices, make sure there is a funded channel with Bob having outbound liquidity toward Alice. In plain language: Bob needs spendable Lightning balance that can reach Alice.

## Important Moving Parts

The backend uses these LND settings:

- `LND_HOST`: where Alice's LND gRPC server is.
- `LND_MACAROON_HEX`: Alice's permission token, encoded as hex.
- `LND_TLS_PATH`: Alice's TLS certificate path on your machine.
- `LND_CLIENT_MODE=grpc`: tells the backend to use the gRPC LND client.

The backend endpoints we will use:

- `POST /api/v1/auth/register`
- `POST /api/v1/tasks`
- `POST /api/v1/tasks/:slug/donate`
- `GET /api/v1/lightning/invoices/status?payment_hash=...`
- `GET /api/v1/ledger/tasks/:slug`

## Terminal Layout

Use four terminals:

1. **Host backend terminal**
   - Run the Go backend from this repo.
2. **Host API terminal**
   - Run `curl` commands against the backend.
3. **Polar Alice terminal**
   - Check the backend LND node.
4. **Polar Bob terminal**
   - Pay invoices.

You can open Polar node terminals from Polar, but all commands below are terminal commands.

## Step 1: Pick the Polar LND Container for Alice

Run this on your host machine:

```bash
docker ps --format '{{.Names}}\t{{.Ports}}' | grep -i lnd
```

Find the container that belongs to the Polar LND node you want the backend to use.

Set it:

```bash
export POLAR_ALICE_CONTAINER="<paste alice lnd container name here>"
```

Example:

```bash
export POLAR_ALICE_CONTAINER="polar-n1-alice"
```

Check that Docker can see it:

```bash
docker exec "$POLAR_ALICE_CONTAINER" lncli getinfo
```

Expected result:

- You should see JSON about Alice's node.
- The command should not fail.

## Step 2: Export Alice LND Connection Settings

Run these commands from the repo root or from `backend`. The examples below assume you are in `backend`.

```bash
cd /home/frawuor/projects/personal/pamojabuild1/backend
mkdir -p .tmp/polar-lnd
```

Find the macaroon and TLS certificate inside the Alice container:

```bash
export LND_MACAROON_IN_CONTAINER=$(docker exec "$POLAR_ALICE_CONTAINER" sh -lc 'find / -name admin.macaroon 2>/dev/null | head -n 1')
export LND_TLS_IN_CONTAINER=$(docker exec "$POLAR_ALICE_CONTAINER" sh -lc 'find / -name tls.cert 2>/dev/null | head -n 1')

echo "$LND_MACAROON_IN_CONTAINER"
echo "$LND_TLS_IN_CONTAINER"
```

Expected result:

- The first path should end in `admin.macaroon`.
- The second path should end in `tls.cert`.

Copy Alice's TLS cert from the container to your local backend folder:

```bash
docker cp "$POLAR_ALICE_CONTAINER:$LND_TLS_IN_CONTAINER" .tmp/polar-lnd/tls.cert
```

Get Alice's macaroon as hex:

```bash
export LND_MACAROON_HEX=$(docker exec "$POLAR_ALICE_CONTAINER" sh -lc "xxd -p -c 100000 '$LND_MACAROON_IN_CONTAINER'")
```

If `xxd` is missing in the container, use this fallback:

```bash
export LND_MACAROON_HEX=$(docker exec "$POLAR_ALICE_CONTAINER" sh -lc "od -An -v -tx1 '$LND_MACAROON_IN_CONTAINER' | tr -d ' \n'")
```

Find Alice's exposed gRPC port:

```bash
export LND_GRPC_PORT=$(docker port "$POLAR_ALICE_CONTAINER" 10009/tcp | awk -F: 'NR==1 {print $NF}')
echo "$LND_GRPC_PORT"
```

Set the backend LND environment variables:

```bash
export LND_CLIENT_MODE=grpc
export LND_HOST="127.0.0.1:$LND_GRPC_PORT"
export LND_TLS_PATH="$PWD/.tmp/polar-lnd/tls.cert"
```

Quick check:

```bash
echo "$LND_HOST"
echo "$LND_TLS_PATH"
test -n "$LND_MACAROON_HEX" && echo "macaroon hex is set"
```

Expected result:

- `LND_HOST` should look like `127.0.0.1:xxxxx`.
- `LND_TLS_PATH` should point to `.tmp/polar-lnd/tls.cert`.
- You should see `macaroon hex is set`.

## Step 3: Start the Backend

Use a dedicated PostgreSQL database for this manual test:

```bash
cd backend

dropdb --if-exists pamoja_lightning_manual
createdb pamoja_lightning_manual

export DATABASE_URL='postgres://localhost:5432/pamoja_lightning_manual?sslmode=disable'
export SERVER_PORT=8080
export SESSION_COOKIE_SECURE=false
export SERVER_SECRET=manual-test-ledger-secret

go run ./cmd/migrate up
go run ./cmd/app
```

Expected result:

- The migration command applies the current schema before server startup.
- The backend logs `Server starting on port 8080`.
- The backend keeps running.
- Leave this terminal open.

If you see a TLS error when the backend tries to talk to LND, check that:

- `LND_HOST` points to Alice's exposed `10009` port.
- `LND_TLS_PATH` points to Alice's `tls.cert`.
- You copied the cert from the same node you are connecting to.

## Step 4: Create a User and Task

Run this in the Host API terminal:

```bash
export API="http://localhost:8080/api/v1"
export PHONE="+1555$(date +%s)"
export COOKIE_JAR="$(mktemp)"

export REGISTER_RESPONSE=$(curl -sS -X POST "$API/auth/register" \
  -c "$COOKIE_JAR" \
  -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -d "{\"phone_number\":\"$PHONE\",\"password\":\"password123\",\"display_name\":\"Manual Lightning Tester\"}")

echo "$REGISTER_RESPONSE" | jq

export USER_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.user_id')

echo "$USER_ID"
```

Expected result:

- `COOKIE_JAR` should contain the backend's `pamojabuild_session` cookie.
- `USER_ID` should be a number.

Create a task:

```bash
export TASK_RESPONSE=$(curl -sS -X POST "$API/tasks" \
  -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Manual Lightning Test\",
    \"description\": \"Testing LND invoice creation and settlement\",
    \"category\": \"testing\",
    \"region\": \"regtest\",
    \"goal_sats\": 5000,
    \"max_volunteers\": 1,
    \"volunteer_mode\": \"open\"
  }")

echo "$TASK_RESPONSE" | jq

export TASK_SLUG=$(echo "$TASK_RESPONSE" | jq -r '.slug')
echo "$TASK_SLUG"
```

Expected result:

- `TASK_SLUG` should not be empty.

## Test 1: Create a Real Lightning Invoice

This proves:

- The backend can ask Alice LND to create an invoice.
- The invoice is saved in the backend database.

Run:

```bash
export INVOICE_RESPONSE=$(curl -sS -X POST "$API/tasks/$TASK_SLUG/donate" \
  -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -d '{"amount_sats": 1000}')

echo "$INVOICE_RESPONSE" | jq

export BOLT11=$(echo "$INVOICE_RESPONSE" | jq -r '.payment_request')
export PAYMENT_HASH=$(echo "$INVOICE_RESPONSE" | jq -r '.payment_hash')

echo "$BOLT11"
echo "$PAYMENT_HASH"
```

Expected result:

- `payment_request` should start with something like `lnbcrt`.
- `payment_hash` should be 64 hex characters.
- `expires_at` should be a Unix timestamp.

Check the invoice status:

```bash
curl -sS "$API/lightning/invoices/status?payment_hash=$PAYMENT_HASH" \
  -b "$COOKIE_JAR" | jq
```

Expected result:

```json
{
  "payment_hash": "...",
  "status": "pending",
  "settled": false,
  "expires_at": 1234567890
}
```

## Test 2: Pay the Invoice From Bob

This proves:

- Bob can pay the BOLT11 invoice.
- Alice LND sees the payment.
- The backend listener receives the settlement.
- The backend marks the invoice `settled`.
- The task ledger is credited.

In the Polar Bob terminal, run:

```bash
lncli payinvoice --force "<paste BOLT11 here>"
```

Use the value from:

```bash
echo "$BOLT11"
```

Expected result in Bob terminal:

- The payment should succeed.
- If it fails with `unable to find a path`, Bob does not have a usable payment route to Alice. Fix the Polar channel/liquidity, then try again with a new invoice.

Back in the Host API terminal, poll the invoice status:

```bash
for i in $(seq 1 20); do
  curl -sS "$API/lightning/invoices/status?payment_hash=$PAYMENT_HASH" \
    -b "$COOKIE_JAR" | jq
  sleep 1
done
```

Expected result:

- It should change from `pending` to `settled`.
- `settled` should become `true`.
- `settled_at` should appear.

Check the ledger:

```bash
curl -sS "$API/ledger/tasks/$TASK_SLUG" \
  -b "$COOKIE_JAR" | jq
```

Expected result:

- The task's Lightning balance should include the donation amount.
- For this test, expect `1000` sats to be reflected.

What happened in the code:

1. `RequestDonationInvoice` created the invoice through LND.
2. `StartSettlementListener` was already subscribed to LND invoice updates.
3. LND reported the invoice as settled.
4. `ProcessIncomingSettlement` marked the invoice settled.
5. The service published `PaymentSettled`.
6. The router's event subscriber recorded an `INBOUND_DONATION` ledger entry.

## Test 3: Duplicate Settlement Safety

This proves:

- Paying the same invoice cannot credit the ledger twice.

Manual duplicate settlement is hard to force from LND, because LND normally will not let Bob pay the exact same invoice twice.

So for manual testing, we verify the effect:

1. Check the ledger balance after the first payment.
2. Restart the backend.
3. Confirm the invoice still says `settled`.
4. Confirm the ledger balance did not increase again.

Stop the backend with `Ctrl+C`, then start it again with the same environment variables:

```bash
cd /home/frawuor/projects/personal/pamojabuild1/backend
go run ./cmd/app
```

In the Host API terminal:

```bash
curl -sS "$API/lightning/invoices/status?payment_hash=$PAYMENT_HASH" \
  -b "$COOKIE_JAR" | jq

curl -sS "$API/ledger/tasks/$TASK_SLUG" \
  -b "$COOKIE_JAR" | jq
```

Expected result:

- Invoice status remains `settled`.
- The ledger balance does not increase a second time.

What happened in the code:

- The database update only changes invoices that are not already settled.
- If LND sends the same settlement again, `MarkSettled` returns "nothing changed".
- Because nothing changed, the backend does not publish another `PaymentSettled` event.

## Test 4: Restart Recovery

This proves:

- If the backend is offline when the invoice is paid, it catches up after restart.

Create a new invoice while the backend is running:

```bash
export RECOVERY_INVOICE_RESPONSE=$(curl -sS -X POST "$API/tasks/$TASK_SLUG/donate" \
  -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -d '{"amount_sats": 777}')

echo "$RECOVERY_INVOICE_RESPONSE" | jq

export RECOVERY_BOLT11=$(echo "$RECOVERY_INVOICE_RESPONSE" | jq -r '.payment_request')
export RECOVERY_PAYMENT_HASH=$(echo "$RECOVERY_INVOICE_RESPONSE" | jq -r '.payment_hash')
```

Stop the backend with `Ctrl+C`.

Pay the invoice from the Polar Bob terminal while the backend is stopped:

```bash
lncli payinvoice --force "<paste RECOVERY_BOLT11 here>"
```

Start the backend again:

```bash
cd /home/frawuor/projects/personal/pamojabuild1/backend
go run ./cmd/app
```

Poll the status:

```bash
for i in $(seq 1 20); do
  curl -sS "$API/lightning/invoices/status?payment_hash=$RECOVERY_PAYMENT_HASH" \
    -b "$COOKIE_JAR" | jq
  sleep 1
done
```

Expected result:

- The invoice should become `settled` after restart.
- The ledger should include the extra `777` sats once.

Check the ledger:

```bash
curl -sS "$API/ledger/tasks/$TASK_SLUG" \
  -b "$COOKIE_JAR" | jq
```

What happened in the code:

- The backend stored a Lightning settlement cursor.
- On restart, the settlement listener asks LND for invoice updates after that cursor.
- LND sends the missed settlement.
- The backend processes it exactly like a live payment.

You can inspect the recovery cursor in PostgreSQL:

```bash
psql "$DATABASE_URL" \
  -c "SELECT key, value_integer, updated_at FROM lightning_sync_state;"
```

Expected result:

- You should see a row with key `lnd_settle_index`.
- `value_integer` should be greater than zero after settled payments.

## Test 5: Invoice Timeout

This proves:

- An unpaid invoice past its expiry time becomes `expired`.

The real invoice expiry is currently one hour, which is too long for a quick manual test. So we will manually move one invoice's `expires_at` into the past in the local test database.

Create a new invoice and do not pay it:

```bash
export EXPIRE_INVOICE_RESPONSE=$(curl -sS -X POST "$API/tasks/$TASK_SLUG/donate" \
  -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -d '{"amount_sats": 333}')

echo "$EXPIRE_INVOICE_RESPONSE" | jq

export EXPIRE_PAYMENT_HASH=$(echo "$EXPIRE_INVOICE_RESPONSE" | jq -r '.payment_hash')
```

Force it to be expired in the test database:

```bash
psql "$DATABASE_URL" \
  -c "UPDATE lightning_invoices SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 minute' WHERE payment_hash = '$EXPIRE_PAYMENT_HASH';"
```

Now ask the backend for status:

```bash
curl -sS "$API/lightning/invoices/status?payment_hash=$EXPIRE_PAYMENT_HASH" \
  -b "$COOKIE_JAR" | jq
```

Expected result:

```json
{
  "payment_hash": "...",
  "status": "expired",
  "settled": false
}
```

What happened in the code:

- `GetInvoiceStatus` first runs expiry cleanup.
- The repository changes old unpaid `pending` invoices to `expired`.
- Then the endpoint returns the updated status.

## Test 6: Invalid Donation Amount

This proves:

- The API rejects bad donation amounts before asking LND for an invoice.

Run:

```bash
curl -sS -i -X POST "$API/tasks/$TASK_SLUG/donate" \
  -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -d '{"amount_sats": 0}'
```

Expected result:

- HTTP status should be `400 Bad Request`.
- No LND invoice should be created.

## Test 7: Unknown and Invalid Payment Hashes

This proves:

- Invalid payment hash format is rejected.
- Valid-looking but unknown payment hash returns `404`.

Invalid format:

```bash
curl -sS -i "$API/lightning/invoices/status?payment_hash=not-a-hash" \
  -b "$COOKIE_JAR"
```

Expected result:

- HTTP status should be `400 Bad Request`.

Unknown but valid-looking hash:

```bash
export UNKNOWN_HASH="ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

curl -sS -i "$API/lightning/invoices/status?payment_hash=$UNKNOWN_HASH" \
  -b "$COOKIE_JAR"
```

Expected result:

- HTTP status should be `404 Not Found`.

## Useful Database Checks

Show invoices:

```bash
psql "$DATABASE_URL" \
  -c "SELECT payment_hash, task_slug, amount_sats, status, settled, settle_index, expires_at, settled_at FROM lightning_invoices ORDER BY created_at;"
```

Show Lightning sync cursor:

```bash
psql "$DATABASE_URL" \
  -c "SELECT key, value_integer, updated_at FROM lightning_sync_state;"
```

Show ledger entries:

```bash
psql "$DATABASE_URL" \
  -c "SELECT id, task_slug, entry_type, amount_sats, reference_id FROM ledger_entries ORDER BY id;"
```

Expected ledger behavior:

- Each paid invoice should create one `INBOUND_DONATION` ledger entry.
- Duplicate/replayed settlements should not create another entry for the same payment hash.

## Troubleshooting

### Bob cannot pay the invoice

Symptom:

- `lncli payinvoice` fails with a route or liquidity error.

Likely cause:

- Bob does not have spendable outbound liquidity toward Alice.

Fix:

- Make sure Bob and Alice are connected in Polar.
- Make sure there is a funded channel that lets Bob pay Alice.
- Create a new backend invoice after fixing the route.

### Backend cannot connect to LND

Symptom:

- Backend logs show TLS, macaroon, or connection errors.

Check:

```bash
echo "$LND_HOST"
echo "$LND_TLS_PATH"
test -n "$LND_MACAROON_HEX" && echo "macaroon hex is set"
docker exec "$POLAR_ALICE_CONTAINER" lncli getinfo
```

Common fixes:

- Use Alice's LND container, not Bob's.
- Use Alice's `tls.cert`, not Bob's.
- Use Alice's `admin.macaroon`, not Bob's.
- Make sure `LND_HOST` points to Alice's exposed `10009` gRPC port.

### Backend says invoice is still pending after Bob paid

Wait a few seconds, then poll again:

```bash
curl -sS "$API/lightning/invoices/status?payment_hash=$PAYMENT_HASH" \
  -b "$COOKIE_JAR" | jq
```

If it stays pending:

1. Check that Bob's payment actually succeeded.
2. Check backend logs for settlement listener errors.
3. Check the invoice exists in the database:

```bash
psql "$DATABASE_URL" \
  -c "SELECT payment_hash, status, settle_index FROM lightning_invoices WHERE payment_hash = '$PAYMENT_HASH';"
```

### Migration errors

Check the recorded version:

```bash
go run ./cmd/migrate version
```

Do not use `force` until you have inspected the failed migration and database schema. See [postgresql.md](postgresql.md).

## Cleanup

Stop the backend with `Ctrl+C`.

Remove the manual test database:

```bash
dropdb --if-exists pamoja_lightning_manual
```

Remove copied Polar certs:

```bash
rm -rf /home/frawuor/projects/personal/pamojabuild1/backend/.tmp/polar-lnd
```
