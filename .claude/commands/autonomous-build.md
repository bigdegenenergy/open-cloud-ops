---
description: Launch autonomous coding agent to generate production-quality applications
model: claude-opus-4.1
allowed-tools: Bash(*), Read(*), Write(*), Edit(*)
---

# Autonomous Coding Agent

Launch a two-agent autonomous coding system that can build complete applications over multiple sessions.

## Usage

```bash
# Start a fresh project
python tools/autonomous-coding/autonomous_agent_demo.py --project-dir ./my_app

# Continue an existing project
python tools/autonomous-coding/autonomous_agent_demo.py --project-dir ./my_app

# Limited iterations (for testing)
python tools/autonomous-coding/autonomous_agent_demo.py --project-dir ./my_app --max-iterations 3
```

## How It Works

### Two-Agent Pattern

**Session 1 (Initializer)**:
- Reads specification from `app_spec.txt`
- Generates 200 comprehensive test cases in `feature_list.json`
- Sets up project structure and git repository
- Takes 10-20+ minutes (normal behavior, agent is working)

**Sessions 2+ (Coding Agent)**:
- Picks up where previous session left off
- Implements features one by one
- Marks tests as passing in `feature_list.json`
- Auto-continues with 3-second delay between sessions

### Progress Tracking

Progress is persisted via:
- `feature_list.json` — source of truth for test status
- Git commits — each feature implementation
- `claude-progress.txt` — session notes

### Running the Generated App

After generation completes:

```bash
cd my_app
./init.sh           # Run the setup script
# Or manually:
npm install && npm run dev
```

Then open `http://localhost:3000` (or check `init.sh` for the exact URL).

## Security

Defense-in-depth approach:

1. **OS-level Sandbox** — Bash commands run in isolated environment
2. **Filesystem Restrictions** — Operations restricted to project directory
3. **Bash Allowlist** — Only safe commands permitted:
   - File: `ls`, `cat`, `head`, `tail`, `wc`, `grep`, `cp`, `mkdir`, `chmod`
   - Node.js: `npm`, `node`
   - Git: `git`
   - Process: `ps`, `lsof`, `sleep`, `pkill` (dev servers only)

See `tools/autonomous-coding/security.py` for details.

## Customization

### Change the Application Spec

Edit `tools/autonomous-coding/prompts/app_spec.txt` to specify a different application.

### Adjust Feature Count

Edit `tools/autonomous-coding/prompts/initializer_prompt.md` and change "200 features" to your desired count (smaller = faster demos, e.g., 20-50).

### Add Allowed Commands

Edit `tools/autonomous-coding/security.py` to add commands to `ALLOWED_COMMANDS` (use caution).

## Troubleshooting

**"Appears to hang on first run"**
- Normal. The initializer agent is generating 200 detailed test cases.
- Watch for `[Tool: ...]` output to confirm the agent is working.

**"Command blocked by security hook"**
- The agent tried to run a blocked command — the security system is working.
- Add to `ALLOWED_COMMANDS` if needed (security-reviewed).

**"API key not set"**
- Ensure `ANTHROPIC_API_KEY` is exported in your shell environment.

## Performance Expectations

- **First session**: 10-20+ minutes (test case generation)
- **Subsequent sessions**: 5-15 minutes per iteration
- **Full app** (200 features): Many hours across multiple sessions

The 200 features is designed for comprehensive coverage. Reduce to 20-50 for faster demos.
