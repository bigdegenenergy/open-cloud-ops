# MCP Server Setup Guide

## Puppeteer MCP Server Installation

This project uses the **Puppeteer MCP Server** for browser automation capabilities. The server is installed via a pinned npm dependency for security and reproducibility.

### ⚠️ Security Note

The MCP server is specified with a pinned version (`package.json`) rather than using unpinned `npx` to prevent supply-chain attacks. This ensures:
- Reproducible builds across environments
- Protection against compromised or malicious package versions
- Transparent dependency management via `package-lock.json`

### Installation

**⚠️ REQUIRED: One-time setup (must run before using the agent):**

```bash
cd tools/autonomous-coding/
npm install
```

This creates:
- `node_modules/puppeteer-mcp-server/` (locked to version 1.2.0)
- `package-lock.json` (integrity lock file with real cryptographic hashes)

**CRITICAL:** You MUST run `npm install` to generate the actual `package-lock.json` with valid integrity hashes. The repository may include a template lockfile, but it requires this step to be valid.

After running `npm install`, commit both:
- `package-lock.json` (with real hashes and dependencies)
- `node_modules/` is NOT committed (git-ignored; recreated by `npm install`)

**Why this matters:**
- The `package-lock.json` ensures reproducible builds across all environments (local, CI/CD, etc.)
- Integrity hashes prevent supply-chain attacks (confirms you get the exact version you tested)
- Without it, different machines may pull different versions, leading to non-reproducible behavior or security regressions

**Verify installation:**

```bash
npm list puppeteer-mcp-server
```

Expected output:
```
autonomous-coding-agent@1.0.0 /path/to/tools/autonomous-coding
└── puppeteer-mcp-server@1.2.0
```

### How It Works

1. **`package.json`** specifies the pinned version: `^1.2.0`
2. **`npm install`** installs to `node_modules/` and creates `package-lock.json`
3. **`npm exec puppeteer-mcp-server`** runs the pinned version from `node_modules/`
4. **No unpinned `npx` calls** — all MCP servers are reproducibly locked

### Updating the MCP Server

To update to a newer version:

```bash
npm update puppeteer-mcp-server
# or explicitly pin a version:
npm install puppeteer-mcp-server@1.3.0
```

Then commit the updated `package-lock.json`:

```bash
git add package-lock.json
git commit -m "chore: update puppeteer-mcp-server to 1.3.0"
```

### Troubleshooting

**"puppeteer-mcp-server not found"**
- Run: `npm install`
- Ensure `node_modules/puppeteer-mcp-server/` exists

**"command not found: npm"**
- Install Node.js: https://nodejs.org/
- Requires Node.js 18+ and npm 9+

**Version mismatch**
- Delete `node_modules/` and `package-lock.json`
- Run: `npm install`
- Verify: `npm list puppeteer-mcp-server`

### CI/CD Integration

In GitHub Actions workflows, ensure the setup step includes:

```yaml
- name: Install MCP dependencies
  run: |
    cd tools/autonomous-coding/
    npm install
```

This guarantees the pinned version is available in all environments.
