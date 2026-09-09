# Lesson 3a — The Words People Use

> **Prerequisites:** [Lesson 1 — HTTP Services](./readme-lesson1-httpservices.md)
> and [Lesson 2 — Databases](./readme-lesson2-database.md). This lesson refers to
> the code you wrote there constantly.

You now have a working service: HTTP in, SQL out, JSON back. It works, and it
has problems you already know about — a global `db`, handlers that do five jobs
each, SQL strings sitting next to `http.ResponseWriter`.

Before you fix any of that, you need the vocabulary. Not because vocabulary is
important on its own, but because every article, code review, and job interview
about "how to structure a Go service" is written in it. Right now those
conversations are noise. After this lesson they're sentences.

**This lesson is reading, not typing.** There's one short exercise at the end.

---

## Part 0 — Why this feels confusing

The words feel confusing because four completely different kinds of thing get
listed together, as if they were alternatives to each other:

| Category | Examples | What it actually is |
| --- | --- | --- |
| **Heuristics** | DRY, KISS, YAGNI, DAMP | Judgment calls. Advice, not rules. |
| **Principles** | SOLID | Rules about how types depend on each other. |
| **Roles** | repository, usecase, entity, DTO, handler | Names for *jobs a piece of code does*. |
| **Structures** | layered, Clean, Hexagonal, MVC, microservices | Ways to arrange those roles. |

They're not competing options. You can use all four at once — a **layered**
structure, containing a **repository**, following **dependency inversion**,
kept **simple**.

Someone asking *"should I learn DDD or microservices first?"* is asking
*"should I learn adjectives or paragraphs first?"* The question doesn't parse,
which is exactly why it's hard to answer.

**This lesson covers the first three: heuristics, principles, roles.**
[Lesson 3b](./readme-lesson3-intermediate-b-structures.md) covers structures.

---

## Part 1 — Heuristics

A heuristic is advice that's usually right. Every one of them has a situation
where following it makes your code worse. Anyone who states one as a law hasn't
been burned by it yet.

---

### SoC — Separation of Concerns

**One piece of code should have one job.**

This is the parent of almost every other idea in this lesson. Layers,
repositories, Clean Architecture — all of them are SoC applied at different
sizes.

Here's `handleCreateItem` from Lesson 2, with its concerns labelled:

```go
func handleCreateItem(w http.ResponseWriter, r *http.Request) {
    var item Item
    if err := json.NewDecoder(r.Body).Decode(&item); err != nil {  // 1. HTTP
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    if item.Name == "" || item.Price < 0 {                          // 2. rules
        http.Error(w, "name required, price must be >= 0", http.StatusBadRequest)
        return
    }

    err := db.QueryRow(r.Context(),                                 // 3. storage
        `INSERT INTO items (name, price) VALUES ($1, $2) RETURNING id`,
        item.Name, item.Price).Scan(&item.ID)
    if err != nil {
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")              // 4. HTTP
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(item)
}
```

Four concerns, one function. That's not a disaster at this size — it's about 20
lines and you can read all of it. But notice what it costs you:

- You can't test rule 2 without starting an HTTP server *and* a database.
- If you add a mobile-app endpoint that also creates items, you copy rules.
- If you switch from Postgres to something else, you edit a file whose name
  says `http` in it.

**Splitting those four concerns apart is the entire content of Lesson 4.**
Everything else in this lesson is vocabulary for doing it.

---

### DRY — Don't Repeat Yourself

**Every piece of knowledge should have one home.**

Note what DRY actually says. It says *knowledge*, not *characters*. This
distinction is the whole game, and it's the most misquoted idea in software.

**Genuine duplication** — from your Lesson 2 handlers:

```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusNotFound)
json.NewEncoder(w).Encode(map[string]string{"error": "item not found"})
```

That block appears three times. And it encodes one decision: *"this is how our
API reports errors."* If you decide errors should include a `code` field, you
edit three places and forget one. Extract it:

```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]string{"error": msg})
}
```

Good DRY. One decision, one home.

**Fake duplication** — `handleGetItem` and `handleDeleteItem` both start with:

```go
id, err := strconv.Atoi(r.PathValue("id"))
if err != nil {
    writeError(w, http.StatusBadRequest, "invalid id")
    return
}
```

Same characters. But is it the same *knowledge*? Sort of — "ids are integers."
Extracting it into a helper that returns `(int, bool)` is reasonable. But the
tempting next step is not:

```go
// Don't do this.
func handleItemByID(w http.ResponseWriter, r *http.Request, isDelete bool) {
    // ...
    if isDelete {
        // ...
    } else {
        // ...
    }
}
```

Two functions that *looked* alike are now one function with a mode flag. It's
shorter and worse: you can't read either path without mentally deleting the
other, and when GET grows a `?fields=` parameter that DELETE doesn't have, the
flag becomes two flags.

**The counterweights, worth memorising:**

| Saying | Source | Meaning |
| --- | --- | --- |
| "Duplication is far cheaper than the wrong abstraction." | Sandi Metz | An unwanted copy is a local problem. A bad shared function is everyone's problem. |
| "A little copying is better than a little dependency." | [Go Proverbs](https://go-proverbs.github.io) | Especially across package boundaries. |
| **Rule of three** | folklore | Wait until you see it a *third* time. Two occurrences aren't a pattern yet. |
| **AHA** — Avoid Hasty Abstractions | Kent C. Dodds | Prefer duplication over the wrong abstraction, and wait for the right one to become obvious. |

Undoing duplication is a five-minute job. Undoing an abstraction that eleven
files depend on is a Tuesday.

---

### KISS — Keep It Simple, Stupid

**Prefer the boring solution.**

The catch: *simple* is not the same as *short*, and it's not the same as
*easy*. A 400-line file with no indirection can be simpler than 12 files of
interfaces, because you can read it top to bottom.

In Go this shows up as a constant question: **do I need an interface here?**
Usually no. An interface with exactly one implementation and no test double is
pure ceremony — a layer of indirection you pay for on every jump-to-definition
and get nothing back for.

Lesson 4 introduces exactly one interface, and only after showing you the
specific pain it removes.

---

### YAGNI — You Aren't Gonna Need It

**Don't build for a requirement you don't have.**

The trap is that YAGNI-violations always feel responsible. "I'll add an
interface so we can swap Postgres for MongoDB later." You won't swap it. And
the abstraction you invent today, without knowing what the second database
actually needs, will be wrong for it anyway.

| Feels responsible | Reality |
| --- | --- |
| Interface in case we swap databases | You won't, and it'd be the wrong interface if you did |
| Config for a feature nobody enabled | Dead code with a switch on it |
| Generic `Repository[T]` for one table | Solving a problem you don't have |
| Kafka "for when we scale" | You have 40 requests a day |

**YAGNI vs. SoC.** These pull in opposite directions, and that's fine.
Separating concerns is worth doing when the concerns exist *now* — testing is
hard *now*, the handler is unreadable *now*. Doing it for a future that may not
arrive is YAGNI.

---

### DAMP — Descriptive And Meaningful Phrases

**In tests, favour readability over DRY.**

The one place where duplication is actively good. A test should be readable
top-to-bottom, with no jumping to helper functions to find out what it's
actually asserting. Copy the setup. Spell out the values.

Full treatment in the testing lesson — it's here because it's on every list of
these acronyms and you'll wonder why it contradicts DRY.

---

### Go's own proverbs

Rob Pike's [Go Proverbs](https://go-proverbs.github.io) are the local dialect.
These four decide real arguments:

| Proverb | What it means for you |
| --- | --- |
| "A little copying is better than a little dependency." | Copy 10 lines rather than import a package for them. |
| "Clear is better than clever." | Boring code that reads plainly beats compact code that impresses. |
| "The bigger the interface, the weaker the abstraction." | 1–2 methods is a good interface. 9 methods is a class with extra steps. |
| "Don't communicate by sharing memory; share memory by communicating." | A concurrency rule. Filed away for later. |

There's a fifth, and it's the one that will change how you write Go:

> **"Accept interfaces, return structs."**

Your *constructor* returns a concrete `*PostgresItemStore`. Your *consumer*
accepts a small `ItemStore` interface. It means the caller decides what
abstraction it needs, instead of the package author guessing. Lesson 4 is built
on it.

---

## Part 2 — SOLID, honestly

Five principles from the object-oriented world (Robert C. Martin, ~2000). Go
isn't that world — no classes, no inheritance — so they don't all land. Here's
the honest scorecard:

| | Name | Verdict in Go |
| --- | --- | --- |
| **S** | Single Responsibility | ✅ **Matters.** Same idea as SoC. |
| **O** | Open/Closed | ⚠️ Mostly about inheritance. Weak here. |
| **L** | Liskov Substitution | ❌ Barely applies. No inheritance to get wrong. |
| **I** | Interface Segregation | ✅ **Matters a lot.** Go's culture is built on it. |
| **D** | Dependency Inversion | ✅ **The important one.** Everything in 3b depends on it. |

Learn S, I, and D. Recognise L and O so you know what people mean.

---

### I — Interface Segregation

**Don't force code to depend on methods it doesn't call.**

Suppose Lesson 4 gave you one big interface:

```go
// Too big.
type Store interface {
    ListItems(ctx context.Context) ([]Item, error)
    GetItem(ctx context.Context, id int) (Item, error)
    CreateItem(ctx context.Context, i Item) (Item, error)
    DeleteItem(ctx context.Context, id int) error
    ListUsers(ctx context.Context) ([]User, error)
    CreateOrder(ctx context.Context, o Order) (Order, error)
    // ...twelve more
}
```

A fake for one test now needs eighteen methods, seventeen of which do nothing.
Split by what each consumer actually uses:

```go
type ItemReader interface {
    GetItem(ctx context.Context, id int) (Item, error)
}
```

One method. A test fake is four lines. This is what "the bigger the interface,
the weaker the abstraction" means in practice — and it's why Go interfaces in
the standard library are mostly one method (`io.Reader`, `io.Writer`,
`error`, `http.Handler`).

**Go's superpower here:** interfaces are satisfied *implicitly*. There's no
`implements` keyword. So the interface can be declared in the package that
*consumes* it, next to the code that needs it — and the implementation, which
doesn't even import that package, satisfies it for free. In Java, the
implementation must name the interface. In Go, it doesn't have to know it
exists.

---

### D — Dependency Inversion

**Depend on an abstraction, not on a specific thing.**

This is the principle that Clean Architecture, Hexagonal Architecture, DDD's
repository pattern, and every testing strategy are all built on. Here it is in
your code.

Right now:

```go
func handleGetItem(w http.ResponseWriter, r *http.Request) {
    var item Item
    err := db.QueryRow(r.Context(), `SELECT ...`).Scan(...)
}
```

Your handler depends on `*pgxpool.Pool` — a specific thing. To run this
function at all, Postgres must exist.

Inverted:

```go
type ItemGetter interface {
    GetItem(ctx context.Context, id int) (Item, error)
}
```

Now the handler depends on `ItemGetter` — an abstraction. Postgres satisfies
it. So does a `map[int]Item` in a test. So would a file, or an HTTP call to
another service.

**Why "inversion"?** Before, the arrow pointed from your logic *down* to the
database. After, both your logic and the Postgres code point at the interface
in the middle. The database's arrow got turned around:

```
Before:   handler ──────────────► pgxpool

After:    handler ──► ItemGetter ◄────── PostgresStore
```

That flipped arrow is the whole idea. Everything in Lesson 3b is a diagram of
it drawn at a different size.

---

## Part 3 — Roles: the nouns

These aren't design philosophies. They're **names for jobs**. When someone says
"put that in the repository," they mean one specific kind of file.

| Role | Job | Also called |
| --- | --- | --- |
| **Handler** | Translate HTTP ↔ Go. Parse, validate shape, set status codes. | Controller, resource, endpoint |
| **Usecase** | One business operation, start to finish. No HTTP, no SQL. | Service, interactor, application service |
| **Repository** | Read and write stored data. Hides the fact that it's SQL. | Store, DAO, gateway, persistence |
| **Entity** | The thing itself — an `Item`, an `Order`. | Domain model, model, aggregate |
| **DTO** | A shape used only for transport in or out. | Request/response type, view model |
| **Presenter** | Turn a result into an output format. | Serializer, view, responder |

Where your current code sits:

| Your code | Role it's playing |
| --- | --- |
| `handleCreateItem` | Handler **and** usecase **and** repository, all at once |
| `type Item struct` | Entity **and** DTO, all at once |
| `db.QueryRow(...)` inside the handler | Repository work, done in the wrong place |
| `main()` | Composition root (see below) — this one's already correct |

That's not a failure. It's the *normal* shape of a small service, and it's the
right shape until it hurts. Lesson 4 is the moment it hurts.

---

### Three honest notes

**"Repository" means two different things.** In DDD it's a strict thing: it
deals in whole domain objects, and the domain layer can't tell there's a
database behind it. In everyday usage it means "the file with the SQL in it."
You'll meet both. Nobody will tell you which one they mean.

**"Entity vs DTO" only matters once they differ.** Right now your `Item` is
both — it's what the database returns *and* what the JSON contains. The day the
database gains a `deleted_at` column that the API must not expose, they split
into two types. Splitting them before that day is YAGNI.

**"Presenter" is probably not for you.** It's the fourth box in Clean
Architecture, and it exists because Uncle Bob's examples output to a UI. Your
output is `json.Encoder`. In a JSON API, the presenter is a struct tag. Don't
build a layer for it.

---

### DI — Dependency Injection

The most intimidating name for the least intimidating idea in software.

**Dependency injection means: pass the thing in, instead of reaching out for
it.** That's it. That's the whole concept.

Your Lesson 2 code *reaches out*:

```go
var db *pgxpool.Pool                              // package-level global

func handleGetItem(w http.ResponseWriter, r *http.Request) {
    db.QueryRow(...)                              // reaches out and grabs it
}
```

Injected:

```go
type ItemHandler struct {
    db *pgxpool.Pool                              // handed to us
}

func NewItemHandler(db *pgxpool.Pool) *ItemHandler {
    return &ItemHandler{db: db}
}

func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
    h.db.QueryRow(...)                            // uses what it was given
}
```

And in `main`:

```go
pool, err := pgxpool.New(ctx, dsn)
// ...
h := NewItemHandler(pool)                         // the "injection"

mux := http.NewServeMux()
mux.HandleFunc("GET /items/{id}", h.Get)
```

**That's the injection.** Passing an argument to a function. No framework, no
annotations, no container, no XML.

What it buys you is real, though:

| Before | After |
| --- | --- |
| Every handler silently uses the same global | Each handler's needs are visible in its constructor |
| Two databases (main + read replica) = impossible | Two `ItemHandler`s with different pools |
| Test = start a real Postgres | Test = pass whatever you want |
| Init order is implicit and fragile | `main` builds things in dependency order, or won't compile |

<details>
<summary>What about wire, fx, dig?</summary>

Those are **DI containers** — tools that write the wiring for you when `main`
gets long. Google's `wire` generates the code at build time; Uber's `fx` and
`dig` do it at runtime with reflection.

They solve a problem you get at roughly 50+ constructors. You have four. Wiring
by hand in `main` is the normal, recommended Go approach, and plenty of large
production services never outgrow it.

If you take one thing from this: **DI is the pattern, a DI container is an
optional tool.** People conflate them, then conclude DI is complicated.
</details>

---

### Composition root

**The one place that knows how everything is wired together.**

In Go, that's `main()`. It's where you read config, open the database, build
handlers, and start the server — and it should be the *only* place that does.
Nothing deeper in your code should call `pgxpool.New` or `os.Getenv`.

You already got this right by accident. Keep it.

```go
func main() {
    cfg := loadConfig()                      // config lives here
    pool := mustConnect(cfg.DatabaseURL)     // connections are made here
    defer pool.Close()

    store := NewPostgresItemStore(pool)      // wiring happens here
    svc := NewItemService(store)
    h := NewItemHandler(svc)

    http.ListenAndServe(":8080", h.Routes())
}
```

Read top to bottom, that's the entire architecture of the service. That's the
goal — and it's why "how long is `main`?" is a legitimate architecture
question.

---

## Part 4 — Which of these are worth arguing about

Straight answers, so you can spend your attention correctly:

| Worth learning properly | Worth recognising | Safe to ignore for now |
| --- | --- | --- |
| SoC | KISS, YAGNI | Liskov Substitution |
| Dependency inversion (D) | Rule of three, AHA | Open/Closed |
| Interface segregation (I) | DAMP (until testing) | Presenter |
| DI as "pass the argument" | Entity vs DTO | DI containers |
| Composition root | "Repository means two things" | Generic repositories |

And one meta-rule that will save you years:

> **Every one of these is a tool for reducing the cost of change.** If applying
> one doesn't make some future change cheaper, you're doing ceremony. Ceremony
> has a cost and no benefit.

When someone tells you your code violates a principle, the question isn't "is
that true?" It's **"what change does that make harder?"** If they can't answer,
the principle isn't the real issue.

---

## Exercise

No code. Open your Lesson 2 `main.go` and annotate it:

1. For each handler, write a comment listing every concern it handles. Use the
   four from Part 1's SoC section as a starting point.
2. Find one piece of **genuine** duplication (same knowledge) and one piece of
   **coincidental** duplication (same characters, different knowledge).
3. Write down every place that touches the global `db`. That count is how many
   files you'd have to edit to support a read replica.
4. Answer in one sentence: *what would you have to start up to test whether
   `price < 0` is rejected?*

Question 4 is the one that makes Lesson 4 feel necessary rather than academic.

---

## Quick reference

| Term | One line |
| --- | --- |
| **SoC** | One piece of code, one job. |
| **DRY** | One home per piece of *knowledge* — not per piece of text. |
| **Rule of three** | Wait for the third occurrence before extracting. |
| **AHA** | Avoid Hasty Abstractions. Duplication beats a wrong abstraction. |
| **KISS** | Prefer boring. Simple ≠ short. |
| **YAGNI** | Don't build for requirements you don't have. |
| **DAMP** | Tests should read plainly, even if that means repeating yourself. |
| **SOLID** | Five OO principles. In Go, S/I/D matter; L/O mostly don't. |
| **Dependency inversion** | Depend on an interface, not a concrete type. |
| **Interface segregation** | Small interfaces. One or two methods. |
| **Handler / controller** | Translates HTTP to Go and back. |
| **Usecase / service** | One business operation. No HTTP, no SQL. |
| **Repository / store** | Reads and writes data. Hides the SQL. |
| **Entity** | The domain object itself. |
| **DTO** | A shape that only exists for transport. |
| **DI** | Pass the dependency in as an argument. |
| **Composition root** | The one place that wires everything — your `main()`. |

---

## What you learned

- Why the vocabulary feels like a jumble: **four different categories** get
  listed as if they were alternatives.
- **Heuristics** are advice with known failure modes — including DRY, the most
  over-applied idea in the industry.
- **SOLID** is five principles of unequal value in Go; **D** is the one that
  everything else is built on.
- **Roles** are job titles for code, and your current code has one function
  doing four of them.
- **DI** is passing an argument. The word is scarier than the thing.
- **Composition root** is `main()`, and you already built one.

---

## Next

**[Lesson 3b — Structures](./readme-lesson3-intermediate-b-structures.md)** —
layered, Clean, Onion, Hexagonal, MVC/MVP/MVVM, and DDD. You now have the nouns;
3b arranges them. The main revelation is that four of those names describe the
same diagram.

Then **Lesson 4** stops reading and starts typing: your `main.go` gets taken
apart using exactly the terms above, one pain at a time.
