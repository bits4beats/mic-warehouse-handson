# Phase 01 — MIC Monolith Baseline

> Italian version: [`README-IT.md`](./README-IT.md)

```
   __  __ ___ ___
  |  \/  |_ _/ __|     Modernizing with AI
  | |\/| || | (__      -------------------
  |_|  |_|___\___|     From monolith to microservices,
                       with an AI agent as copilot.
```

> **MIC** is a nod to **FIC** (FattureInCloud) and to the author "**Mic**" (Michele
> Mondora). MIC is a fictional **invoicing SaaS for Italian SMBs**, designed as a training
> ground for the **Strangler Fig** pattern: we extract Bounded Contexts from the monolith into Go
> microservices, one piece at a time, with an AI coding agent as copilot.

> Inspired by FattureInCloud, built to be modernized.

Estimated time: ~1h activity in breakout rooms, then a shared restitution.

---

## Your mission

You have just inherited **MIC**, a legacy PHP invoicing monolith. Nobody on the team has written
down how it works: what it does, how it is built, or where its parts are tangled together. Before
anyone can modernize it, the team needs that shared understanding.

**Bring MIC up, explore it, and produce the five artifacts below.** The artifacts *are* the
deliverable of this phase. Treat them as documents you would hand to a teammate joining tomorrow.

You have an **AI coding agent**. Use it as your exploration engine: ask it to read the code, the
GUI, and the database with you. But you own the conclusions, and you decide what goes in the
artifacts.

> 🛠 **Start it in the right folder.** Open your AI agent with `phase-01-monolith/` as its working
> directory (or `phase-01-monolith/php-app/` when you focus on the application code). Pointing it at
> the phase folder, not the whole repo, is the single most useful habit: it sees the code that
> matters and is not distracted by the rest.

> 💡 This is the first checkpoint of a multi-lesson extraction. What you learn to *see* here (the
> screens, the components, the way data is stored, the couplings) is exactly what the next phases
> will act on. The better your map, the easier every later phase becomes.

### Deliverables

Produce these as files (markdown, or a diagram image where it helps). Keep them in your working
copy; you will compare them with other rooms during the restitution.

| # | Artifact | What it must answer |
|---|---|---|
| 1 | **Page map** — `page-map.md` | Every screen the app exposes, and what each one is for. Group them however makes sense to you. |
| 2 | **Architecture diagram** — `architecture.md` (or an image) | The running components (containers, app, database) and how a request flows through them from browser to database and back. |
| 3 | **Repo guide** — `repo-guide.md` | What each top-level file/folder of `phase-01-monolith/` is and does. Enough that a newcomer knows where to look. |
| 4 | **Coupling map** — `coupling-map.md` | How the business data is actually stored, and which parts of the domain read or write each other's data. Name the couplings that a future extraction would have to break. |
| 5 | **Worst-of list** — `worst-antipatterns.md` | The **top 3 worst architectural choices / antipatterns** you find. For each: what it is, why it hurts, and what it would cost the team to keep living with it. |

> Don't aim for exhaustive perfection: aim for a map a newcomer could actually use. Capturing the
> *shape* of the system and its *worst* couplings is worth more than listing every column.

### Exit questions (your own self-check)

By the end you should be able to answer these in your own words, pointing at evidence in your
artifacts:

> *If we wanted to carve one piece of MIC out into its own service first, which dependency between
> two parts of the domain would we have to deal with, and why?*

> *Which single design choice makes MIC hardest to change safely, and what would you do about it?*

> *List the functional domains that show up in MIC: the distinct areas of the business the app
> deals with. Where does each one live in the screens, the code, and the data?*

If you can answer these from your own artifacts, you are ready for Phase 02.

---

## Prerequisites

Everything is containerized: nothing "application-level" to install on the host.

> Rancher Desktop running with the Docker engine, a browser, and a terminal. The Bash smoke checks
> use `curl` + `jq`; the Windows PowerShell path below uses `Invoke-RestMethod` and does not require
> `jq`. Go is not needed until the later Go service phases.

---

## Run

### Step 0 — Free the ports first

Phase folders reuse host ports. If another phase or an old MIC stack is still running, Docker may
fail with an error like:

```text
Bind for 0.0.0.0:8088 failed: port is already allocated
```

From `phase-01-monolith/`, stop any previous MIC containers before starting:

```bash
docker compose down
```

Quick port check:

```bash
docker ps --filter "publish=8088" --filter "publish=8082" --filter "publish=3306"
# expected before start: no rows
```

Run this before Step 1, not after Docker has already failed.

### Step 1 — Build and start

From the **repository root**:

```bash
cd phase-01-monolith
docker compose up --build
```

### Step 2 — Open the app and the database browser

| URL                                       | What                                              |
|-------------------------------------------|---------------------------------------------------|
| http://localhost:8088                     | MIC GUI                                           |
| http://localhost:8088/api/dashboard/kpi   | API smoke check                                   |
| http://localhost:8082                     | Adminer (server `mysql`, db `mic`, user/pass `root`/`root`) |
| `localhost:3306`                          | Direct MySQL                                      |

The schema is created and seeded automatically on the first boot of the `mysql` container. To
recreate from scratch:

```bash
docker compose down -v && docker compose up --build
```

### Step 3 — Confirm it is up

Two quick calls. If both return coherent JSON, the monolith is running and you can start exploring.

```bash
# Business KPIs (envelope { data })
curl -s http://localhost:8088/api/dashboard/kpi | jq '.data | keys'

# A paginated list (envelope { data, meta })
curl -s "http://localhost:8088/api/articles?limit=5" | jq '{first: .data[0], total: .meta.total}'
```

<details>
<summary><b>Windows PowerShell</b> (no <code>jq</code> needed)</summary>

```powershell
$base = "http://localhost:8088"
(Invoke-RestMethod "$base/api/dashboard/kpi").data | Get-Member -MemberType NoteProperty | Select-Object Name
$r = Invoke-RestMethod "$base/api/articles?limit=5"; "total=$($r.meta.total)"
```

On Windows, the simplest path is **Git Bash** (bundled with Git for Windows): the `curl + jq`
commands above work as-is.
</details>

### Step 4 — Shutdown

```bash
docker compose down
```

Use `docker compose down -v` only when you intentionally want to delete the MIC database volume and
reload seed data from scratch.

---

## How to explore

Three lenses. Use all three; they show you different things.

1. **The GUI** (http://localhost:8088). Click through every section in the sidebar. Create an order,
   generate an invoice. Notice what the app lets a user *do*: that is the surface your page map and
   parts of your coupling map come from.
2. **The database** (Adminer, http://localhost:8082). Browse the tables. Look at how a business
   record you just created in the GUI is actually stored. This is where the most important findings
   for your **coupling map** and **worst-of list** live: the storage shape is not what the GUI suggests.
3. **The code** (`phase-01-monolith/`, with your AI agent). Read the controllers, the models, the
   router. Trace one request end to end. This feeds your **repo guide** and **architecture diagram**.

> Treat the AI as a fast pair, not an oracle. When it asserts a coupling or a column meaning, ask
> *"where in the code or schema did you see that?"* before you write it into an artifact.

---

## Success criteria

You are done with Phase 01 when:

- [ ] `docker compose up --build` starts `mic-app`, `mysql`, and `adminer`, and `http://localhost:8088` opens the GUI.
- [ ] The two confirm-it-is-up calls return coherent JSON.
- [ ] You have produced all five artifacts (page map, architecture diagram, repo guide, coupling map, worst-of list).
- [ ] Your coupling map describes **how the business data is stored** and names at least one cross-domain coupling, with evidence.
- [ ] Your worst-of list names the **top 3 antipatterns** with a concrete reason for each.
- [ ] You can answer the **exit questions** from your own artifacts.

---

## References

- [`ADR-001`](../docs/adr/ADR-001-strangler-fig-pattern.md) — Strangler Fig migration strategy.
- [`ADR-006`](../docs/adr/ADR-006-php-monolite-implementation.md) — MIC PHP monolith baseline decision.
- [`openapi.yaml`](./openapi.yaml) — legacy API contract for the MIC routes.

---

## Solutions

Reference versions of all five deliverables live under `solutions/` on the **`lezione-7-soluzione`**
branch: `page-map.md`, `architecture.md`, `repo-guide.md`, `coupling-map.md`, `worst-antipatterns.md`.
The branch you clone (`lezione-7`) is the starting point and has none.

**Switch to the solution only after you have produced your own artifacts.** The value of this
phase is in exploring and getting it slightly wrong, not in reading the answer.

> To see it: `git switch lezione-7-soluzione`, or browse that branch on the repository host.

---

## Next Phase

→ [Phase 02](../phase-02-analysis/) — Strategic and tactical DDD analysis. Turn what you discovered
into clean aggregate boundaries and an extraction plan.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
