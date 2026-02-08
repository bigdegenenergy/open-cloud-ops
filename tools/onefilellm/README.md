# OneFileLLM Integration

This directory contains the integration for [OneFileLLM](https://github.com/jimmc414/onefilellm), a content aggregation tool that consolidates data from multiple sources into structured XML format for LLM consumption.

## What is OneFileLLM?

OneFileLLM extracts and combines content from diverse sources:

- **Local files and directories**
- **GitHub repositories**, branches, PRs, and issues
- **Web pages** with configurable crawling
- **ArXiv papers** and academic identifiers (DOI, PMID)
- **YouTube transcripts**
- **PDFs, Excel files, Jupyter notebooks**

Output is structured XML optimized for LLM context windows.

## GitHub Action Usage

The workflow automatically:

1. Runs OneFileLLM to aggregate content
2. Saves output to `docs/onefilellm/`
3. Creates a new branch (`onefilellm/output-TIMESTAMP`)
4. Opens a Pull Request
5. Auto-merges the PR (squash)
6. Deletes the branch

### Requirements

- **`GH_TOKEN` secret**: A Personal Access Token with `repo` scope (for creating branches and PRs)

### Auto-Merge Behavior

Auto-merge requires:

- Token with admin/write permissions
- Auto-merge enabled in repo settings
- No blocking branch protection rules

If auto-merge fails (insufficient permissions or repo settings), the workflow will:

- Still create the PR successfully
- Report that manual merge is required
- Not fail the workflow

### Manual Trigger (workflow_dispatch)

1. Go to **Actions** > **OneFileLLM Content Aggregator**
2. Click **Run workflow**
3. Enter your sources (space-separated)
4. Configure options as needed (output name is auto-generated if left empty)
5. The workflow creates a PR and merges it to `main`

### Auto-Generated Filenames

When `output_name` is left empty, the filename is auto-generated from the source:

| Source Type | Example Input                   | Generated Filename |
| ----------- | ------------------------------- | ------------------ |
| GitHub repo | `https://github.com/owner/repo` | `owner-repo.xml`   |
| Website     | `https://docs.example.com/api`  | `docs-example.xml` |
| Local path  | `./src/components`              | `components.xml`   |

### Example Inputs

```yaml
# Single GitHub repo
sources: "https://github.com/anthropics/claude-code"

# Multiple sources
sources: "https://github.com/user/repo https://docs.example.com/api"

# Local paths (relative to repo root)
sources: "./src ./docs/api.md"

# ArXiv paper
sources: "https://arxiv.org/abs/2301.00001"

# YouTube transcript
sources: "https://youtube.com/watch?v=VIDEO_ID"
```

### Workflow Options

| Option            | Default              | Description                                    |
| ----------------- | -------------------- | ---------------------------------------------- |
| `sources`         | (required)           | Space-separated input sources                  |
| `output_name`     | `aggregated-content` | Output filename (without .xml)                 |
| `format`          | `auto`               | Force format: text, markdown, json, html, yaml |
| `crawl_depth`     | `3`                  | Max recursion depth for web crawling           |
| `crawl_max_pages` | `100`                | Max pages to process when crawling             |

### Calling from Another Workflow

```yaml
jobs:
  aggregate:
    uses: ./.github/workflows/onefilellm.yml
    with:
      sources: "https://github.com/owner/repo"
      output_name: "repo-context"
      crawl_depth: "2"
    secrets: inherit # Pass GH_TOKEN

  # The output is auto-committed to docs/onefilellm/
  # You can also access the artifact:
  use-output:
    needs: aggregate
    runs-on: ubuntu-latest
    steps:
      - name: Download artifact
        uses: actions/download-artifact@v4
        with:
          name: repo-context

      - name: Use aggregated content
        run: |
          echo "PR URL: ${{ needs.aggregate.outputs.pr_url }}"
          echo "Token count: ${{ needs.aggregate.outputs.token_count }}"
          cat repo-context.xml
```

### Workflow Outputs

| Output        | Description                                           |
| ------------- | ----------------------------------------------------- |
| `output_file` | Path to generated file (`docs/onefilellm/<name>.xml`) |
| `token_count` | Estimated token count                                 |
| `pr_url`      | URL of the created Pull Request                       |

## Local Usage

### Installation

```bash
pip install -r tools/onefilellm/requirements.txt
```

### Basic Commands

```bash
# Aggregate a GitHub repo
onefilellm https://github.com/owner/repo

# Aggregate multiple sources
onefilellm ./local/path https://docs.site.com/api

# Web crawling with depth limit
onefilellm --crawl-max-depth 2 --crawl-max-pages 50 https://docs.example.com

# Force output format
onefilellm -f markdown https://github.com/owner/repo

# Save to file (instead of clipboard)
onefilellm https://github.com/owner/repo > output.xml
```

### Environment Variables

- `GITHUB_TOKEN`: For authenticated GitHub API access (higher rate limits, private repos)

## Output Format

OneFileLLM generates XML with semantic tags:

```xml
<onefilellm>
  <source type="github" url="https://github.com/owner/repo">
    <file path="README.md">
      [file content here]
    </file>
    <file path="src/main.py">
      [file content here]
    </file>
  </source>
</onefilellm>
```

## Use Cases

1. **Codebase context for Claude**: Aggregate an entire repo for code review or feature planning
2. **Documentation synthesis**: Combine multiple doc sources for comprehensive context
3. **Research compilation**: Gather ArXiv papers and related docs
4. **PR review context**: Pull in related files and documentation for thorough reviews

## Links

- [OneFileLLM GitHub](https://github.com/jimmc414/onefilellm)
- [OneFileLLM PyPI](https://pypi.org/project/onefilellm/)
