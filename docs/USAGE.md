# Mnemon — Usage & Reference

> You don't run mnemon commands yourself — the agent does, driven by hooks and guided by the skill file. This document is a reference for understanding what the agent can do, for debugging, and for advanced manual operation.

---

## Global Flags

These flags are available on every command:

| Flag | Default | Description |
|---|---|---|
| `--store <name>` | (auto) | Named memory store (overrides `MNEMON_STORE` and active file) |
| `--data-dir <path>` | `~/.mnemon` | Base data directory |
| `--version` | | Print version and exit |

---

## Setup

Deploy mnemon into LLM CLI environments. This is the first command to run after installation.

```bash
# Interactive: detect environments and install (project-local)
mnemon setup

# User-wide install (all projects)
mnemon setup --global

# Non-interactive: specific target only
mnemon setup --target claude-code
mnemon setup --target openclaw

# Auto-confirm all prompts (CI-friendly)
mnemon setup --yes

# Remove mnemon integrations
mnemon setup --eject
mnemon setup --eject --target claude-code
```

| Flag | Default | Description |
|---|---|---|
| `--global` | `false` | Install to user-wide config (`~/.claude/`) instead of project-local (`.claude/`) |
| `--target <name>` | (auto-detect) | Target environment: `claude-code` or `openclaw` |
| `--eject` | `false` | Remove mnemon integrations |
| `--yes` | `false` | Auto-confirm all prompts |

---

## CLI Commands

### Core

```bash
# Remember — store a new insight (built-in diff: duplicates skipped, conflicts auto-replaced)
mnemon remember "Chose Qdrant over Milvus for vector search" \
  --cat decision --imp 5 --entities "Qdrant,Milvus" --tags "architecture,search" --source agent

# Skip duplicate/conflict detection
mnemon remember "Raw note" --no-diff

# Recall — intent-aware graph-enhanced retrieval (default)
mnemon recall "vector database" --limit 10

# Recall with explicit intent override
mnemon recall "why did we choose Qdrant" --intent WHY

# Recall with category/source filter
mnemon recall "auth" --cat decision --source agent

# Simple SQL LIKE matching (faster, no graph traversal)
mnemon recall "auth" --basic

# Search — token-scored keyword search
mnemon search "authentication" --limit 10

# Forget — soft-delete an insight
mnemon forget <id>
```

**Remember flags:**

| Flag | Default | Description |
|---|---|---|
| `--cat` | `general` | Category: `preference`, `decision`, `fact`, `insight`, `context`, `general` |
| `--imp` | `3` | Importance: 1–5 |
| `--tags` | | Comma-separated tags |
| `--entities` | | Comma-separated entities (merged with auto-extraction) |
| `--source` | `user` | Source: `user`, `agent`, `external` |
| `--no-diff` | `false` | Skip duplicate/conflict detection |

**Recall flags:**

| Flag | Default | Description |
|---|---|---|
| `--limit` | `10` | Max results |
| `--intent` | (auto-detect) | Override intent: `WHY`, `WHEN`, `ENTITY`, `GENERAL` |
| `--cat` | | Filter by category |
| `--source` | | Filter by source |
| `--basic` | `false` | Use simple SQL LIKE matching instead of smart recall |

### Graph Operations

```bash
# Link — create a typed edge
mnemon link <source_id> <target_id> --type semantic --weight 0.85
mnemon link <source_id> <target_id> --type causal --weight 0.8 \
  --meta '{"sub_type":"causes","reason":"..."}'

# Related — BFS traversal from an insight
mnemon related <id> --edge causal --depth 2
```

### Lifecycle Management

```bash
# GC — view low-retention candidates
mnemon gc --threshold 0.5 --limit 20

# GC keep — boost an insight's retention
mnemon gc --keep <id>
```

### Store Management

Mnemon supports named stores for data isolation. Each store has its own independent database.

```bash
# List all stores (* marks the active one)
mnemon store list

# Create a new store
mnemon store create work

# Switch the default active store
mnemon store set work

# Remove a store (cannot remove the active store)
mnemon store remove old-project
```

**Store resolution priority** (highest to lowest):

1. `--store <name>` CLI flag
2. `MNEMON_STORE` environment variable
3. `~/.mnemon/active` file
4. Falls back to `"default"`

Different agents or processes can use different stores via the `MNEMON_STORE` environment variable — no global state contention. Legacy databases (`~/.mnemon/mnemon.db`) are automatically migrated to `~/.mnemon/data/default/` on first run.

### Observability

```bash
mnemon status           # memory statistics
mnemon log              # operation log (default: last 20)
mnemon log --limit 50   # show more entries
```

### Audio Transcription

Transcribe a local audio file or public audio URL with a locally hosted Qwen ASR server. Upstream Mnemon operates on text memories; this command adds an optional local audio-to-text ingestion step without introducing a hosted transcription dependency.

Start a local Qwen ASR server:

```bash
docker run -p 17003:8000 quantatrisk/qwen3-asr:cpu-latest
```

Then transcribe:

```bash
mnemon transcribe ./voice-note.mp3
mnemon transcribe https://example.com/audio.wav --language English
mnemon transcribe ./meeting.mp3 --json --system-prompt "Names: Mnemon, OpenClaw"
mnemon transcribe ./meeting.mp3 --response-format verbose_json --speaker-diarization
```

NanoClaw/container deployment can run Qwen ASR as a sidecar service and point Mnemon at it:

```bash
mnemon transcribe ./voice-note.mp3 --endpoint http://qwen-asr:8000/v1/audio/transcriptions
```

Defaults:

| Flag | Default | Description |
|---|---|---|
| `--provider` | `qwen` | ASR provider |
| `--endpoint` | `http://localhost:17003/v1/audio/transcriptions` | Local Qwen ASR OpenAI-compatible transcription endpoint |
| `--model` | `qwen3-asr-0.6b` | Qwen ASR model |
| `--api-key` | *(empty)* | Optional bearer token for a protected local endpoint |
| `--language` | *(auto)* | Optional language hint, such as `English` or `Chinese` |
| `--system-prompt` | *(empty)* | Context text to bias recognition with names/domain terms |
| `--response-format` | `json` | Qwen response format sent to the server. Use `verbose_json` only when segment metadata is needed |
| `--speaker-diarization` | `false` | Enable speaker diarization. This is slower on CPU Qwen servers |
| `--json` | `false` | Include metadata such as detected language, model, and seconds |
| `--timeout` | `5m` | Request timeout. CPU Qwen servers can run much slower than realtime |

Mnemon defaults to the lightweight Qwen request path: `response_format=json` and `enable_speaker_diarization=false`. This avoids forcing segment/speaker metadata for ordinary transcript extraction, which is especially important on CPU-only Qwen deployments.

### Visualization

Export the knowledge graph for visual exploration:

```bash
# DOT format — render with Graphviz (brew install graphviz)
mnemon viz --format dot -o graph.dot
dot -Tpng graph.dot -o graph.png

# Interactive HTML — open directly in the browser (vis.js, no install needed)
mnemon viz --format html -o graph.html
open graph.html
```

Nodes are colored by category (decision, fact, insight, preference, context); edges are colored by type (temporal, semantic, causal, entity).

---

## Configuration

| Variable | Default | Description |
|---|---|---|
| `MNEMON_DATA_DIR` | `~/.mnemon` | Base data directory |
| `MNEMON_STORE` | `default` | Active named store |
| `MNEMON_EMBED_ENDPOINT` | `http://localhost:11434` | Ollama API endpoint |
| `MNEMON_EMBED_MODEL` | `nomic-embed-text` | Ollama embedding model |

---

## Embedding Support (Optional)

Mnemon works fully without Ollama — all core features (remember, recall, link, graph traversal) function out of the box. Adding Ollama enhances recall precision through vector similarity, but is never required.

### What changes with and without embeddings

| Capability | Without Ollama | With Ollama |
|---|---|---|
| **Recall anchors** | Keyword + recency | Keyword + vector + recency (RRF hybrid) |
| **Semantic edges** | Token overlap (coarser) | Cosine similarity ≥ 0.50 (precise) |
| **Traversal scoring** | Pure structural | Structural + semantic |
| **Rerank weights** | Keyword 45%, Entity 25%, Graph 30% | Keyword 30%, Entity 15%, Similarity 35%, Graph 20% |

When Ollama is unavailable, the reranking system automatically redistributes similarity weight to keyword and graph signals — no configuration needed, no degraded mode flag. The system detects Ollama availability at runtime with a 2-second timeout.

### Setup

```bash
brew install ollama              # or see https://ollama.ai
ollama pull nomic-embed-text     # download the embedding model
```

Verify with:

```bash
mnemon embed --status
```

```json
{
  "total_insights": 87,
  "embedded": 87,
  "coverage": "100%",
  "ollama_available": true,
  "model": "nomic-embed-text"
}
```

### Backfilling existing insights

If you install Ollama after already using mnemon, existing insights won't have embeddings. Backfill them in one command:

```bash
mnemon embed --all
```

This generates embeddings for all un-embedded insights and automatically creates semantic edges. You can check coverage before and after with `mnemon embed --status`.

---

## Architecture

```
┌──────────────────┐     CLI commands      ┌──────────────────┐
│   LLM Agent      │ ───────────────────── │     Mnemon       │
│ (Claude Code,    │  remember, recall,    │                  │
│  Cursor, etc.)   │  link, forget, gc     │  SQLite (WAL)    │
└──────────────────┘                       │  ┌────────────┐  │
                                           │  │ Insights   │  │
        The LLM decides WHAT               │  ├────────────┤  │
        to remember and link.              │  │ 4 Edge     │  │
                                           │  │ Types:     │  │
        Mnemon handles HOW                 │  │ temporal   │  │
        to store, index, and               │  │ entity     │  │
        retrieve.                          │  │ causal     │  │
                                           │  │ semantic   │  │
      ┌──────────────────┐                 │  ├────────────┤  │
      │  Ollama          │  (optional)     │  │ Embeddings │  │
      │  nomic-embed-text│ ◄───────────── │  └────────────┘  │
      └──────────────────┘                 └──────────────────┘
```

Inspired by [MAGMA](https://arxiv.org/abs/2601.03236) four-graph model. See [Design & Architecture](DESIGN.md) for the full deep dive.
