# Browser Automation Skill

> Patterns and best practices for headless browser automation, E2E testing, and visual validation.

## When to Use

- E2E testing of web applications
- Visual regression testing
- Web scraping (with permission)
- Automated form submission
- Screenshot capture
- PDF generation
- Accessibility auditing

## Tool Selection

| Tool           | Best For                          | Installation                |
| -------------- | --------------------------------- | --------------------------- |
| **Playwright** | Cross-browser testing, modern API | `npm i -D @playwright/test` |
| **Puppeteer**  | Chrome-specific, CDP access       | `npm i puppeteer`           |
| **Cypress**    | Component testing, time-travel    | `npm i -D cypress`          |

## Quick Commands

### Playwright (Recommended)

```bash
# Install browsers
npx playwright install chromium

# Take screenshot
npx playwright screenshot https://example.com screenshot.png

# Open browser interactively
npx playwright open https://example.com

# Run tests
npx playwright test

# Show report
npx playwright show-report
```

### Puppeteer

```bash
# Quick screenshot
npx puppeteer screenshot https://example.com screenshot.png

# PDF generation
npx puppeteer pdf https://example.com output.pdf
```

## Selector Strategies

### Priority Order (Most Stable → Least Stable)

1. **Test IDs** (most stable)

   ```javascript
   page.locator('[data-testid="submit-button"]');
   ```

2. **ARIA Labels**

   ```javascript
   page.getByRole("button", { name: "Submit" });
   ```

3. **Text Content**

   ```javascript
   page.getByText("Submit Form");
   ```

4. **CSS Selectors** (use sparingly)

   ```javascript
   page.locator(".submit-btn");
   ```

5. **XPath** (avoid if possible)
   ```javascript
   page.locator('//button[contains(text(), "Submit")]');
   ```

### Selector Best Practices

```javascript
// GOOD: Specific, stable selectors
await page.click('[data-testid="login-button"]');
await page.fill('[data-testid="email-input"]', email);

// BAD: Brittle selectors
await page.click(".btn.btn-primary.mt-4"); // CSS classes change
await page.click("div > div > button"); // Structure changes
```

## Wait Strategies

### Automatic Waits (Playwright)

```javascript
// Playwright auto-waits for elements
await page.click("button"); // Waits for button to be visible & enabled
```

### Explicit Waits

```javascript
// Wait for element
await page.waitForSelector(".loaded");

// Wait for URL
await page.waitForURL("**/dashboard");

// Wait for network idle
await page.waitForLoadState("networkidle");

// Wait for condition
await page.waitForFunction(() => {
  return document.querySelector(".count").textContent === "10";
});
```

### Timeout Handling

```javascript
// Set default timeout
page.setDefaultTimeout(30000);

// Per-action timeout
await page.click("button", { timeout: 5000 });

// Handle timeout gracefully
try {
  await page.waitForSelector(".optional", { timeout: 3000 });
} catch {
  console.log("Optional element not found");
}
```

## Page Object Model

Encapsulate page interactions for maintainability:

```typescript
// pages/LoginPage.ts
export class LoginPage {
  readonly page: Page;
  readonly emailInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly errorMessage: Locator;

  constructor(page: Page) {
    this.page = page;
    this.emailInput = page.locator('[data-testid="email"]');
    this.passwordInput = page.locator('[data-testid="password"]');
    this.submitButton = page.locator('[data-testid="submit"]');
    this.errorMessage = page.locator('[data-testid="error"]');
  }

  async goto() {
    await this.page.goto("/login");
  }

  async login(email: string, password: string) {
    await this.emailInput.fill(email);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }

  async getError(): Promise<string | null> {
    if (await this.errorMessage.isVisible()) {
      return this.errorMessage.textContent();
    }
    return null;
  }
}
```

## Visual Testing

### Screenshot Comparison

```javascript
// Capture baseline
await page.screenshot({ path: "baseline.png", fullPage: true });

// Compare with baseline
const screenshot = await page.screenshot({ fullPage: true });
expect(screenshot).toMatchSnapshot("page.png", {
  threshold: 0.1, // 10% difference allowed
});
```

### Masking Dynamic Content

```javascript
await page.screenshot({
  path: "screenshot.png",
  mask: [page.locator(".timestamp"), page.locator(".random-ad")],
});
```

### Visual Diff Tools

```bash
# Using pixelmatch
npx pixelmatch baseline.png current.png diff.png 0.1

# Using reg-cli
npx reg-cli actual/ expected/ diff/ -R report.html
```

## Network Control

### Intercept & Mock

```javascript
// Mock API response
await page.route("**/api/users", (route) => {
  route.fulfill({
    status: 200,
    body: JSON.stringify({ users: [] }),
  });
});

// Modify request
await page.route("**/api/*", (route) => {
  route.continue({
    headers: {
      ...route.request().headers(),
      "X-Test-Header": "true",
    },
  });
});
```

### Block Resources

```javascript
// Speed up tests by blocking non-essential resources
await page.route("**/*.{png,jpg,gif,svg}", (route) => route.abort());
await page.route("**/analytics/**", (route) => route.abort());
await page.route("**/ads/**", (route) => route.abort());
```

### Capture Network

```javascript
// Log all requests
page.on("request", (request) => {
  console.log(">>", request.method(), request.url());
});

page.on("response", (response) => {
  console.log("<<", response.status(), response.url());
});
```

## Accessibility Testing

```javascript
import { AxeBuilder } from "@axe-core/playwright";

test("should not have accessibility violations", async ({ page }) => {
  await page.goto("/");

  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa"])
    .analyze();

  expect(results.violations).toEqual([]);
});
```

## Mobile & Responsive

```javascript
// Emulate mobile device
const iPhone = playwright.devices["iPhone 13"];
const context = await browser.newContext({
  ...iPhone,
});

// Custom viewport
await page.setViewportSize({ width: 375, height: 667 });
```

## Authentication Patterns

### Reuse Auth State

```javascript
// Save auth state after login
await page.context().storageState({ path: "auth.json" });

// Reuse in other tests
const context = await browser.newContext({
  storageState: "auth.json",
});
```

### API Login (Faster)

```javascript
// Login via API instead of UI
const response = await page.request.post("/api/login", {
  data: { email, password },
});
const { token } = await response.json();

// Set token in context
await page.evaluate((token) => {
  localStorage.setItem("authToken", token);
}, token);
```

## Debugging

### Trace Recording

```bash
# Record trace
npx playwright test --trace on

# View trace
npx playwright show-trace trace.zip
```

### Screenshots on Failure

```javascript
test.afterEach(async ({ page }, testInfo) => {
  if (testInfo.status !== "passed") {
    await page.screenshot({
      path: `screenshots/${testInfo.title}.png`,
      fullPage: true,
    });
  }
});
```

### Headed Mode

```javascript
// Run with visible browser
const browser = await chromium.launch({ headless: false });

// Slow down actions
const browser = await chromium.launch({
  headless: false,
  slowMo: 100,
});
```

## Performance Tips

1. **Reuse browser context** across tests when possible
2. **Block unnecessary resources** (images, fonts, analytics)
3. **Use API for setup** instead of UI clicks
4. **Parallelize tests** with test sharding
5. **Cache static assets** with `page.route()`

## Common Pitfalls

| Pitfall          | Solution                               |
| ---------------- | -------------------------------------- |
| Flaky waits      | Use auto-waiting or explicit waitFor   |
| Selector changes | Use data-testid attributes             |
| Race conditions  | Wait for specific conditions, not time |
| Memory leaks     | Close browsers and contexts            |
| Slow tests       | Mock APIs, block resources             |

## Activation Triggers

This skill auto-activates when prompts contain:

- "browser", "puppeteer", "playwright"
- "e2e test", "end-to-end"
- "screenshot", "visual test"
- "headless", "CDP"
- "click", "fill form"
- "scrape", "automation"

## Integration

Works with:

- **@browser-automator**: Agent for browser tasks
- **@test-automator**: E2E test creation
- **/browser**: Quick browser commands
- **/screenshot**: Page capture
- **/visual-diff**: Screenshot comparison
