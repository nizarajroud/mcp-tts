---
name: speak
description: Announces plans, issues, and summaries out loud using TTS. Use this skill PROACTIVELY after completing major tasks like finalizing a plan, resolving an issue, or generating a summary. Defaults to the local macOS `say` voice (zero config, always works); optional per-project voices from installed `say` voices or cloud providers (google, openai, elevenlabs) can distinguish projects and message types.
---

# Speak

Announce plans, issues, and summaries aloud. Triggered automatically after major milestones.

The default path is **zero-config**: call `say_tts` (macOS, no API key, always works) with `voice` unset. Explicit installed `say` voices and cloud voices are optional enhancements when distinct per-project or per-message identity is wanted. Do not build cloud config or try cloud providers unless a saved config or the user selects one.

## When to Announce

Announce automatically after a milestone — or when you are blocked and need the user:
- **Planning complete** - a plan or todo list is finalized
- **Issue resolved** - a bug fix or error is resolved
- **Summary generated** - a sprint or major task is completed
- **Input needed** - you have a question, or need a decision, direction, or approval before continuing. The user may be away or running several agents, so speak up so they know to come back.

## When NOT to Announce

Stay silent for:
- Trivial or intermediate steps (single edits, mid-task progress, routine tool calls)
- Output that is already short text the user is reading right now
- Non-interactive, headless, or CI runs (no one is listening)
- When the user has asked for quiet, or already heard an announcement for the same milestone
- More than once per logical milestone — do not narrate every step

## Identify the Speaker

Many agents may be announcing from different projects and tmux sessions at the same time, so every announcement MUST start by saying who is speaking. Lead with a short spoken label, then the message:

> "<speaker> says: <message>"

Determine the label once and cache it as `speaker` in `.claude/tts-config.json`:
- Default to the project directory name:
  ```bash
  basename "$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
  ```
- Override via the `speaker` field in config for a custom phrase (e.g. `"Claude on the auth service"`).

Keep the preamble short — it is spoken before every announcement. Good: "mcp-tts says: ...", "On the auth service, ...", "Claude on project X: ...". This identifies the source even when several projects share the same `say` voice.

## Choosing a Provider (fail fast — never probe secrets)

Pick the provider **once**, cheaply. Do not inspect environment variables or shell startup files to discover credentials, and do not call every cloud provider just to see which one fails.

1. **Reuse a saved choice** - if `.claude/tts-config.json` exists, use it and skip detection.
2. **No saved choice -> use `say_tts` directly.** Leave `voice` unset unless Voice Identity has intentionally assigned exact installed `say` voices. This is the common case and must be instant.
3. **Use a cloud provider only when the user or saved config selects one.** Preference is google, then openai, then elevenlabs, with `say_tts` as the final fallback. Optionally assign per-message voices (see Voice Identity).
4. **On cloud auth/config errors, fall back.** Mark that provider unavailable in `.claude/tts-config.json` and use the next saved provider or `say_tts`.
5. **Persist intentional choices** in `.claude/tts-config.json` so later announcements skip detection.

`say_tts` always works as the last resort and does not require credential discovery.

## Workflow

1. Detect the message type — planning, issue, summary, or question (input needed).
2. Pick the provider (above) — cached from `.claude/tts-config.json` if present, else use `say_tts`.
3. Transform the text to speech-friendly form (see Text Transformation), and prepend the speaker label (see Identify the Speaker).
4. Call the chosen TTS tool. On error, fall back per the Error Handling table; `say_tts` is the guaranteed final step.

## Error Handling

An announcement is best-effort — never let it block or derail the main task. On any failure, fall back; the final fallback (`say_tts`) needs no key.

| Error pattern | Action |
|---------------|--------|
| "API key", "unauthorized", "authentication", "...API_KEY is not set" | Mark provider unavailable in `.claude/tts-config.json`, use next |
| "rate limit", "quota", "429" | Use next provider (temporary) |
| "402", "payment", "paid_plan_required" (ElevenLabs library voice on a free-tier key) | Switch to a premade voice (e.g. unset `ELEVENLABS_VOICE_ID`) or use next provider |
| Other errors | Use next provider |

**Persist auth/config failures.** When a provider fails for a missing key, add it to `unavailable_providers` in `.claude/tts-config.json` so it is skipped next time:
```json
{
  "provider_order": ["openai", "say"],
  "unavailable_providers": ["google"]
}
```

## Text Transformation

Convert verbose output to conversational speech:

| Remove/Replace | With |
|----------------|------|
| URLs | "see the link" or omit |
| Code blocks | "see the code changes" or brief description |
| File paths | Just the filename (e.g., `/src/lib/foo.rs` -> "foo.rs") |
| Long hashes/IDs | "a commit hash" or omit |
| Long number lists | "several values" or count |
| Markdown formatting | Plain text |
| Technical jargon | Simpler alternatives when possible |

**Target length**: ~15-30 seconds of speech (roughly 50-100 words).

**Tone by type**:
- Planning: "Here's the plan..." (forward-looking, organized)
- Issue: "Found a problem..." (alert but calm)
- Summary: "All done..." (satisfied, accomplished)
- Question: "I need your input on..." (direct — clearly state the question or decision, then stop and wait)

A `question` announcement reuses the `issue` voice if no dedicated voice is assigned (both signal the user is needed).

## TTS Tools

### say_tts (default — local, free, no API key)
```
mcp__mcp-tts__say_tts
- text: string (required)
- voice: string (optional; any installed macOS voice — see `say -v '?'`)
- rate: integer (50-500; recommended 200-250; default 200)
```
- Prefer leaving `voice` unset to use the system default — it usually sounds most natural.
- A bare `say` command uses the host's configured System Voice, which can be a Siri voice. Do not set a fallback voice merely because `voice` is absent.
- If Voice Identity intentionally chooses a `say` voice, pass only an exact installed name from `/usr/bin/say -v '?'`.
- Rate hard limit is 50-500; keep 200-250 for comfortable listening, go higher only when the user explicitly asks.

### google_tts (cloud, preferred when configured)
```
mcp__mcp-tts__google_tts
- text: string (required)
- voice: string (default: "Kore")
- model: string (default: "gemini-3.1-flash-tts-preview")
```
Voices: Achernar, Achird, Algenib, Algieba, Alnilam, Aoede, Autonoe, Callirrhoe, Charon, Despina, Enceladus, Erinome, Fenrir, Gacrux, Iapetus, Kore, Laomedeia, Leda, Orus, Puck, Pulcherrima, Rasalgethi, Sadachbia, Sadaltager, Schedar, Sulafat, Umbriel, Vindemiatrix, Zephyr, Zubenelgenubi

### openai_tts (cloud fallback)
```
mcp__mcp-tts__openai_tts
- text: string (required)
- voice: string (default: "alloy") - alloy, ash, ballad, coral, echo, fable, nova, onyx, sage, shimmer, verse
- model: string (default: "gpt-4o-mini-tts-2025-12-15") - accepted: gpt-4o-mini-tts-2025-12-15, tts-1, tts-1-hd
- speed: number (0.25-4.0, default: 1.0)
- instructions: string (voice modulation hints)
```
`voice` and `model` are enforced enums — use only the values listed above, do not improvise.

### elevenlabs_tts (cloud fallback)
```
mcp__mcp-tts__elevenlabs_tts
- text: string (required)
```
- Voice/model are not tool parameters; they come from the server defaults (premade "Sarah") or the `ELEVENLABS_VOICE_ID` / `ELEVENLABS_MODEL_ID` env vars.
- **Free-tier API keys can only use premade voices.** A Voice Library (community/professional) voice returns HTTP 402 `paid_plan_required` — handle per the Error Handling table.

## Voice Identity (optional, cloud or explicit say voices)

Skip this unless distinct per-project or per-message voices are wanted. It exists so each project and message type can be recognized from another room.

For cloud providers:
1. Read `references/voice-pools.json` for candidate voices per provider and message type.
2. Check `~/.claude/tts-assignments.json` for voices already used by other projects (avoid reuse).
3. Pick one voice per message type (planning/issue/summary) from the configured provider's pool.
4. Save to `.claude/tts-config.json` and record in `~/.claude/tts-assignments.json`.

For macOS `say`:
1. Run `/usr/bin/say -v '?'` and parse the exact installed voice names.
2. Prefer installed Premium voices first, then Enhanced voices, then stable legacy voices such as `Samantha` or `Alex`.
3. Choose different voices for `planning`, `issue`/`question`, and `summary` only when good matches exist.
4. If a message type has no good installed candidate, leave `voice` unset for that type so the host System Voice is used.
5. Do not infer that the host default is broken, and do not replace an unset voice as a fallback. Passing a voice is an intentional identity choice only.

Example `.claude/tts-config.json`:
```json
{
  "speaker": "mcp-tts",
  "provider_order": ["google", "say"],
  "unavailable_providers": [],
  "voices": {
    "planning": { "provider": "google", "voice": "Kore" },
    "issue": { "provider": "google", "voice": "Aoede" },
    "summary": { "provider": "google", "voice": "Charon" }
  }
}
```

## Examples

Each example leads with the speaker label so the listener knows which project/agent is talking.

**Planning** (after TodoWrite with multiple items):
> "mcp-tts says: Here's the plan for the authentication feature. First, I'll create the login component. Then add session management. Finally, write the tests. Three tasks total."

**Issue** (after fixing an error):
> "mcp-tts says: Found and fixed an issue. The rate limiter wasn't catching timeout errors. Added a try-catch block in the handler. Tests are passing now."

**Summary** (after completing a feature):
> "mcp-tts says: All done with the authentication system. Added login, logout, and session management. Created five new files and updated the main router. Ready for review."

**Question** (blocked, needs a decision):
> "mcp-tts says: I need your input. Should the session tokens expire after one hour or stay valid for a day? I'll wait for your call before wiring up the middleware."
