@# Lesson 2 — Databases

In Lesson 1 your data lived in a Go `map`. In this lesson it moves to
PostgreSQL, and survives a restart.

**By the end you will have:** a running Postgres database, a table you designed,
and the same API from Lesson 1 — same URLs, same `curl` commands — reading and
writing real, persistent data.

**We write SQL by hand.** No ORM. The goal is that you understand what's
actually happening, so that any tool you're handed later makes sense.

**Prerequisites:** [Lesson 1](./readme-lesson1-httpservices.md) complete, and
Docker installed ([main README §1](./README.md)).

---

## Part 0 — Why the map isn't enough

Start your Lesson 1 server, add an item, and confirm it's there:

```bash
curl -X POST http://localhost:8080/items \
  -H "Content-Type: application/json" \
  -d '{"name":"Mouse","price":25}'

curl http://localhost:8080/items
```

Now press **`Ctrl+C`**, start it again with `go run .`, and list the items.

Your mouse is gone.

That's the obvious problem. There are four more that matter just as much:

| Problem | Why it's fatal |
| --- | --- |
| **Data dies with the process** | Every deploy, crash, or restart wipes everything. |
| **Only one copy of your app can exist** | Run two instances for more traffic and they each have *different* data. |
| **Concurrent access crashes it** | That map isn't safe for simultaneous requests. It's a real bug in your Lesson 1 code, not a theoretical one. |
| **You can't ask questions** | "Items under $50, sorted by price, 20 at a time." With a map, you write that by hand, every time. |
| **No safety rules** | Nothing stops a negative price or a duplicate name. |

A database is a separate program whose entire job is solving those five
problems. Your app talks to it over the network — which also means your app can
restart, crash, or be replaced without the data noticing.

---

## Part 1 — The landscape

Before installing anything, a map of the territory. **This is awareness, not
homework** — you'll use exactly one of these today.

### The two questions

People say "SQL vs NoSQL" as if databases came in two flavours. They don't.
There are two independent questions:

**1. How is the data shaped?**

| Kind | Examples | Shape |
| --- | --- | --- |
| **Relational** | PostgreSQL, MySQL, SQLite | Tables with fixed columns. Rows link to other rows. |
| **Key-value** | Redis, Valkey, Memcached | One key, one value. Extremely fast, no querying. |
| **Document** | MongoDB | JSON-ish documents, no fixed shape required. |
| **Graph** | Neo4j | Nodes and the connections between them. |
| **Time-series** | InfluxDB, TimescaleDB, Prometheus | Measurements stamped with a time. |
| **Vector** | pgvector, Qdrant, Pinecone | Lists of numbers, searched by similarity. Powers AI features. |
| **Search** | Elasticsearch, Meilisearch, Typesense | Text, indexed for "find things like this". |

**2. What job does it do in your system?**

| Role | Meaning |
| --- | --- |
| **System of record** | The truth. If it's not here, it didn't happen. |
| **Cache** | A fast, disposable copy of data that lives elsewhere. |
| **Search index** | A copy reshaped for finding things. |
| **Queue / broker** | Not storage at all — a pipe between programs (RabbitMQ, Kafka, NATS). |
| **Object storage** | Files: images, PDFs, video (S3, MinIO). |

The second question matters more, and it's the one beginners skip.

> ### The rule worth memorising
>
> **One database holds the truth. Everything else is a copy you can throw away
> and rebuild.**
>
> You could delete your entire cache and your system would get slow, not wrong.
> Delete your system of record and you're finished.

### About Redis specifically

You'll hear "Postgres is persistent, Redis is a cache." That's a useful
shorthand and a slightly false statement — **Redis can save to disk** and can be
run as a system of record.

The accurate version: Redis is *treated* as disposable. You design so that
losing it is survivable, because rebuilding it from Postgres is always allowed.
The danger isn't that Redis loses data — it's storing something important
*only* in Redis.

Redis gets its own lesson later. Caching something you don't have yet is a
strange place to start.

### Where files go

The other question every beginner asks: *where do I put uploaded images?*

**Not in your database.** Put the file in object storage (S3, or MinIO locally)
and store the *URL* in Postgres. Databases are bad at large binary blobs — they
bloat backups and slow everything down.

### And now the punchline: just use Postgres

That table looks like five things to install. It isn't, because **Postgres does
most of them**:

| You want | Postgres answer |
| --- | --- |
| Document storage | `JSONB` columns — schemaless when you want it |
| Full-text search | Built in, and good enough for a long time |
| Vector / AI similarity | The `pgvector` extension |
| Time-series | The TimescaleDB extension |
| Simple pub/sub | `LISTEN` / `NOTIFY` |
| A job queue | A table with `SELECT ... FOR UPDATE SKIP LOCKED` |

"Use Postgres until it hurts" is a mainstream, well-respected default among
experienced engineers. Adding a new datastore for each feature means operating
five systems to serve a thousand users — and every one of them can break at 3am.

**Start with Postgres. Add a second thing only when you can name the specific
problem it solves.**

### One honourable mention: SQLite

A complete SQL database that lives in a single file. No server, no Docker, no
connection string. It is genuinely excellent, powers most phone apps, and is the
right answer for small projects and local tools.

We use Postgres because it's what you'll meet at work — but if you build
something personal later, SQLite deserves a look before you reach for a server.

---

## Part 2 — Install Postgres

### What Docker is, briefly

Installing Postgres directly on your machine means system services, config
files, and a mess to undo when it goes wrong.

Docker runs it in a **container** instead: an isolated, pre-built box with
Postgres and everything it needs already inside. You start it with one command,
you stop it with one command, and deleting it leaves no trace on your system.

Three terms:

| Term | Meaning |
| --- | --- |
| **Image** | The template — a downloadable "Postgres, ready to go" |
| **Container** | A running copy of an image |
| **Volume** | A folder Docker manages that survives the container being deleted. This is where your data actually lives. |

That last one is why your data won't disappear when you restart the container.

### The setup

In your project folder, create **`docker-compose.yml`**:

```yaml
services:
  db:
    image: postgres:17
    restart: unless-stopped
    environment:
      POSTGRES_USER: shop
      POSTGRES_PASSWORD: shoppass
      POSTGRES_DB: shop
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./schema.sql:/docker-entrypoint-initdb.d/schema.sql

volumes:
  pgdata:
```

Line by line:

| Line | Meaning |
| --- | --- |
| `image: postgres:17` | Which Postgres to download. Always pin a version — `latest` changes under you. |
| `environment:` | Postgres reads these on first start and creates the user, password, and database. |
| `ports: "5432:5432"` | Connect port 5432 on *your machine* to port 5432 *inside the container*. Without this, your Go program can't reach it. |
| `volumes: pgdata:...` | Where the data files live, outside the container. |
| `./schema.sql:/docker-entrypoint-initdb.d/...` | Any `.sql` file in that magic folder runs automatically the first time the database is created. |

### The schema

Create **`schema.sql`** next to it:

```sql
CREATE TABLE items (
    id    INT  GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name  TEXT NOT NULL,
    price INT  NOT NULL CHECK (price >= 0)
);

INSERT INTO items (name, price) VALUES
    ('Keyboard', 45),
    ('Monitor', 220);
```

This is your Lesson 1 `Item` struct, described to the database:

- **`GENERATED ALWAYS AS IDENTITY`** — Postgres assigns the id. You never send
  one. (You'll see `SERIAL` in older tutorials; this is the modern spelling.)
- **`PRIMARY KEY`** — unique, and the fast way to find one row.
- **`NOT NULL`** — this column cannot be empty. Ever.
- **`CHECK (price >= 0)`** — a rule the database enforces.

That last one is worth pausing on. You already validate price in your Go
handler, so why again here?

> **Because the database is the last line of defence.** Your handler is one way
> in. Later there will be a second service, an admin script, someone typing SQL
> at 2am. A `CHECK` constraint holds for all of them. Validate in your app for a
> friendly error message; constrain in the database so bad data is *impossible*.

### Start it

```bash
docker compose up -d
```

`-d` means detached — it runs in the background. First run downloads the image;
give it a minute.

```bash
docker compose ps        # is it running?
docker compose logs db   # what did it say?
```

You're looking for `database system is ready to accept connections`.

Useful commands from here on:

| Command | Effect |
| --- | --- |
| `docker compose up -d` | Start |
| `docker compose stop` | Stop, keep data |
| `docker compose down` | Remove the container, keep data |
| `docker compose down -v` | Remove the container **and delete all data** |

> ### The gotcha that will get you
>
> **`schema.sql` only runs when the data directory is empty** — i.e. the very
> first start.
>
> Edit `schema.sql` later and restart, and nothing happens. Your change is
> ignored and you'll lose twenty minutes wondering why. To force it during
> development:
>
> ```bash
> docker compose down -v && docker compose up -d
> ```
>
> That wipes everything and starts clean. Fine now, obviously catastrophic in
> production — which is exactly the problem **migrations** solve in Part 6.

---

## Part 3 — SQL before Go

Talk to the database directly first. Learning SQL and the Go plumbing
simultaneously means two unfamiliar things at once; if a query misbehaves later,
you want to know which half is wrong.

```bash
docker compose exec db psql -U shop -d shop
```

That runs `psql` — Postgres's built-in terminal client — inside the container.
Your prompt becomes `shop=#`.

> Prefer a GUI? Point GoLand's database tool or DBeaver at
> `localhost:5432`, database `shop`, user `shop`, password `shoppass`. Same
> thing with more clicking.

### The four statements you need

**Read:**

```sql
SELECT * FROM items;
SELECT id, name FROM items;
SELECT * FROM items WHERE price < 100;
SELECT * FROM items ORDER BY price DESC;
SELECT * FROM items WHERE id = 1;
```

`SELECT *` means "every column". Handy while exploring, **bad in real code** —
name your columns explicitly, so adding a column later doesn't silently change
what your program receives.

**Create:**

```sql
INSERT INTO items (name, price) VALUES ('Mouse', 25);
```

No `id` — the database fills it in. Ask for it back with `RETURNING`:

```sql
INSERT INTO items (name, price) VALUES ('Webcam', 60) RETURNING id;
```

**Update:**

```sql
UPDATE items SET price = 50 WHERE id = 1;
```

**Delete:**

```sql
DELETE FROM items WHERE id = 3;
```

> **`WHERE` is not optional.** `UPDATE items SET price = 0;` sets *every* item
> to zero. `DELETE FROM items;` empties the table. Both are instant, both are
> silent, and neither asks if you're sure. Write the `WHERE` clause first, then
> go back and write the rest.

### Try breaking your own rules

```sql
INSERT INTO items (name, price) VALUES ('Broken', -5);
```

```
ERROR:  new row for relation "items" violates check constraint "items_price_check"
```

```sql
INSERT INTO items (price) VALUES (10);
```

```
ERROR:  null value in column "name" violates not-null constraint
```

The database refused. It doesn't care that your Go code is careful — the rule
holds regardless.

Useful `psql` shortcuts:

| Command | Does |
| --- | --- |
| `\dt` | List tables |
| `\d items` | Describe the `items` table |
| `\q` | Quit |

---

## Part 4 — Connect from Go

### The driver

Go's standard library has no Postgres support. You need a **driver** — a package
that speaks Postgres's network protocol.

```bash
go get github.com/jackc/pgx/v5
```

<details>
<summary><strong>Why pgx directly, and not <code>database/sql</code>?</strong></summary>

You'll meet two styles in the wild:

- **`database/sql`** — a generic interface in the standard library. Works with
  any database that has a driver. Portable, but every Postgres-specific feature
  has to squeeze through a lowest-common-denominator API.
- **pgx native** — talks to Postgres directly. Faster, full support for
  Postgres types (arrays, JSONB, ranges), better errors.

We use pgx native because this lesson is about understanding Postgres, not
abstracting over it.

The good news: they're barely different. `database/sql` uses `?` or `$1`
placeholders, `QueryRow(...).Scan(...)`, `rows.Next()` — all concepts you're
about to learn transfer directly. If you need `database/sql` later, pgx supports
that too, via `github.com/jackc/pgx/v5/stdlib`.
</details>

### Connecting

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://shop:shoppass@localhost:5432/shop"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("cannot configure database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("cannot reach database: %v", err)
	}
	log.Println("connected to database")
}
```

Run it. You should see `connected to database`.

Now stop the database (`docker compose stop`) and run it again:

```
cannot reach database: failed to connect to `host=localhost user=shop database=shop`: dial error
```

Start it back up (`docker compose up -d`) before continuing.

### Three things in that snippet matter

**1. `pgxpool.New` does not connect.**

It parses the connection string and prepares a pool. It errors on a *malformed*
string, not an unreachable database. **`Ping` is what actually connects** —
without it, a wrong password surfaces later, during some unrelated request, far
from the cause.

Always `Ping` at startup. Fail loudly and immediately.

**2. It's a pool, not a connection.**

`pool` holds several connections and hands them out as needed. Your server
handles many requests at once; each needs its own connection while it's working.
The pool manages that for you — but it explains a bug you'll meet in Part 5: if
you borrow a connection and forget to give it back, the pool eventually runs
dry and your app hangs with no error at all.

**3. The connection string doesn't belong in your code.**

```
postgres://shop:shoppass@localhost:5432/shop
 ─────── ─────── ──────── ───────── ────
 scheme   user   password   host:port  database
```

Reading `DATABASE_URL` from the environment with a local fallback means
production credentials never touch your source code. `os.Getenv` returns `""`
when unset — the same "absent is empty" pattern as query parameters in Lesson 1.

### Context, briefly

Every pgx call takes a `context.Context` as its first argument. A context
carries **"should I still be doing this?"**.

`context.Background()` means "no deadline, nobody's going to cancel this" — fine
at startup. But inside a handler, use **`r.Context()`**, which Go cancels
automatically when the client disconnects.

Why that's worth doing: someone requests a slow report and closes the tab. With
`r.Context()`, Postgres is told to abandon the query. Without it, the database
keeps grinding away on a result nobody will ever read — and under load, that's
how a service falls over.

**Rule: `context.Background()` in `main`, `r.Context()` in handlers.**

---

## Part 5 — Rewrite the handlers

Same endpoints, same JSON, different storage. Delete the `items` map.

### One change to the struct

```go
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}
```

`ID` becomes an `int`, because the database generates numeric ids. That has a
knock-on effect: `r.PathValue("id")` gives you a string, so handlers now have to
convert — and reject junk.

> **Why `Price` is an `int`.** It's a count of *cents*, not dollars. **Never
> store money in a float.** `0.1 + 0.2` is not `0.3` in binary floating point,
> and those errors accumulate into real accounting problems. Integers of the
> smallest unit, or Postgres's `NUMERIC` type. Never `float`.

### Reading one row

```go
func handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	var item Item
	err = db.QueryRow(r.Context(),
		`SELECT id, name, price FROM items WHERE id = $1`, id).
		Scan(&item.ID, &item.Name, &item.Price)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get item %d: %v", id, err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}
```

Four new ideas here.

**`$1` is a placeholder.** The `id` value travels to Postgres *separately* from
the query text. See the security note below — this is the most important
paragraph in the lesson.

**`Scan` fills your struct.** It takes pointers — `&item.ID` — for exactly the
reason covered in [Lesson 1's struct interlude](./readme-lesson1-httpservices.md#interlude--structs):
it needs to write into *your* variables, not copies. **The order must match the
`SELECT`.** Swap two columns in the query and forget to swap them in `Scan`, and
you get a confusing type error — or worse, silently wrong data if both columns
are the same type.

**`pgx.ErrNoRows` is your new 404.** In Lesson 1 you checked `ok` from a map
lookup. Now, "no such row" arrives as an error, and it's the *expected* kind:

```go
item, ok := items[id]          // Lesson 1
if !ok { /* 404 */ }

err := db.QueryRow(...).Scan(...)   // Lesson 2
if errors.Is(err, pgx.ErrNoRows) { /* 404 */ }
```

Use `errors.Is`, not `err == pgx.ErrNoRows`. Errors are often wrapped in layers,
and `errors.Is` looks through the wrapping.

**Two different error paths.** "Not found" is a `404`; anything else is a `500`.
Note what the `500` branch does:

```go
log.Printf("get item %d: %v", id, err)                              // full detail → your logs
http.Error(w, "something went wrong", http.StatusInternalServerError) // vague → the client
```

**Never send a database error to a client.** Those messages contain table names,
column names, and sometimes fragments of your query — a free schema map for
anyone probing your service. Detail goes to your logs, where you need it.

### 🔴 SQL injection — read this twice

Here is the same query written the wrong way:

```go
// NEVER. NOT ONCE. NOT FOR A QUICK TEST.
query := fmt.Sprintf("SELECT id, name, price FROM items WHERE name = '%s'", name)
rows, err := db.Query(r.Context(), query)
```

Looks harmless. Now someone calls your endpoint with:

```
'; DROP TABLE items; --
```

The string you assembled becomes:

```sql
SELECT id, name, price FROM items WHERE name = ''; DROP TABLE items; --'
```

Your table is gone. Or, more quietly, `' OR '1'='1` returns every row in a table
they should never have seen.

**The fix is the thing you're already doing:**

```go
db.Query(r.Context(), `SELECT id, name, price FROM items WHERE name = $1`, name)
```

With `$1`, the query text and the value travel to Postgres as **separate
things**. Postgres parses the query first, *then* slots the value in as data. It
is never parsed as SQL, so there is nothing to inject. The user could type the
entire Postgres manual and it would just be a very long name that matches
nothing.

> **The rule, no exceptions:** values go in placeholders. If you are building a
> query string with `+`, `fmt.Sprintf`, or a template, stop.
>
> (Placeholders work for *values* only — you can't parameterise a table or
> column name. If you ever need dynamic column names, they must come from a
> hardcoded allowlist you wrote, never from user input.)

### Reading many rows

```go
func handleListItems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(r.Context(),
		`SELECT id, name, price FROM items ORDER BY id`)
	if err != nil {
		log.Printf("list items: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Price); err != nil {
			log.Printf("scan item: %v", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate items: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
```

The shape is always the same: **query → defer close → loop → check `rows.Err()`.**

**`defer rows.Close()` is not optional.** `rows` is holding a connection from
the pool. Skip the close and that connection is never returned — do it enough
times and the pool is empty, and your app hangs forever with no error message.
`defer` runs it whichever way the function exits.

**`rows.Err()` after the loop is the step everyone forgets.** `rows.Next()`
returns `false` for two different reasons: you reached the end, *or* something
broke halfway. Without that check, a connection dropped mid-read looks exactly
like a successful empty result — you'd return `200 OK` and a short list, and
never know.

**`make([]Item, 0)` not `var items []Item`.** A nil slice encodes to JSON as
`null`; an empty slice encodes as `[]`. Clients parsing `null` as an array
break. Same reason as Lesson 1.

<details>
<summary><strong>The shortcut, now that you know the long way</strong></summary>

pgx can do that entire loop for you:

```go
rows, err := db.Query(r.Context(), `SELECT id, name, price FROM items ORDER BY id`)
if err != nil { /* 500 */ }

items, err := pgx.CollectRows(rows, pgx.RowToStructByName[Item])
if err != nil { /* 500 */ }
```

It matches columns to struct fields by name, closes `rows`, and checks
`rows.Err()` for you.

Use it in real code. But it's worth having written the loop once, so you know
what it's doing — and so a scan error at 3am isn't magic.
</details>

### Writing

```go
func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var item Item

	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if item.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if item.Price < 0 {
		http.Error(w, "price cannot be negative", http.StatusBadRequest)
		return
	}

	err := db.QueryRow(r.Context(),
		`INSERT INTO items (name, price) VALUES ($1, $2) RETURNING id`,
		item.Name, item.Price).Scan(&item.ID)
	if err != nil {
		log.Printf("create item: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}
```

**`RETURNING id` earns its keep.** The database generates the id, and you need
it to send back. `RETURNING` gets it in the same round trip — one query, not
two, and no race with anyone else inserting at the same moment. It's a Postgres
feature, and a genuinely nice one.

Because it returns a row, you use `QueryRow(...).Scan(...)` even though this is
an insert.

Note that the id the client sends is still ignored — the database decides. Same
principle as Lesson 1.

### Deleting

```go
func handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	tag, err := db.Exec(r.Context(), `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		log.Printf("delete item %d: %v", id, err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if tag.RowsAffected() == 0 {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**`Exec` when no rows come back** — deletes and updates. It returns a *command
tag* instead, and `RowsAffected()` tells you how many rows changed.

That's how you get a 404 here: deleting a row that doesn't exist is not an
error in SQL. It succeeds, having deleted nothing. Only `RowsAffected() == 0`
tells you the difference.

`204 No Content` means "it worked, and there's nothing to send back". Don't
write a body after it.

### Which method to use

| Method | Use for | Returns |
| --- | --- | --- |
| `QueryRow` | Exactly one row (`SELECT` by id, `INSERT ... RETURNING`) | A row you `Scan`; `pgx.ErrNoRows` if absent |
| `Query` | Many rows | `rows` you loop over — **must** `Close` |
| `Exec` | No rows back (`UPDATE`, `DELETE`, DDL) | A tag with `RowsAffected()` |

### Health check, upgraded

```go
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(r.Context()); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "ok")
}
```

In Lesson 1 `/health` proved the Go process was alive. Now it proves the service
can actually *do its job*. A service that's running but can't reach its database
is not healthy, and you want your monitoring to know that.

---

## The complete program

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var db *pgxpool.Pool

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://shop:shoppass@localhost:5432/shop"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("cannot configure database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("cannot reach database: %v", err)
	}
	log.Println("connected to database")

	db = pool

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /items", handleListItems)
	mux.HandleFunc("GET /items/{id}", handleGetItem)
	mux.HandleFunc("POST /items", handleCreateItem)
	mux.HandleFunc("DELETE /items/{id}", handleDeleteItem)

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(r.Context()); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "ok")
}

func handleListItems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(r.Context(),
		`SELECT id, name, price FROM items ORDER BY id`)
	if err != nil {
		log.Printf("list items: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Price); err != nil {
			log.Printf("scan item: %v", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate items: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	var item Item
	err = db.QueryRow(r.Context(),
		`SELECT id, name, price FROM items WHERE id = $1`, id).
		Scan(&item.ID, &item.Name, &item.Price)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get item %d: %v", id, err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var item Item

	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if item.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if item.Price < 0 {
		http.Error(w, "price cannot be negative", http.StatusBadRequest)
		return
	}

	err := db.QueryRow(r.Context(),
		`INSERT INTO items (name, price) VALUES ($1, $2) RETURNING id`,
		item.Name, item.Price).Scan(&item.ID)
	if err != nil {
		log.Printf("create item: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	tag, err := db.Exec(r.Context(), `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		log.Printf("delete item %d: %v", id, err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if tag.RowsAffected() == 0 {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

### The moment that matters

```bash
curl -X POST http://localhost:8080/items \
  -H "Content-Type: application/json" \
  -d '{"name":"Mouse","price":25}'

curl http://localhost:8080/items
```

Now stop the server with `Ctrl+C`. Start it again. List the items.

**The mouse is still there.** That's the whole lesson.

Then go further — stop the *database* too:

```bash
docker compose down     # note: no -v, so the volume survives
docker compose up -d
go run .
curl http://localhost:8080/items
```

Still there. The data outlived both programs.

---

## Part 6 — Migrations

You have a problem you may not have noticed.

Your table exists because `schema.sql` ran once, on an empty volume. Now try
adding a column: edit `schema.sql`, restart, and… nothing. The file is ignored
on every start after the first.

Your options right now are to type `ALTER TABLE` into `psql` by hand — with
nothing recording that you did it — or to wipe the database and start over.
Neither works with other people, and neither works in production.

**A migration is a schema change saved as a file, applied in order, exactly
once.** Instead of one `schema.sql` that describes the current shape, you keep a
numbered history of changes:

```
migrations/
  001_create_items.sql
  002_add_items_description.sql
  003_add_items_created_at.sql
```

A migration tool keeps a small table in your database recording which files have
run. On startup it applies the ones that haven't, in order. That gives you:

- **Reproducibility** — a fresh database becomes the current one by replaying history
- **Teamwork** — a colleague pulls your branch, runs migrations, has your schema
- **Deployment** — the same command runs in production
- **A record** — schema changes live in git, reviewable like code

The common Go tools:

| Tool | Notes |
| --- | --- |
| **golang-migrate** | Most widely used. CLI plus a Go library. |
| **goose** | Very popular. Supports Go-code migrations, not just SQL. |
| **atlas** | Newer. Can generate migrations by diffing schemas. |
| **tern** | From pgx's author. Postgres-only, minimal. |

Any of them is fine. Pick one and keep it.

> **Two rules that will save you.**
>
> **1. Never edit a migration that has already run somewhere.** Your machine has
> applied it; the tool thinks it's done and will not re-run it. Your colleague's
> machine hasn't, and applies your edited version. You now have two different
> schemas that both believe they're up to date. Always add a new migration
> instead.
>
> **2. Write the rollback, and test it.** Most tools want an `up` and a `down`
> for each migration. The one time you need `down` is a bad deploy at 5pm, which
> is the worst possible moment to discover it doesn't work.

We'll wire up a real migration tool in a later lesson. For now, know that
`schema.sql` in `docker-entrypoint-initdb.d` is a learning shortcut, not
something you'd ship.

---

## Part 7 — What's still wrong

Honest list, and the lesson that fixes each:

| Problem | Why it matters | Fixed in |
| --- | --- | --- |
| `db` is a package-level global | Any code anywhere can reach the database. Impossible to swap for tests. | Lesson 3 — structuring code |
| SQL is scattered through handlers | Handlers do HTTP *and* persistence. Change the schema, hunt through every handler. | Lesson 3 |
| `main.go` is ~180 lines | Still readable. Won't be at 500. | Lesson 3 |
| No tests | And you can't write good ones while handlers talk straight to a global pool. | Lesson 4 |
| `GET /items` returns everything | Fine with 3 rows, fatal with 3 million. Needs `LIMIT`/`OFFSET`. | Lesson on pagination |
| No transactions | Two writes that must both succeed or both fail can't be expressed yet. | Lesson on transactions |
| No index beyond the primary key | `WHERE name = ...` scans every row. | Lesson on performance |
| Password sits in `docker-compose.yml` | Fine locally. Not fine anywhere else. | Lesson on configuration |

That first one is the thread Lesson 3 pulls on. A global `db` works fine today
and quietly makes everything else harder — which is exactly the kind of problem
architecture exists to solve.

---

## Exercises

1. **`PUT /items/{id}`** — update an item. Use `Exec` and `RowsAffected()` to
   return `404` when the id doesn't exist.
2. **`GET /items?max=100`** — filter by price. Remember: the value goes in a
   `$1` placeholder, never into the query string. Decide what `?max=banana`
   should do.
3. **Add a `description` column.** Do it properly: `ALTER TABLE` in `psql`, then
   update your `SELECT` column lists and `Scan` calls. Notice how many places
   you had to touch — that's the pain Lesson 3 addresses.
4. **Make `name` unique.** Add `UNIQUE` to the column, then insert a duplicate
   through your API. You'll get a `500`. It should be a `409 Conflict` — look up
   `pgconn.PgError` and error code `23505`.
5. **Break the pool on purpose.** Delete `defer rows.Close()` from
   `handleListItems`, then hit `GET /items` repeatedly. Count how many requests
   it takes before your app stops responding entirely — with no error printed.
   This is the single most instructive failure in the lesson.
6. **Try the injection.** Rewrite one query with `fmt.Sprintf`, then call it with
   `' OR '1'='1`. See it return everything. Then put the placeholder back.

---

## What you learned

- Why in-memory data fails: restarts, multiple instances, concurrency, querying
- The database landscape — and that **Postgres covers most of it**
- One system of record; everything else is a rebuildable copy
- Running Postgres in Docker, with a volume so data survives
- SQL: `SELECT`, `INSERT ... RETURNING`, `UPDATE`, `DELETE`, and why `WHERE` is
  never optional
- Constraints (`NOT NULL`, `CHECK`) as a last line of defence under your Go
  validation
- `pgxpool.New` doesn't connect — **always `Ping` at startup**
- `QueryRow` / `Query` / `Exec`, and when each applies
- `pgx.ErrNoRows` is the new "not found"
- `defer rows.Close()` and `rows.Err()` — the two lines everyone forgets
- **Placeholders, always.** Never build SQL with string formatting.
- Log the real error; send the client a vague one
- Money is an integer of cents, never a float
- What migrations are and why `schema.sql` doesn't scale
