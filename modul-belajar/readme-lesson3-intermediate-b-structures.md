# Lesson 3b — Structures

> **Prerequisites:**
> [Lesson 3a — Terminology](./readme-lesson3-intermediate-a-terminology.md).
> This lesson uses the role nouns from 3a constantly: handler, usecase,
> repository, entity, DTO.

3a gave you the nouns. This one arranges them.

The obvious question — *"how many architectures are there?"* — has an
intimidating answer if you count names and a reassuring one if you count ideas.
There are about twenty names in common use. There are about six ideas. Most of
the names are synonyms that different authors coined for the same diagram, and
one whole family doesn't apply to backend Go at all.

**This lesson is reading, not typing.** Lesson 4 does the typing.

---

## Part 0 — Four families

The word "architecture" gets used for four completely unrelated decisions.
Almost all the confusion comes from comparing a member of one family against a
member of another.

| Family | Question it answers | Members |
| --- | --- | --- |
| **Presentation** | How does a UI talk to its state? | MVC, MVP, MVVM, MVU, VIPER |
| **Internal layering** | How is one service arranged inside? | Layered / N-tier, Clean, Onion, Hexagonal, Vertical Slice |
| **System shape** | How many deployable pieces? | Monolith, modular monolith, microservices, SOA, serverless |
| **Data flow** | How do the pieces talk? | REST, RPC/gRPC, GraphQL, event-driven, CQRS, event sourcing |

"Should we use MVC or microservices?" is a question like "should we use
past tense or paragraphs?" You can have both. They're not alternatives.

**What this lesson does with each family:**

| Family | Treatment |
| --- | --- |
| Presentation | Part 1 — explained, then set aside. It's not for JSON APIs. |
| **Internal layering** | **Parts 2–4 — this is the family that matters to you now.** |
| System shape | Part 6 — named so you recognise them. Its own lesson, much later. |
| Data flow | Part 7 — named so you recognise them. |

---

## Part 1 — The MVC family (and why it isn't yours)

All three are about the same problem: **you have state, and you have a screen
showing it. Who updates whom?**

| Pattern | The third letter | Where you'll meet it |
| --- | --- | --- |
| **MVC** — Model, View, **Controller** | Controller handles input, updates Model; View reads Model. | Original: Smalltalk desktop UIs, 1979. |
| **MVP** — Model, View, **Presenter** | View is dumb and passive; Presenter pushes data into it. | Android (older), WinForms. |
| **MVVM** — Model, View, **ViewModel** | ViewModel exposes bindable state; View auto-syncs to it. | WPF, SwiftUI, Vue, Angular. |

<details>
<summary>Two more you'll see</summary>

- **MVU / The Elm Architecture** — state is immutable; every event produces a
  new state and the view is a pure function of it. Elm, Redux, Jetpack Compose.
- **VIPER** — View, Interactor, Presenter, Entity, Router. An iOS-community
  expansion of MVP. Notice **Interactor** and **Entity** — those are Clean
  Architecture's words, which is exactly where VIPER got them.
</details>

### Why this doesn't apply to you

**Your service has no View.** It emits `{"id":1,"name":"Keyboard"}`. Nothing
observes state, nothing re-renders, nothing binds. The problem all three
patterns solve — keeping a screen in sync with data — does not exist in your
program.

So when a Go backend calls itself "MVC," what's actually happening is:

| The word they use | What it is in a Go API |
| --- | --- |
| Controller | Your handler |
| Model | Your entity, and usually the SQL too |
| View | The `json.Encoder`. That's the whole View. |

It's **layering with borrowed names**. Harmless, but it explains why searching
"golang MVC" gives you contradictory folder layouts: everyone is mapping a
UI pattern onto a non-UI program, and each of them maps it differently.

> **Takeaway:** know MVC/MVP/MVVM well enough to recognise them, and don't
> structure a JSON API around them. The Presenter role from 3a comes from
> here — and that's also why 3a said you probably don't need one.

Everything from here on is the layering family.

---

## Part 2 — Layered (N-tier)

**The default. Probably the most common backend structure in existence.**

Three stacked layers. Each one may call the layer below it, and never the layer
above.

```
┌──────────────────────────────┐
│  Presentation  (HTTP)        │   handlers
├──────────────────────────────┤
│  Business      (logic)       │   usecases
├──────────────────────────────┤
│  Data          (persistence) │   repositories
└──────────────────────────────┘
              │
              ▼
          Postgres
```

Your items service, layered:

| Layer | File | Contains |
| --- | --- | --- |
| Presentation | `handler.go` | Decode JSON, validate shape, write status codes |
| Business | `service.go` | "Price must be ≥ 0." "Name is required." |
| Data | `store.go` | The `INSERT ... RETURNING id` |

The rewrite of `handleCreateItem`, split three ways:

```go
// handler.go — knows HTTP, knows nothing about SQL
func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
    var item Item
    if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON")
        return
    }
    created, err := h.svc.Create(r.Context(), item)   // hand it down
    if errors.Is(err, ErrInvalidItem) {
        writeError(w, http.StatusBadRequest, err.Error())
        return
    }
    if err != nil {
        writeError(w, http.StatusInternalServerError, "database error")
        return
    }
    writeJSON(w, http.StatusCreated, created)
}

// service.go — knows the rules, knows nothing about HTTP or SQL
func (s *ItemService) Create(ctx context.Context, item Item) (Item, error) {
    if item.Name == "" || item.Price < 0 {
        return Item{}, ErrInvalidItem
    }
    return s.store.Create(ctx, item)
}

// store.go — knows SQL, knows nothing about HTTP or the rules
func (s *PostgresItemStore) Create(ctx context.Context, item Item) (Item, error) {
    err := s.db.QueryRow(ctx,
        `INSERT INTO items (name, price) VALUES ($1, $2) RETURNING id`,
        item.Name, item.Price).Scan(&item.ID)
    return item, err
}
```

Read those three functions again and notice: **no function mentions more than
one concern.** `service.go` has no `http` import. `store.go` has no `http`
import. That's SoC, done.

**Strengths:** obvious, universally understood, and the split matches how you'd
describe the code out loud.

**The one real weakness** — and it's the reason Clean Architecture exists:

```
service.go  ──────────►  store.go  ──────────►  pgx
```

The business layer *depends on* the data layer. So to test "price must be ≥ 0"
you still need a `PostgresItemStore`, which needs a `*pgxpool.Pool`, which
needs Postgres running. The layers are separated but the **arrow still points
down into the database.**

Hold that thought. It's the entire content of Part 3.

---

## Part 3 — Clean = Onion = Hexagonal = Ports & Adapters

Four famous names. **One idea.** Here they are with their origins:

| Name | Author | Year | Signature image |
| --- | --- | --- | --- |
| **Hexagonal / Ports & Adapters** | Alistair Cockburn | 2005 | A hexagon with plugs around the edge |
| **Onion Architecture** | Jeffrey Palermo | 2008 | Concentric rings |
| **Clean Architecture** | Robert C. Martin | 2012 | Concentric rings, four of them, colour-coded |
| **DCI / Screaming Architecture** | various | — | Variations on the same |

Martin himself says in the Clean Architecture book that these are the same
thing under different names. He wasn't being modest — read the diagrams side by
side and they're the same diagram.

### The one rule

> **The Dependency Rule: source code dependencies point inward only.**
> Inner layers must not know anything about outer layers.

That's it. Everything else is illustration.

```
        ┌─────────────────────────────────────┐
        │  Adapters:  HTTP · Postgres · Redis │   ← replaceable
        │   ┌─────────────────────────────┐   │
        │   │  Usecases (application)     │   │   ← your operations
        │   │    ┌───────────────────┐    │   │
        │   │    │  Domain (entities)│    │   │   ← the rules, pure Go
        │   │    └───────────────────┘    │   │
        │   └─────────────────────────────┘   │
        └─────────────────────────────────────┘
                  arrows point ⟶ inward
```

The centre imports nothing. Not `net/http`, not `pgx`, not `encoding/json`.
Just Go and your own domain types.

### Their vocabularies, translated

The same three or four boxes, renamed by each author. This table is the single
most useful thing in this lesson:

| Layered | Hexagonal | Onion | Clean | In your Go code |
| --- | --- | --- | --- | --- |
| Presentation | Driving adapter | Infrastructure | Interface Adapters | `handler.go` |
| Business | Application | Application Services | Use Cases | `service.go` |
| — | Port | Interface | Boundary | `type ItemStore interface` |
| Data | Driven adapter | Infrastructure | Interface Adapters / Frameworks | `store.go` |
| Entity | Domain | Domain Model | Entities | `type Item struct` |

**"Port" is just an interface. "Adapter" is just the thing that implements
it.** Cockburn picked hardware words — a port is the socket, an adapter is the
plug. Once you know that, Hexagonal Architecture is a two-page idea.

### How it differs from layered, in code

Layered had `service.go` importing `store.go`. Clean adds one line to fix it:

```go
// service.go — the interface is declared HERE, by the consumer
type ItemStore interface {
    Create(ctx context.Context, item Item) (Item, error)
    Get(ctx context.Context, id int) (Item, error)
}

type ItemService struct {
    store ItemStore        // depends on the interface, not on Postgres
}
```

And `store.go` never mentions `ItemStore` at all — Go interfaces are implicit,
so `*PostgresItemStore` satisfies it just by having the methods.

The arrow flipped:

```
Layered:   service ─────────────► store ─────► pgx

Clean:     service ──► ItemStore ◄──────────── store ─────► pgx
```

That's **dependency inversion** from 3a, applied at the level of a whole
program instead of a single type. And the payoff is concrete:

```go
type fakeStore struct{ items map[int]Item }

func (f *fakeStore) Create(_ context.Context, i Item) (Item, error) {
    i.ID = len(f.items) + 1
    f.items[i.ID] = i
    return i, nil
}
func (f *fakeStore) Get(_ context.Context, id int) (Item, error) { /* ... */ }

// Test the price rule with no Docker, no Postgres, in 0.001 seconds:
svc := NewItemService(&fakeStore{items: map[int]Item{}})
_, err := svc.Create(ctx, Item{Name: "Keyboard", Price: -5})
// err == ErrInvalidItem
```

**That is the whole reason Clean Architecture exists.** Not elegance. Not
diagrams. The ability to run your business rules without starting a database.

### The honest downsides

Clean Architecture done by the book, in Go, is frequently a mistake:

| Cost | What it looks like |
| --- | --- |
| **File explosion** | Four folders and six files to add one field |
| **Mapping tax** | `Item` → `ItemDTO` → `ItemModel` → `ItemResponse`, with converters between each |
| **Interfaces with one implementation** | Ceremony, per KISS in 3a |
| **Indirection on every read** | Jump-to-definition lands on an interface, not the code |

Go's culture pushes back on all of this. The community position is roughly:
**take the Dependency Rule, leave the folder structure.** Lesson 4 does exactly
that — one interface, at the one boundary where it buys something.

---

## Part 4 — Vertical Slice

The real alternative to layering, and the one most people never hear about.

Layered architecture cuts the program **horizontally**, by technical concern —
all handlers together, all repositories together. Vertical slice cuts it
**by feature**:

```
Layered (by technical role)      Vertical slice (by feature)

internal/                        internal/
  handler/                         item/
    item.go                          handler.go
    order.go                         service.go
    user.go                          store.go
  service/                         order/
    item.go                          handler.go
    order.go                         service.go
    user.go                          store.go
  store/                           user/
    item.go                          handler.go
    order.go                         service.go
    user.go                          store.go
```

**Same nine files. Completely different day-to-day experience.**

The argument for vertical slice is a practical one: *"add a field to Item"*
touches three files in one folder, not three files in three folders. Code that
changes together lives together. And a whole feature can be deleted by deleting
a directory.

This suits Go unusually well, because Go's package system is designed around
exactly this — a package is a unit of meaning, and `internal/item` can keep its
own types unexported.

| Cut by | Good when | Painful when |
| --- | --- | --- |
| **Layer** (horizontal) | Everything shares the same rules and shapes | You navigate three folders per change |
| **Feature** (vertical) | Features are independent; teams own features | Two features genuinely need the same logic |

Note that these compose: **vertical slices, each internally layered** is a very
common and very good arrangement, and it's what "modular monolith" (Part 6)
usually means in practice.

---

## Part 5 — Your current code already has a name

Martin Fowler catalogued the ways business logic gets organised. Yours is in
there:

| Pattern | What it is | Verdict |
| --- | --- | --- |
| **Transaction Script** | One procedure per operation, doing everything start to finish | **This is your Lesson 2 code.** |
| **Table Module** | One class per table, holding all operations for it | Common in .NET |
| **Active Record** | The entity knows how to save itself: `item.Save()` | Rails, GORM, Django |
| **Domain Model** | Rich objects with behaviour; persistence handled separately | What DDD aims at |

Two things worth knowing here.

**Transaction Script is a legitimate pattern, not a beginner mistake.** For
CRUD with thin rules, it's frequently the correct choice, and dressing it up in
four layers is the actual error. You didn't write bad code in Lesson 2 — you
wrote a Transaction Script, and it stops being the right answer at a specific
point that Lesson 4 will show you.

**Active Record is why this course skipped ORMs.** `item.Save()` is
comfortable and it welds your entity to your database schema permanently. That
weld is exactly what Clean Architecture's inner circle forbids — which is why
"ORM" and "Clean Architecture" arguments never end.

---

## Part 6 — System shapes

The family everyone jumps to first. It's last here because it's the one you can
act on last — you need one service to be good before the number of services is
an interesting question.

| Shape | What it means | Honest note |
| --- | --- | --- |
| **Monolith** | One deployable. All code, one process. | The correct default. Most successful systems are this. |
| **Modular monolith** | One deployable, strong internal module boundaries. | Usually the right target. Vertical slices, enforced. |
| **Microservices** | Many deployables, each owning its own data. | Solves an *organisational* problem — many teams shipping independently. |
| **SOA** | Microservices' 2000s ancestor, usually with a central bus. | Mostly historical. |
| **Distributed monolith** | Microservices that must deploy together. | **The worst outcome.** All the cost, none of the benefit. |
| **Serverless / FaaS** | Functions, no server to manage. | A deployment model more than an architecture. |

### The one thing to internalise now

**Microservices are not "the advanced version of a monolith."** They trade an
easy problem for a very hard one:

| Monolith | Microservices |
| --- | --- |
| A function call | A network call that can fail, retry, or arrive twice |
| One transaction, `COMMIT` | Distributed transactions, sagas, eventual consistency |
| One log file | Distributed tracing, correlation IDs |
| Refactor across modules freely | Coordinated deploys across teams |
| Run it locally | Run 12 of them locally, somehow |

The industry's own consensus, after a decade of trying it:
[**start with a monolith**](https://martinfowler.com/bliki/MonolithFirst.html).
Split only when a specific piece has a genuinely different scaling profile, or
a separate team needs to deploy on their own schedule.

**This gets its own lesson, and it comes last in the course.** Not because it's
the hardest concept — because it's the one where premature adoption does the
most damage.

---

## Part 7 — Data flow styles

Also called "architecture," also a different question. Named here so you
recognise them; each is a topic of its own.

| Style | One line |
| --- | --- |
| **REST** | Resources and HTTP verbs. What you built in Lesson 1. |
| **RPC / gRPC** | Call a remote function. Typed contracts, binary, fast. Common between internal services. |
| **GraphQL** | Client asks for exactly the fields it wants, one endpoint. |
| **Event-driven / pub-sub** | Emit "OrderPlaced"; whoever cares subscribes. Kafka, NATS, RabbitMQ. |
| **CQRS** | Separate the write model from the read model. |
| **Event sourcing** | Store the events, not the current state. Rebuild state by replaying. |

Two warnings, since these are the most over-adopted ideas on the list:

- **CQRS and event sourcing are frequently confused.** They're independent —
  you can do either without the other.
- **Event sourcing makes "what is the price right now?" a hard question.** It's
  a superb fit for auditing and finance, and a needless tax on a product
  catalogue.

---

## Part 8 — Where DDD fits

**DDD is not on any of these lists, because it isn't a structure.** It's
Domain-Driven Design — a *method* for deciding what your code should be about,
proposed by Eric Evans in 2003. It produces structures; it isn't one.

Its ideas, split by how much they'll pay you back today:

### Worth taking now — free, no structure required

| Idea | What it means |
| --- | --- |
| **Ubiquitous language** | Code uses the words the business uses. If they say "SKU," don't call it `ProductCode`. Same word, everywhere, no translation layer in anyone's head. |
| **Repository** | Data access hidden behind a domain-shaped interface. You met it in 3a. |
| **Anemic domain model** (an anti-pattern) | Structs with only fields, all behaviour in "service" files. DDD says the entity should own its rules. Go's community is genuinely split on this — anemic is common and often fine. |

### Worth knowing, applies later

| Idea | What it means | When you need it |
| --- | --- | --- |
| **Value object** | Identity is its value, not an ID. `Money{cents, currency}`. Immutable. | Once "price" stops being a bare `int` |
| **Aggregate** | A cluster of objects saved and validated as one unit, entered through one root. `Order` owns its `OrderLine`s. | Once one write touches several tables |
| **Bounded context** | The same word means different things in different parts of the business. "Customer" in Billing ≠ "Customer" in Support. Keep separate models. | Once the system spans real business areas |
| **Domain event** | "OrderPlaced" as a first-class thing the domain emits | Once side effects multiply |

**Bounded contexts are the concept that connects DDD to Part 6.** They're the
only principled way anyone has found to decide *where to cut* a system into
services. Cut along bounded contexts and you get microservices; cut anywhere
else and you get a distributed monolith. That's why DDD and microservices are
always mentioned together.

**For a single items service, DDD is overkill** — except for ubiquitous
language, which costs nothing and is worth adopting today.

---

## Part 9 — What Lesson 4 actually does

Everything above, filtered down to what earns its place in a small Go service:

| Adopting | Why |
| --- | --- |
| **Separation of concerns** | Handler / service / store. Three jobs, three places. |
| **The Dependency Rule** | The service layer will not import `pgx`. |
| **One interface, at the store boundary** | Because it makes rules testable without Docker. |
| **Dependency injection by hand** | Constructor arguments. `main` wires it. |
| **Ubiquitous language** | Call things what the domain calls them. |

| Skipping | Why |
| --- | --- |
| Presenter layer | Your output is `json.Encoder`. |
| Separate DTO and entity types | They're identical today. YAGNI. |
| Interfaces on the handler and service | One implementation each, no test double needed. |
| Aggregates, value objects, domain events | One table. Nothing to aggregate. |
| CQRS, event sourcing | No read/write asymmetry to solve. |
| Microservices | You have one service and no team. |

> **The rule underneath all of that:** adopt a structure when you can name the
> specific pain it removes. "Because it's the best practice" is not a pain.

---

## Exercise

Still no code.

1. Take the vocabulary table from Part 3 and write, next to each row, the name
   *you* would give that file in your own project. You'll have to pick one
   dialect eventually — pick it now, deliberately.
2. Sketch your items service as vertical slices instead of layers. If you added
   `orders` tomorrow, which layout means fewer folders open at once?
3. Look at Part 6's monolith-vs-microservices table. Write down which row you'd
   find hardest to deal with if your one service became five.
4. Find one word in your Lesson 2 code that the "business" wouldn't use.
   (Hint: is a row in `items` called an item, a product, a SKU, or a listing?
   Pick one and mean it.)

Question 4 is ubiquitous language. It's the cheapest idea in this lesson and
the one most consistently skipped.

---

## Quick reference

| Term | One line |
| --- | --- |
| **MVC / MVP / MVVM** | UI patterns. No View in a JSON API — don't structure around them. |
| **Layered / N-tier** | Presentation → business → data. The sane default. |
| **Clean / Onion / Hexagonal / Ports & Adapters** | One idea, four names: dependencies point inward. |
| **Dependency Rule** | Inner layers know nothing about outer layers. |
| **Port** | An interface. |
| **Adapter** | The thing that implements the port. |
| **Vertical slice** | Organise by feature, not by technical role. |
| **Transaction Script** | One procedure per operation. Your Lesson 2 code. |
| **Active Record** | The entity saves itself. `item.Save()`. What ORMs give you. |
| **Monolith** | One deployable. Correct default. |
| **Modular monolith** | One deployable, real internal boundaries. Usually the target. |
| **Microservices** | Many deployables. Solves a team problem, costs a distributed-systems problem. |
| **Distributed monolith** | Services that must deploy together. Worst of both. |
| **CQRS** | Separate read model from write model. |
| **Event sourcing** | Store events, derive state. |
| **DDD** | A method for modelling the domain, not a structure. |
| **Ubiquitous language** | Code uses the business's words. Free. Do it. |
| **Aggregate** | Objects saved and validated as one unit, via one root. |
| **Bounded context** | Same word, different meaning in different areas. Where to cut services. |
| **Anemic domain model** | Data-only structs, logic elsewhere. An anti-pattern, arguably. |

---

## What you learned

- **Four families**, not one list. Comparing across families is why the
  vocabulary feels incoherent.
- **MVC/MVP/MVVM are presentation patterns.** Your API has no View, so "Go MVC"
  is layering wearing borrowed names.
- **Layered** is the default, and its one weakness is that business depends on
  data.
- **Clean, Onion, Hexagonal, and Ports & Adapters are the same architecture**,
  and its entire content is one rule: dependencies point inward. A port is an
  interface; an adapter implements it.
- The payoff isn't elegance — it's **testing your rules without Postgres
  running.**
- **Vertical slice** cuts by feature instead of by layer, and fits Go's package
  system well.
- Your Lesson 2 code is a **Transaction Script**, which is a real pattern and
  often correct.
- **Microservices trade an easy problem for a hard one.** Monolith first.
- **DDD is a method, not a structure.** Take ubiquitous language today; leave
  aggregates and bounded contexts for a system big enough to need them.

---

## Next

**Lesson 4 — Decoupling & Dependency Injection.** The reading stops. You take
your Lesson 2 `main.go` apart into handler, service, and store, introduce
exactly one interface, delete the global `db`, and write the first test that
runs without Docker.

Every move in it has a name from 3a or 3b — and now you know them.
