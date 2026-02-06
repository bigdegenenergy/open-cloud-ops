# Browser: Headless Browser Automation

Control a headless browser for testing, scraping, or automation tasks.

## Target

$ARGUMENTS

Examples:

- `https://example.com` - Navigate and interact
- `login https://app.com` - Perform login flow
- `test https://app.com/dashboard` - Run E2E test
- `scrape https://example.com .article` - Extract content

## Prerequisites

Ensure Playwright or Puppeteer is available:

```bash
# Check if Playwright is installed
npx playwright --version 2>/dev/null || echo "Playwright not installed"

# Install if needed
npm install -D @playwright/test
npx playwright install chromium
```

## Operations

### Navigation

```bash
# Open URL and take screenshot
npx playwright screenshot "$URL" screenshot.png

# Open in headed mode for debugging
npx playwright open "$URL"
```

### Screenshot

```bash
# Full page
npx playwright screenshot "$URL" .claude/artifacts/browser/screenshots/page.png --full-page

# Specific element
npx playwright screenshot "$URL" element.png --selector="$SELECTOR"
```

### PDF Generation

```bash
npx playwright pdf "$URL" .claude/artifacts/browser/pdfs/output.pdf
```

### Form Interaction (via script)

```javascript
// Save as temp script and run
const { chromium } = require("playwright");

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();

  await page.goto("$URL");
  await page.fill('[data-testid="email"]', process.env.TEST_EMAIL || "test@example.com");
  await page.fill('[data-testid="password"]', process.env.TEST_PASSWORD);
  await page.click('[data-testid="submit"]');
  await page.waitForURL("**/dashboard");

  console.log("Login successful:", page.url());
  await browser.close();
})();
```

## E2E Test Pattern

For testing a flow:

```javascript
const { test, expect } = require("@playwright/test");

test("user can complete checkout", async ({ page }) => {
  // Navigate
  await page.goto("https://shop.example.com");

  // Add item to cart
  await page.click('[data-testid="add-to-cart"]');
  await page.click('[data-testid="go-to-cart"]');

  // Checkout
  await page.click('[data-testid="checkout"]');
  await page.fill('[data-testid="email"]', process.env.TEST_EMAIL || "test@example.com");
  await page.click('[data-testid="place-order"]');

  // Verify
  await expect(page.locator(".order-confirmation")).toBeVisible();
});
```

## Output

Save all artifacts to `.claude/artifacts/browser/`:

```bash
mkdir -p .claude/artifacts/browser/{screenshots,pdfs,traces}
```

## Report Format

After browser operations, report:

```markdown
## Browser Session Report

**URL:** [target URL]
**Operation:** [navigation/test/scrape]
**Status:** [success/failure]

### Actions Performed

1. [Action 1]
2. [Action 2]
3. [Action 3]

### Screenshots

- `.claude/artifacts/browser/screenshots/step-1.png`
- `.claude/artifacts/browser/screenshots/step-2.png`

### Results

[What was found/tested/scraped]

### Errors (if any)

[Error messages with context]
```

---

**Execute browser task:** $ARGUMENTS
