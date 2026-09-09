# Go Backend Development Plan

A structured learning plan for building backend services in Go, aimed at someone
who has **never developed a backend before**. No prior server, API, or database
experience assumed.

## Guiding principles

- **Standard library first.** Frameworks and ORMs come after the thing they
  abstract is understood.
- **One tool per job.** Every extra tool is another thing that can break during
  a lesson.
- **Everything runs locally.** No cloud accounts needed.

---

## 1. Prerequisites

You need three things installed. That's it — everything else gets introduced
later in the plan, when there's a reason for it.

| What | Why |
| --- | --- |
| **Go** | The language and its compiler. |
| **A code editor** | VS Code or GoLand. Gives you autocomplete and inline errors. |
| **Git** | For downloading code and saving your work. |

> Later modules add Docker (for the database). Don't install it yet.

---

### 1.1 Install Go

Get the latest version from **<https://go.dev/dl>**. Take whatever the page
offers as the newest release — this plan needs **Go 1.22 or newer**.

> **Don't install Go with `apt install golang` or `yum install golang`.**
> Those versions are usually years out of date, and you'll hit errors that the
> lessons don't cover.

#### Windows

1. Go to <https://go.dev/dl> and download the file ending in `.windows-amd64.msi`.
2. Double-click it and click through the installer. Accept the default location.
3. **Close every terminal window you have open**, then open a new one.
4. Check it worked:

   ```
   go version
   ```

   You should see something like `go version go1.xx.x windows/amd64`.

Step 3 matters. The installer sets up Go for *new* terminals only — an already
open terminal won't find it.

#### macOS

First, find out which chip you have:

```bash
uname -m
```

- `arm64` → download the file ending in **`.darwin-arm64.pkg`** (Apple Silicon: M1/M2/M3/M4)
- `x86_64` → download the file ending in **`.darwin-amd64.pkg`** (older Intel Macs)

Then double-click the `.pkg` and click through the installer.

Open a new terminal and check:

```bash
go version
```

<details>
<summary>Prefer Homebrew?</summary>

```bash
brew install go
```

This works fine and picks the right chip automatically.
</details>

#### Linux

1. Go to <https://go.dev/dl> and copy the link to the file ending in
   `.linux-amd64.tar.gz`.
2. Run these commands, replacing `X.Y.Z` with the version number you saw:

   ```bash
   # download it
   curl -LO https://go.dev/dl/goX.Y.Z.linux-amd64.tar.gz

   # remove any old install, then unpack the new one into /usr/local
   sudo rm -rf /usr/local/go
   sudo tar -C /usr/local -xzf goX.Y.Z.linux-amd64.tar.gz
   ```

3. Tell your terminal where Go lives. Run **one** of these, depending on your
   shell (run `echo $SHELL` if you're not sure):

   ```bash
   # bash
   echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc

   # zsh
   echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.zshrc
   ```

4. Open a new terminal (or run `source ~/.bashrc`) and check:

   ```bash
   go version
   ```

---

### 1.2 Install a code editor

Use **[VS Code](https://code.visualstudio.com)** — install it, then add the
**Go** extension by *Go Team at Google*. When it offers to install extra tools,
say yes.

**[GoLand](https://www.jetbrains.com/go)** also works and needs no setup.

You'll check that it's working properly in section 1.4.

---

### 1.3 Install Git

Download from **<https://git-scm.com/downloads>**, or use your package manager
(`sudo apt install git`). Check with `git --version`.

You'll also want a free account at <https://github.com>.

---

### 1.4 Write your first Go program

This confirms Go *and* your editor work. Written for VS Code — GoLand notes at
the end.

---

#### Step 1 — Create the project folder

**File → Open Folder…** → click **New Folder** → name it `hello-go` → open it.

The Explorer panel on the left now shows **HELLO-GO**.

> Asked *"Do you trust the authors of the files in this folder?"* — click
> **Yes, I trust the authors**. It's your own folder.

---

#### Step 2 — Open the terminal

Press **``Ctrl + ` ``** (backtick), or **View → Terminal**.

A panel opens at the bottom, already inside `hello-go`.

---

#### Step 3 — Turn the folder into a Go project

Type into that terminal:

```bash
go mod init example/hello
```

A file called **`go.mod`** appears in the Explorer.

*Every Go project needs one — it names your project and tracks the outside
packages you'll add later.*

---

#### Step 4 — Create the code file

In the Explorer, hover over **HELLO-GO** → click the **New File** icon → name
it `main.go`.

*`main.go` is the conventional name for a program's starting point.*

---

#### Step 5 — Type the program

**Type it, don't paste** — you want to see autocomplete working.

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, backend!")
}
```

| Line | Meaning |
| --- | --- |
| `package main` | A runnable program, not a library. |
| `import "fmt"` | Bring in `fmt`, the standard package for printing text. |
| `func main()` | Where the program starts. |
| `fmt.Println(...)` | Print a line of text. |

Two things to notice:

- Typing `fmt.` pops up a suggestion list. **If it doesn't, your editor tooling
  isn't set up** — see [1.5](#15-troubleshooting).
- Saving (**`Ctrl+S`** / **`Cmd+S`**) reformats the file automatically. That's
  normal — Go has one official style, so nobody argues about it.

---

#### Step 6 — Run it

Either one:

- Click the grey **`run`** link that sits just above `func main()`.
- Or type `go run .` in the terminal.

The terminal prints:

```
Hello, backend!
```

**Setup complete.** You just wrote, compiled, and ran a Go program.

---

#### Step 7 — Break it on purpose

Delete the closing quote after `backend!`:

```go
	fmt.Println("Hello, backend!)
```

A red squiggle appears within a second or two, and the filename turns red in
the Explorer. That's your editor catching mistakes *before* you run anything.

Put the quote back and save — the red disappears.

---

Keep `hello-go` for scratch experiments, or delete it.

<details>
<summary>Doing this in GoLand instead</summary>

1. **File → New Project → Go**, name it `hello-go`, click **Create**.
2. Right-click the project folder → **New → Go File**, name it `main`.
3. Type the same program as above.
4. Click the green **▶** arrow in the left gutter next to `func main()`.

GoLand creates the `go.mod` for you, so there's no `go mod init` step.
</details>

---

### 1.5 Troubleshooting

#### `go: command not found` (or `'go' is not recognized...` on Windows)

Your terminal doesn't know where Go is installed.

1. **Did you open a new terminal after installing?** The old one won't pick it
   up. Close it and open a fresh one.
2. Still broken? Check whether Go is actually on disk:
   - Windows: `dir "C:\Program Files\Go\bin"`
   - macOS / Linux: `ls /usr/local/go/bin`
3. If the folder exists but the command doesn't work, the PATH step was missed.
   On macOS/Linux, redo step 3 of the Linux instructions. On Windows, re-run the
   `.msi` installer.

#### `go version` prints an old version, like `go1.18`

You have a second, older Go installed — usually from `apt` or Homebrew.

```bash
# see which one is actually being used
which go        # macOS / Linux
where go        # Windows (cmd)

# remove the old one
sudo apt remove golang-go golang      # Debian / Ubuntu
brew uninstall go                     # macOS Homebrew
```

Then open a new terminal and run `go version` again.

#### macOS: "bad CPU type" or the installer refuses to run

You downloaded the wrong file for your chip. Run `uname -m` and re-download —
`arm64` needs the `darwin-arm64` file, `x86_64` needs `darwin-amd64`.

#### Linux: `tar: ... Cannot open: Permission denied`

You left off `sudo`. Both the `rm -rf` and `tar` commands need it.

#### No autocomplete in VS Code

Press `Ctrl+Shift+P` (`Cmd+Shift+P` on macOS), run **Go: Install/Update Tools**,
select everything, then restart VS Code.

#### `go: cannot find main module`

You're in a folder that isn't a Go project. Either `cd` into your project
folder, or create one with:

```bash
go mod init example/myproject
```

#### Downloading packages fails: `dial tcp: i/o timeout`

Your network is blocking Go's package server — common on office or school
Wi-Fi. Try:

```bash
go env -w GOPROXY=https://goproxy.io,direct
```

Then retry. If you're on a corporate laptop with a proxy, ask IT for the proxy
address and set `HTTP_PROXY` / `HTTPS_PROXY`.

#### Something else

Run this and share the output when asking for help — it tells whoever's helping
you almost everything they need:

```bash
go version
go env GOROOT GOPATH GOPROXY
```

---

## 2. Knowledge prerequisites

_To be written — what to understand before writing Go: HTTP, JSON, SQL, and
command-line basics._

## 3. Curriculum

| # | Lesson | Covers |
| --- | --- | --- |
| 1 | **[HTTP Services](./readme-lesson1-httpservices.md)** | Requests, responses, routing, JSON, status codes — in one `main.go`. Then the same API in Gin, and why every framework looks alike. |
| 2 | **[Databases](./readme-lesson2-database.md)** | The database landscape, Postgres in Docker, SQL by hand, and pgx. No ORM. |
| 3 | _Structuring code_ | Getting the SQL out of the handlers |
| 4 | _Testing_ | — |
| 5 | _Caching with Redis_ | — |
| 6 | _Concurrency_ | — |
