# Contributing to Claude Code Professional Engineering Team

Thank you for your interest in contributing!

## ⚠️ MANDATORY: Read the Source Repository First

**Before contributing, ALL contributors (human or AI) MUST read:**

👉 **https://github.com/bigdegenenergy/ai-dev-toolkit**

This is **NON-NEGOTIABLE**. You must understand:

- The virtual team architecture
- Existing conventions and workflows
- The feedback loop principle
- Quality gates and verification strategies

---

## For AI Agents

If you are an AI agent contributing to this repository:

1. **Read First**: Use WebFetch to read https://github.com/bigdegenenergy/ai-dev-toolkit in full
2. **Plan**: Use `/plan` before making any changes
3. **Implement**: Follow existing patterns and conventions
4. **Simplify**: Use `/simplify` to clean up your code
5. **Verify**: Use `/qa` to ensure tests pass
6. **Review**: Request `@code-reviewer` feedback
7. **Ship**: Use `/ship` to create your PR

### Required Checklist

Before creating a PR, ensure:

- [ ] You have read https://github.com/bigdegenenergy/ai-dev-toolkit
- [ ] Your changes follow existing patterns
- [ ] Tests pass (`/qa`)
- [ ] Code has been simplified (`/simplify`)
- [ ] Code has been reviewed (`@code-reviewer`)

---

## For Human Contributors

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting
5. Commit with conventional commit messages (`feat:`, `fix:`, `docs:`, etc.)
6. Push to your branch
7. Open a Pull Request

### Code Style

- **Markdown**: Follow existing formatting
- **Shell Scripts**: Use ShellCheck-compliant bash
- **Python**: Use Black for formatting, follow PEP 8
- **TOML/JSON**: Validate before committing

### Commit Messages

Use conventional commits:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `refactor:` Code refactoring
- `test:` Test additions/updates
- `chore:` Maintenance tasks

---

## Pull Request Process

1. Ensure your PR description explains the changes
2. Check the "I have read the source repository" checkbox
3. Wait for CI checks to pass
4. Request review from maintainers
5. Address any feedback
6. Merge when approved

## Questions?

- Open an issue for bugs or feature requests
- Check existing documentation first
- Reference specific files when asking questions

---

**Remember:** The goal is to amplify human capabilities. Keep contributions focused and well-tested.
