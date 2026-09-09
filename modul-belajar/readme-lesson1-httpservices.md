# Lesson 1 — HTTP Services

Everything in this lesson lives in a single `main.go`. No folders, no layers, no
framework. Just enough to make a computer answer a question over the network.

**By the end you will have built:** a small JSON API you can call from your
browser, from `curl`, and from any other program — first with Go's standard
library, then the exact same thing again using Gin.

**Prerequisites:** [section 1 of the main README](./README.md) — Go installed
and a working `hello-go` project.

---

## Part 0 — The idea

### What is a "web service"?

A program that stays running, waits for messages over the network, and replies
to them.

That's it. When you open `google.com`, your browser sends a small block of text
to a machine, and that machine sends a block of text back. A web service is the
program on the other end.

The text follows a format called **HTTP**. A request looks like this — literally
this, as plain text:

```
GET /items/1 HTTP/1.1
Host: localhost:8080
```

And the reply:

```
HTTP/1.1 200 OK
Content-Type: application/json

{"id":"1","name":"Keyboard","price":45}
```

Everything in this lesson is about producing that second block.

### The handler

In Go, you answer a request by writing a **handler** — a function with this
exact shape:

```go
func hello(w http.ResponseWriter, r *http.Request) {
}
```

Notice what's missing: **there is no return value.**

That surprises nearly everyone. You don't *return* a response, you *write* one.

| Parameter | What it is | What you do with it |
| --- | --- | --- |
| `r *http.Request` | What they sent you — method, path, headers, body | **Read** from it |
| `w http.ResponseWriter` | A blank page for your reply | **Write** to it |

> **The mental model:** you're handed a letter (`r`) and a blank sheet of paper
> (`w`). You read the letter, write your reply on the sheet, and hand it back.
> Nothing is returned, because the caller is already holding the paper.

That analogy also explains the rule that catches everyone out. Writing on the
sheet happens **in a fixed order**, and you can't un-write it:

```go
w.Header().Set("Content-Type", "application/json")  // 1. headers
w.WriteHeader(http.StatusCreated)                   // 2. status code — once only
json.NewEncoder(w).Encode(item)                     // 3. body
```

Set a header after the status, or call `WriteHeader` twice, and Go prints
`superfluous response.WriteHeader call` to your console while silently sending
the wrong thing. Ink on paper doesn't come off.

---

# Part 1 — Pure `net/http`

Create a new folder `http-service`, open it in your editor, and run
`go mod init example/httpservice` (same steps as §1.4 of the main README).

---

## Step 1 — The smallest possible server

Put this in `main.go`:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, backend!")
	})

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

Run it with `go run .`. **Your terminal will appear to hang.** That's correct —
the server is waiting for requests. This is the first genuinely new idea: unlike
every program you've written before, this one doesn't finish.

Open <http://localhost:8080> in your browser. You should see `Hello, backend!`.

Stop the server with **`Ctrl+C`**.

### What each line does

| Line | Meaning |
| --- | --- |
| `http.NewServeMux()` | Creates a **router** — the thing that decides which handler answers which URL. |
| `mux.HandleFunc("/", ...)` | "For this path, run this function." |
| `fmt.Fprintln(w, ...)` | Write text onto the response. `Fprintln` writes to a destination — here, `w`. |
| `http.ListenAndServe(":8080", mux)` | Start waiting for requests on port 8080, and hand each one to `mux`. |
| `log.Fatal(...)` | If the server fails to start, print why and exit. |

That last one matters more than it looks. **Without `log.Fatal`, a failure to
start is completely silent** — you'd sit there wondering why nothing works. Try
it: start the server in one terminal, then run `go run .` again in a second
terminal. You'll see:

```
listen tcp :8080: bind: address already in use
```

Only one program can hold a port at a time. This will happen to you for real
when you forget to `Ctrl+C` an old server.

---

## Step 2 — Look at the actual HTTP

Don't skip this step. Start the server again, and in a **second terminal** run:

```bash
curl -v http://localhost:8080
```

```
> GET / HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/8.5.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Content-Type: text/plain; charset=utf-8
< Date: Mon, 18 Aug 2026 09:12:44 GMT
<
Hello, backend!
```

Lines starting with `>` were **sent by curl**. Lines with `<` were **sent by
your Go program**. That's the whole conversation.

Your browser sent almost exactly the same thing a moment ago. There is nothing
else going on — a web service is text in, text out, over a socket.

Note that Go filled in `HTTP/1.1 200 OK`, `Content-Type`, and `Date` for you.
You never wrote those. Go picks sensible defaults, and the rest of this lesson
is about overriding them when you need to.

---

## Step 3 — Read the request

The request isn't magic either. It's a struct with fields you can read.

```go
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)
	fmt.Fprintln(w, "Hello, backend!")
})
```

Restart, then try a few:

```bash
curl http://localhost:8080/
curl http://localhost:8080/anything/you/like
curl -X POST http://localhost:8080/submit
```

Your server's terminal prints:

```
GET /
GET /anything/you/like
POST /submit
```

Two things to take from this:

- `r` is just data. Read `r.Method`, `r.URL.Path`, `r.Header` with a dot, like
  any other variable. (`r` is a **struct** — there's a full explanation of what
  that means in the interlude before Step 7.)
- Right now **every** path hits the same handler, because `"/"` acts as a
  catch-all. That's the next problem to solve.

---

## Step 4 — Real routes

Two changes: give handlers proper names instead of burying them inline, and
match on method *and* path.

```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /hello", handleHello)
	mux.HandleFunc("GET /health", handleHealth)

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, backend!")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}
```

```bash
curl http://localhost:8080/hello     # Hello, backend!
curl http://localhost:8080/health    # ok
curl http://localhost:8080/nope      # 404 page not found
curl -X POST http://localhost:8080/hello   # Method Not Allowed
```

You wrote no code for those last two. Because the pattern says `GET /hello`,
Go returns **405 Method Not Allowed** for a POST, and **404 Not Found** for an
unknown path, automatically.

> **A note on `GET /hello` patterns.** Method matching and `{placeholders}` were
> added to the standard library in **Go 1.22** (2024). Most tutorials online are
> older and will tell you that you need a framework for this. You don't, and
> haven't for years. If a code example uses `gorilla/mux` or `chi` purely for
> routing, it's out of date.

`/health` isn't decoration, by the way — a trivial endpoint that says "I'm
alive" is what load balancers and monitoring systems poll in production. You'll
put one in every service you ever write.

---

## Step 5 — Query parameters

The bit after `?` in a URL: `/greet?name=Sam`.

```go
mux.HandleFunc("GET /greet", handleGreet)

func handleGreet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "stranger"
	}
	fmt.Fprintf(w, "Hello, %s!\n", name)
}
```

```bash
curl "http://localhost:8080/greet?name=Sam"   # Hello, Sam!
curl "http://localhost:8080/greet"            # Hello, stranger!
```

`Query().Get()` returns an **empty string** when the parameter is absent — it
does not error. Every value arriving from the network is optional until you
check it. Handling the empty case is your job, and forgetting to is one of the
most common sources of bugs in real services.

> Quote the URL in your shell. Unquoted, `&` in a URL will background your
> command and confuse you badly.

---

## Step 6 — Path parameters

For identifying *a specific thing*, the id belongs in the path, not the query:
`/items/1`.

```go
mux.HandleFunc("GET /items/{id}", handleGetItem)

func handleGetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fmt.Fprintf(w, "you asked for item %s\n", id)
}
```

```bash
curl http://localhost:8080/items/1     # you asked for item 1
curl http://localhost:8080/items/abc   # you asked for item abc
```

The `{id}` in the pattern and the `"id"` in `PathValue("id")` must match — that
string is how you retrieve it.

**Query vs path, the rule of thumb:** the path identifies *which resource*
(`/items/1`), the query modifies *how you want it* (`/items?sort=price&limit=10`).

---

# Interlude — Structs

From Step 7 onwards you stop sending loose strings and start sending **things**:
an item, a user, an order. Go's tool for describing a "thing" is the **struct**.

This is a pause from HTTP. Read it before continuing — everything after Step 6
depends on it.

## The problem structs solve

Say you want to handle an item with an id, a name, and a price. Without structs,
that's three separate variables:

```go
id := "1"
name := "Keyboard"
price := 45
```

This falls apart quickly. How do you return all three from a function? How do
you store a hundred items? Nothing connects these three variables — the language
doesn't know they belong together, so keeping them in sync is entirely your
problem.

A struct fixes that by **grouping related values into one named type**:

```go
type Item struct {
	ID    string
	Name  string
	Price int
}
```

Read it as: *"there is now a type called `Item`. Every `Item` has an `ID` and a
`Name`, both text, and a `Price`, a whole number."*

You've invented a new type. Go now checks it for you everywhere.

## Creating one

```go
keyboard := Item{
	ID:    "1",
	Name:  "Keyboard",
	Price: 45,
}
```

You can write it on one line too — `Item{ID: "1", Name: "Keyboard", Price: 45}`.

> **Always name the fields.** Go also allows `Item{"1", "Keyboard", 45}`,
> relying on declaration order. Don't. The day someone adds a field in the
> middle, every one of those breaks — and some will break silently, by putting
> the right value in the wrong field.

## Reading and changing fields

A dot, and the field name:

```go
fmt.Println(keyboard.Name)    // Keyboard
fmt.Println(keyboard.Price)   // 45

keyboard.Price = 50           // change it
```

That's the same dot you've been using since Step 3 on `r.Method` and
`r.URL.Path`. **You've been reading struct fields this whole time** —
`http.Request` is a struct someone else wrote, with about thirty fields on it.

## Zero values: there is no "undefined"

Declare a struct without filling it in:

```go
var empty Item
fmt.Printf("%+v\n", empty)   // {ID: Name: Price:0}
```

It isn't null, undefined, or an error. Every field gets its type's **zero
value**:

| Type | Zero value |
| --- | --- |
| `string` | `""` (empty text) |
| `int`, `float64` | `0` |
| `bool` | `false` |
| pointers, slices, maps | `nil` |

This is one of Go's better ideas — a declared variable is always usable, so
there's no "cannot read property of undefined" class of crash.

**It's also about to become very relevant.** When JSON arrives missing a field,
Go doesn't complain — it leaves that field at its zero value. A request body of
`{"price": 10}` produces an `Item` whose `Name` is `""`. That's exactly why
Step 8 checks `if item.Name == ""`. Zero values are how you detect what the
caller left out.

## Structs are copied

Assigning a struct copies it. The two variables are then unrelated:

```go
a := Item{Name: "Keyboard"}
b := a              // b is a full, independent copy
b.Name = "Mouse"

fmt.Println(a.Name) // Keyboard — unchanged
fmt.Println(b.Name) // Mouse
```

The same happens when you pass a struct to a function: the function gets a copy,
and changes it makes are thrown away when it returns.

## Pointers: when you want the original

A pointer is the *address* of a value rather than the value itself. `&a` means
"the address of `a`", and `*Item` means "the address of an Item".

```go
p := &a             // p points at a — no copy
p.Name = "Mouse"

fmt.Println(a.Name) // Mouse — the original changed
```

Note you write `p.Name`, not `(*p).Name`. Go follows the pointer for you.

This explains two things you've already seen and one you're about to:

| Code | Why there's a pointer |
| --- | --- |
| `r *http.Request` | Requests are large. Copying one for every handler call would be wasteful — you get the address instead. |
| `json.NewDecoder(r.Body).Decode(&item)` *(Step 8)* | `Decode` has to fill in **your** variable. Hand it a copy and it fills in the copy, and your `item` stays empty. |

Rule of thumb for now: **pass a pointer when the function needs to modify your
value, or when the struct is large.** Otherwise a plain value is simpler and
safer.

## Capital letters are load-bearing

A field starting with a capital letter is **exported** — visible to other
packages. Lowercase means it's private to the package that declared it.

```go
type Item struct {
	Name     string   // exported — other packages can see this
	internal string   // unexported — invisible outside this file's package
}
```

Go has no `public` or `private` keyword. The case of the first letter *is* the
access modifier, and it applies to types, functions, and fields alike.

Remember this at Step 7. The `encoding/json` package is a different package, so
it can only see exported fields. Lowercase a field and it silently disappears
from your JSON output — no error, no warning.

## Why structs fit HTTP so well

Put a JSON object next to a Go struct:

```json
{ "id": "1", "name": "Keyboard", "price": 45 }
```

```go
type Item struct {
	ID    string
	Name  string
	Price int
}
```

Same shape: named fields holding typed values. That correspondence is why Go can
convert between the two automatically, and it's the reason every request and
response type you write from here on will be a struct.

One piece of struct syntax is still missing — **tags**, the `` `json:"name"` ``
bit that controls the exact spelling of those JSON keys. It's introduced in
Step 7, where you'll need it.

## One more thing: maps

Step 7 also stores items in a **map** — a lookup table of keys to values.

```go
var items = map[string]Item{
	"1": {ID: "1", Name: "Keyboard", Price: 45},
	"2": {ID: "2", Name: "Monitor", Price: 220},
}
```

`map[string]Item` means "keys are `string`, values are `Item`". Inside the
braces you can omit the repeated `Item{...}` — Go already knows what type
belongs there.

Working with one:

```go
items["3"] = Item{ID: "3", Name: "Mouse", Price: 25}   // add or replace
delete(items, "3")                                      // remove
count := len(items)                                     // how many
```

And the lookup you'll use constantly:

```go
item, ok := items["1"]
```

A map lookup returns **two** values: the item, and a `bool` saying whether the
key existed. If it didn't, `ok` is `false` and `item` is the zero value.

That second value is your 404 check in Step 7:

```go
item, ok := items[id]
if !ok {
	http.Error(w, "item not found", http.StatusNotFound)
	return
}
```

Ignoring `ok` is a real bug: `items["nope"]` happily returns an empty `Item`
rather than failing, and you'd serve `{"id":"","name":"","price":0}` with a
`200 OK`.

## Try it before moving on

Ten minutes here will save you an hour later. In a scratch folder:

```go
package main

import "fmt"

type Item struct {
	ID    string
	Name  string
	Price int
}

func main() {
	keyboard := Item{ID: "1", Name: "Keyboard", Price: 45}
	fmt.Printf("%+v\n", keyboard)

	var empty Item
	fmt.Printf("%+v\n", empty)

	copied := keyboard
	copied.Name = "Mouse"
	fmt.Println(keyboard.Name, copied.Name)

	pointed := &keyboard
	pointed.Name = "Trackball"
	fmt.Println(keyboard.Name)

	stock := map[string]Item{"1": keyboard}
	found, ok := stock["1"]
	fmt.Println(found.Name, ok)

	missing, ok := stock["99"]
	fmt.Printf("%+v %v\n", missing, ok)
}
```

Expected output:

```
{ID:1 Name:Keyboard Price:45}
{ID: Name: Price:0}
Keyboard Mouse
Trackball
Trackball true
{ID: Name: Price:0} false
```

If every line makes sense — especially why the third prints two different names
and the fourth prints one changed one — you're ready for Step 7.

> `%+v` is a formatting verb that prints a struct with its field names. `%v`
> alone gives just the values. Both are far more useful than `fmt.Println` for
> inspecting structs while debugging.

---

## Step 7 — Return JSON

Text is fine for humans. Other programs want JSON.

First, define what an item *is*, and keep a few in memory:

```go
type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var items = map[string]Item{
	"1": {ID: "1", Name: "Keyboard", Price: 45},
	"2": {ID: "2", Name: "Monitor", Price: 220},
}
```

Then rewrite the handler:

```go
func handleGetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	item, ok := items[id]
	if !ok {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}
```

Add `"encoding/json"` to your imports.

```bash
curl http://localhost:8080/items/1
# {"id":"1","name":"Keyboard","price":45}

curl -i http://localhost:8080/items/99
# HTTP/1.1 404 Not Found
# item not found
```

### Three things worth stopping on

**1. Struct tags.** The `` `json:"name"` `` after each field controls the key in
the JSON output. Without tags you'd get `{"ID":"1","Name":"Keyboard"}` — Go's
field names leak into your API. Tags let the Go code and the JSON follow their
own conventions.

**2. Capital letters are load-bearing.** `encoding/json` can only see
**exported** fields — ones starting with a capital letter. Rename `Name` to
`name` and it silently vanishes from the output, with no error. This one costs
every Go beginner an afternoon at least once.

**3. `return` after an error.** That `return` inside the `if !ok` block is not
optional:

```go
if !ok {
	http.Error(w, "item not found", http.StatusNotFound)
	// no return here
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(item)   // ← still runs! writes an empty item after the 404
```

There's no return value to stop the function, so **you** have to stop it.
Whenever you write an error response, `return` on the next line.

---

## Step 8 — Accept JSON

Reading a body is the mirror image of writing one: `Decode` instead of `Encode`,
`r.Body` instead of `w`.

```go
mux.HandleFunc("POST /items", handleCreateItem)

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

	item.ID = strconv.Itoa(len(items) + 1)
	items[item.ID] = item

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}
```

Add `"strconv"` to your imports.

```bash
curl -X POST http://localhost:8080/items \
  -H "Content-Type: application/json" \
  -d '{"name":"Mouse","price":25}'
# {"id":"3","name":"Mouse","price":25}

curl -X POST http://localhost:8080/items -d 'not json'
# invalid JSON

curl -X POST http://localhost:8080/items -d '{"price":10}'
# name is required
```

### The `&` in `Decode(&item)`

`&item` means "the address of `item`" — you're handing `Decode` a pointer so it
can fill in the variable you already declared. Without the `&`, `Decode` would
receive a copy, fill in the copy, and your `item` would stay empty. Go actually
catches this one for you with an error, thankfully.

### Validate everything

Those two checks are the entire reason this handler is more than three lines.
**Data from the network is not trustworthy.** It may be malformed, incomplete,
or hostile. The check for `item.Name == ""` is the beginning of a topic called
validation that gets a lesson of its own later.

Also note what we *ignore*: the client can send `{"id":"999"}` and we overwrite
it. The server decides ids, not the caller.

> **A known bug we're leaving in.** That `items` map is not safe for concurrent
> use. Two simultaneous POSTs can crash the program outright. Go's HTTP server
> runs every request in its own goroutine, so this is a real risk, not a
> theoretical one. Fixing it is a later lesson — but it's worth knowing it's
> there rather than discovering it in production.

---

## Step 9 — Status codes

The number in `HTTP/1.1 200 OK` is how your service tells the caller what
happened, in a way a *program* can act on. Anyone can read "item not found";
a program checks for `404`.

The handful you'll actually use:

| Code | Name | When |
| --- | --- | --- |
| `200` | OK | It worked |
| `201` | Created | It worked and something new exists |
| `400` | Bad Request | The caller sent nonsense |
| `401` / `403` | Unauthorized / Forbidden | Who are you / you may not do that |
| `404` | Not Found | No such thing |
| `405` | Method Not Allowed | Right path, wrong verb |
| `500` | Internal Server Error | *You* broke, not them |

The line that matters: **`4xx` is their fault, `5xx` is yours.** If a malformed
request produces a `500`, you have a bug — you failed to validate.

Two ways to set one:

```go
http.Error(w, "item not found", http.StatusNotFound)  // status + plain-text message
w.WriteHeader(http.StatusCreated)                      // status only
```

Use the named constants (`http.StatusNotFound`), not the numbers. They're
self-documenting and immune to typos — `418` compiles just as happily as `404`.

### Prove the ordering rule to yourself

Deliberately break it:

```go
w.WriteHeader(http.StatusCreated)
w.Header().Set("Content-Type", "application/json")   // too late — ignored
w.WriteHeader(http.StatusOK)                          // too late — ignored + warning
```

Run it and watch your server's console:

```
http: superfluous response.WriteHeader call from main.handleCreateItem
```

The response still goes out, with the *first* status and *no* content type. It
fails quietly, which is exactly why it's worth breaking on purpose now.

---

## The complete program

Everything above, in one file:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var items = map[string]Item{
	"1": {ID: "1", Name: "Keyboard", Price: 45},
	"2": {ID: "2", Name: "Monitor", Price: 220},
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /greet", handleGreet)
	mux.HandleFunc("GET /items", handleListItems)
	mux.HandleFunc("GET /items/{id}", handleGetItem)
	mux.HandleFunc("POST /items", handleCreateItem)

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

func handleGreet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "stranger"
	}
	fmt.Fprintf(w, "Hello, %s!\n", name)
}

func handleListItems(w http.ResponseWriter, r *http.Request) {
	all := make([]Item, 0, len(items))
	for _, item := range items {
		all = append(all, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all)
}

func handleGetItem(w http.ResponseWriter, r *http.Request) {
	item, ok := items[r.PathValue("id")]
	if !ok {
		http.Error(w, "item not found", http.StatusNotFound)
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

	item.ID = strconv.Itoa(len(items) + 1)
	items[item.ID] = item

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}
```

Around 80 lines, no dependencies, and it's a working JSON API.

> `handleListItems` builds a **slice** before encoding, so the JSON is an array
> `[...]` rather than an object. Note that map iteration order in Go is
> deliberately random — call it twice and the items come back in a different
> order. Sorting is left as an exercise.

---

# Part 2 — The same thing in Gin

Now we build the identical API with [Gin](https://gin-gonic.com), the most
widely used Go web framework.

**The point of this part is not to switch.** It's so that you can read the
framework code you'll meet in real projects, and decide for yourself whether you
need one.

## Install it

In your project folder:

```bash
go get github.com/gin-gonic/gin
```

This downloads Gin and records it in `go.mod`. You'll also see a new file,
`go.sum`, holding checksums of exactly what was downloaded. Both belong in
version control.

## The code

```go
package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var items = map[string]Item{
	"1": {ID: "1", Name: "Keyboard", Price: 45},
	"2": {ID: "2", Name: "Monitor", Price: 220},
}

func main() {
	r := gin.Default()

	r.GET("/health", handleHealth)
	r.GET("/greet", handleGreet)
	r.GET("/items", handleListItems)
	r.GET("/items/:id", handleGetItem)
	r.POST("/items", handleCreateItem)

	r.Run(":8080")
}

func handleHealth(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func handleGreet(c *gin.Context) {
	name := c.DefaultQuery("name", "stranger")
	c.String(http.StatusOK, "Hello, %s!\n", name)
}

func handleListItems(c *gin.Context) {
	all := make([]Item, 0, len(items))
	for _, item := range items {
		all = append(all, item)
	}
	c.JSON(http.StatusOK, all)
}

func handleGetItem(c *gin.Context) {
	item, ok := items[c.Param("id")]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func handleCreateItem(c *gin.Context) {
	var item Item

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.ID = strconv.Itoa(len(items) + 1)
	items[item.ID] = item

	c.JSON(http.StatusCreated, item)
}
```

The `curl` commands from Part 1 all still work, unchanged. From the outside,
these two programs are indistinguishable.

## Side by side

| Task | `net/http` | Gin |
| --- | --- | --- |
| Handler signature | `func(w http.ResponseWriter, r *http.Request)` | `func(c *gin.Context)` |
| Register a route | `mux.HandleFunc("GET /items/{id}", h)` | `r.GET("/items/:id", h)` |
| Path parameter | `r.PathValue("id")` | `c.Param("id")` |
| Query parameter | `r.URL.Query().Get("name")` | `c.Query("name")` |
| Query with default | manual `if name == ""` | `c.DefaultQuery("name", "stranger")` |
| Read JSON body | `json.NewDecoder(r.Body).Decode(&v)` | `c.ShouldBindJSON(&v)` |
| Send JSON | 3 lines: header, status, encode | `c.JSON(status, v)` |
| Send an error | `http.Error(w, msg, code)` | `c.JSON(code, gin.H{"error": msg})` |
| Start the server | `http.ListenAndServe(":8080", mux)` | `r.Run(":8080")` |

Two handlers, unchanged logic. **Gin replaced two parameters with one, and three
lines with one.** That's the entire nature of the difference.

`gin.H`, incidentally, is just `map[string]any` with a shorter name.

## Gin is standing on `net/http`

Not a metaphor — you can reach through it:

```go
func handleGetItem(c *gin.Context) {
	r := c.Request     // *http.Request — the very same type from Part 1
	w := c.Writer      // wraps http.ResponseWriter

	method := r.Method             // works exactly as before
	id := r.PathValue("id")        // even this still works
}
```

And the reverse — a Gin router is an ordinary `http.Handler`, so you can hand it
to the standard library's server:

```go
r := gin.Default()
r.GET("/health", handleHealth)

log.Fatal(http.ListenAndServe(":8080", r))   // no r.Run() needed
```

That compiles and runs. `*gin.Engine` satisfies the `http.Handler` interface,
so `net/http` accepts it without knowing what Gin is.

**This is the thing to take away from Part 2.** Gin is not an alternative to
`net/http`. It's a layer sitting on top of it, and everything you learned in
Part 1 is still true underneath.

## What Gin actually buys you

Being fair to it — the convenience is real, and one feature is genuinely more
than sugar:

```go
type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"  binding:"required"`
	Price int    `json:"price" binding:"required,gt=0"`
}
```

With those `binding` tags, `ShouldBindJSON` rejects a missing name or a negative
price on its own. In Part 1 you wrote that check by hand — and you'd write
another one for every field of every request. On a large API, that adds up.

Gin also ships a middleware ecosystem (logging, CORS, auth, rate limiting) and
route groups for shared prefixes and behaviour:

```go
api := r.Group("/api/v1")
api.Use(authMiddleware())
api.GET("/items", handleListItems)   // becomes GET /api/v1/items, behind auth
```

The cost side: a dependency to keep updated, a custom `Context` type that only
works inside Gin, and handlers that can't be reused anywhere else. For a
learning project the trade is roughly neutral. Judge it per project, not once
and forever.

---

# Part 3 — Why every framework looks like this

Swap Gin for Echo and the same handler becomes:

```go
// Echo
func handleGetItem(c echo.Context) error {
	item, ok := items[c.Param("id")]
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "item not found"})
	}
	return c.JSON(http.StatusOK, item)
}
```

Near-identical to Gin, with one design difference: Echo handlers **return an
error**, which its central error handler deals with. Some people much prefer it.

Chi barely changes anything at all, because it doesn't invent a context type:

```go
// Chi — handlers stay standard net/http
func handleGetItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// ...identical to Part 1 from here
}
```

## The pattern

Every Go web framework provides the same four things:

1. **A router** — match method + path, extract path parameters
2. **A context object** — request and response bundled into one argument
3. **Shortcuts** — `c.JSON(...)` instead of header/status/encode
4. **A middleware mechanism** — run code before and after handlers

Learn those four ideas and you can pick up any of them in an afternoon. The
differences are naming and taste, not concepts.

| Framework | Built on `net/http`? | Character |
| --- | --- | --- |
| **Gin** | Yes | Most popular, big ecosystem, terse |
| **Echo** | Yes | Very similar to Gin; handlers return `error` |
| **Chi** | Yes | Minimal. Keeps standard handler signatures — closest to Part 1 |
| **Fiber** | **No** | Express-style API. See the warning below |

> **The one real exception: Fiber.** Fiber is *not* built on `net/http` — it
> uses a separate library called `fasthttp`. Its handlers are not
> `http.Handler`s, and the middleware, libraries, and testing tools that work
> with everything else in this table **do not work with Fiber**. It's fast, and
> that's the trade. Knowing this before you choose it is the point.

## So which should you use?

For this course: **`net/http`**, all the way through. It's one less thing to
learn, it's what everything else is built on, and since Go 1.22 the routing gap
that made frameworks mandatory is gone.

For real work, a reasonable default: start with `net/http`, and add Chi if you
want route groups and middleware helpers without leaving standard handlers.
Reach for Gin or Echo when the request-binding and validation savings are worth
the coupling — on a large API with many endpoints, they often are.

What you should **not** do is pick a framework before you can write the handler
without it. You've now done that, so the choice is yours to make on the merits.

---

## Exercises

Do these in the Part 1 (`net/http`) version.

1. **`DELETE /items/{id}`** — remove an item. Return `204 No Content` on
   success, `404` if it wasn't there.
2. **`PUT /items/{id}`** — replace an item. Reject the request if the id in the
   URL doesn't match the id in the body.
3. **Sort `GET /items`** by price, so the order stops changing between calls.
4. **Add `GET /items?max=100`** — return only items under a given price.
   Remember `Query().Get()` gives you a *string*; look at `strconv.Atoi`, and
   decide what should happen when someone sends `?max=banana`.
5. **Break it deliberately:** write a handler that calls `w.WriteHeader` twice,
   run it, and read the exact warning your server prints. You will meet this
   message again for real one day.

---

## What you learned

- A web service is a program that waits for text over a network and writes text
  back — nothing more
- A handler returns nothing; it **writes** its response, in a fixed order
- Reading input: `r.PathValue`, `r.URL.Query().Get`, `json.NewDecoder(r.Body)`
- Writing output: `w.Header().Set`, `w.WriteHeader`, `json.NewEncoder(w)`
- Status codes are for programs. `4xx` is their fault, `5xx` is yours
- Always `return` after writing an error
- Frameworks are shortcuts over these same primitives — Gin's context holds the
  same `*http.Request` you started with
