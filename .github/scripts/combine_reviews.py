#!/usr/bin/env python3
"""Combine Gemini and Codex PR review outputs into a unified report.

Reads review JSON from environment variables, detects consensus issues
(flagged by both models), and generates a combined markdown comment
plus REVIEW_INSTRUCTIONS.md for the implementation agent.

Environment variables:
  GEMINI_JSON        - JSON string from Gemini review job
  CODEX_JSON         - JSON string from Codex review job
  GEMINI_HAS_RESULT  - "true" if Gemini produced results
  CODEX_HAS_RESULT   - "true" if Codex produced results
  PR_NUMBER          - Pull request number
  GITHUB_OUTPUT      - Path to GitHub Actions output file
"""

import json
import os
import sys


def load_review(raw, has_result):
    if has_result != "true" or not raw.strip():
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError as e:
        print(f"Warning: failed to parse review JSON: {e}", file=sys.stderr)
        return None


def collect_issues(gemini, codex):
    issues = []
    if gemini and gemini.get("issues"):
        for i in gemini["issues"]:
            i.setdefault("source", "gemini")
            issues.append(i)
    if codex and codex.get("issues"):
        for i in codex["issues"]:
            i.setdefault("source", "codex")
            issues.append(i)
    return issues


def _to_int(val):
    """Safely convert a value to int, returning None on failure."""
    if val is None:
        return None
    try:
        return int(val)
    except (TypeError, ValueError):
        return None


def detect_consensus(all_issues):
    gemini_files = {}
    codex_files = {}
    for i in all_issues:
        key = i.get("file", "").lower()
        if i.get("source") == "gemini":
            gemini_files.setdefault(key, []).append(i)
        else:
            codex_files.setdefault(key, []).append(i)

    consensus = []
    for f in set(gemini_files.keys()) & set(codex_files.keys()):
        for gi in gemini_files[f]:
            for ci in codex_files[f]:
                gt = gi.get("title", "").lower()
                ct = ci.get("title", "").lower()
                g_words = set(gt.split())
                c_words = set(ct.split())
                overlap = len(g_words & c_words)
                g_line = _to_int(gi.get("line") or gi.get("line_start"))
                c_line = _to_int(ci.get("line") or ci.get("line_start"))
                lines_close = (
                    g_line is not None
                    and c_line is not None
                    and abs(g_line - c_line) <= 3
                )
                if overlap >= 2 or lines_close:
                    gi["_consensus"] = True
                    ci["_consensus"] = True
                    consensus.append({"file": f, "gemini": gi, "codex": ci})

    return consensus


def determine_decision(gemini, codex):
    decisions = []
    if gemini:
        decisions.append(gemini.get("decision", "COMMENT"))
    if codex:
        decisions.append(codex.get("decision", "COMMENT"))

    if "REQUEST_CHANGES" in decisions:
        return "REQUEST_CHANGES"
    if all(d == "APPROVE" for d in decisions) and decisions:
        return "APPROVE"
    return "COMMENT"


def build_markdown(gemini, codex, all_issues, consensus, decision, pr_number):
    md = []

    decision_emoji = {"APPROVE": "\u2705", "REQUEST_CHANGES": "\U0001f534"}.get(
        decision, "\U0001f4ac"
    )
    md.append(f"## {decision_emoji} Combined AI PR Review\n")

    models_used = []
    if gemini:
        models_used.append("Gemini")
    if codex:
        models_used.append("Codex")
    md.append(f"**Models:** {' + '.join(models_used)}")
    md.append(f"**Combined Decision:** {decision}\n")

    if gemini and gemini.get("summary"):
        md.append(f"**Gemini Summary:** {gemini['summary']}")
    if codex and codex.get("summary"):
        md.append(f"**Codex Summary:** {codex['summary']}")
    md.append("")

    if consensus:
        md.append("### \U0001f91d Consensus Issues (flagged by both models)\n")
        md.append(
            "These issues were independently identified by both Gemini and Codex, "
            "indicating higher confidence:\n"
        )
        for c in consensus:
            gi = c["gemini"]
            md.append(
                f"- **`{gi.get('file')}`**: {gi.get('title')} _(both models agree)_"
            )
        md.append("")

    if all_issues:
        severity_groups = [
            ("Critical", "critical", "\U0001f534"),
            ("Important", "important", "\U0001f7e1"),
            ("Suggestions", "suggestion", "\U0001f7e2"),
        ]

        for label, sev, icon in severity_groups:
            group = [i for i in all_issues if i.get("severity") == sev]
            if not group:
                continue
            md.append(f"### {icon} {label}\n")
            for i in group:
                src = {
                    "gemini": "\U0001f535 Gemini",
                    "codex": "\U0001f7e0 Codex",
                }.get(i.get("source"), "\u2753")
                badge = " \U0001f91d" if i.get("_consensus") else ""

                md.append(f"#### #{i['id']}: {i.get('title', 'Untitled')}{badge}")
                md.append(f"- **Source:** {src}")
                md.append(f"- **File:** `{i.get('file', '?')}`")
                line = i.get("line") or i.get("line_start")
                if line:
                    md.append(f"- **Line:** {line}")
                desc = i.get("description") or i.get("body", "")
                if desc:
                    md.append(f"- **Details:** {desc}")
                if i.get("suggestion"):
                    md.append(f"> \U0001f4a1 {i['suggestion']}")
                md.append("")

        md.append(
            "<details><summary>\U0001f4cb JSON (for selective acceptance)</summary>\n"
        )
        md.append("```json")
        md.append(json.dumps({"issues": all_issues}, indent=2))
        md.append("```\n</details>\n")
        md.append("---\n")
        md.append("### \U0001f504 Implement with Claude Code\n")
        md.append("Reply to this comment with instructions:\n")
        md.append("- `Accept all` - implement everything as suggested")
        md.append("- `Ignore #2, fix the rest` - selective implementation")
        md.append("- Or any natural language instructions\n")
        md.append(f"<!-- claude-code-prompt:{pr_number} -->")
    else:
        md.append("\nNo issues found by either model. The changes look good!\n")

    return "\n".join(md)


def build_instructions(decision, all_issues):
    lines = [
        "# \u26a0\ufe0f REVIEW INSTRUCTIONS",
        "",
        "> Generated by Combined AI Review (Gemini + Codex)",
        "",
        "```json",
        json.dumps({"decision": decision, "issues": all_issues}, indent=2),
        "```",
    ]
    return "\n".join(lines)


def read_json_input(file_env, fallback_env):
    """Read JSON from a file (preferred) or env var (fallback).

    Using files avoids env var size limits on large PRs.
    """
    file_path = os.environ.get(file_env, "")
    if file_path:
        try:
            with open(file_path) as f:
                content = f.read().strip()
                if content:
                    return content
        except OSError:
            pass
    return os.environ.get(fallback_env, "{}")


def main():
    gemini_raw = read_json_input("GEMINI_JSON_FILE", "GEMINI_JSON")
    codex_raw = read_json_input("CODEX_JSON_FILE", "CODEX_JSON")
    gemini_ok = os.environ.get("GEMINI_HAS_RESULT", "false")
    codex_ok = os.environ.get("CODEX_HAS_RESULT", "false")
    pr_number = os.environ.get("PR_NUMBER", "0")

    gemini = load_review(gemini_raw, gemini_ok)
    codex = load_review(codex_raw, codex_ok)

    all_issues = collect_issues(gemini, codex)
    consensus = detect_consensus(all_issues)
    decision = determine_decision(gemini, codex)

    # Re-number issues sequentially
    for idx, i in enumerate(all_issues, 1):
        i["id"] = idx

    # Write comment markdown
    comment = build_markdown(gemini, codex, all_issues, consensus, decision, pr_number)
    with open("/tmp/combined-comment.md", "w") as f:
        f.write(comment)

    # Write REVIEW_INSTRUCTIONS.md if issues found
    has_feedback = len(all_issues) > 0
    if has_feedback:
        instructions = build_instructions(decision, all_issues)
        with open("/tmp/REVIEW_INSTRUCTIONS.md", "w") as f:
            f.write(instructions)

    # Write GitHub Actions outputs
    gh_output = os.environ.get("GITHUB_OUTPUT")
    if gh_output:
        with open(gh_output, "a") as f:
            f.write(f"has_feedback={str(has_feedback).lower()}\n")
            f.write(f"decision={decision}\n")

    print(
        f"Combined review: decision={decision}, issues={len(all_issues)}, consensus={len(consensus)}"
    )


if __name__ == "__main__":
    main()
