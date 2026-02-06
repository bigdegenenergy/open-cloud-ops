# Screenshot: Capture Web Page

Capture a screenshot of a web page or element.

## Target

$ARGUMENTS

Format: `URL [options]`

Examples:

- `https://example.com` - Full page screenshot
- `https://example.com --selector=".main"` - Element screenshot
- `https://example.com --mobile` - Mobile viewport
- `https://example.com --name=homepage` - Named output

## Execution

### Full Page Screenshot

```bash
# Create output directory
mkdir -p .claude/artifacts/browser/screenshots

# Capture screenshot
npx playwright screenshot "$URL" .claude/artifacts/browser/screenshots/capture.png --full-page
```

### Element Screenshot

```bash
npx playwright screenshot "$URL" .claude/artifacts/browser/screenshots/element.png --selector="$SELECTOR"
```

### Mobile Viewport

```bash
# iPhone 13 viewport
npx playwright screenshot "$URL" .claude/artifacts/browser/screenshots/mobile.png \
  --viewport-size=390,844 \
  --device-scale-factor=3
```

### Multiple Viewports

```bash
# Desktop
npx playwright screenshot "$URL" desktop.png --viewport-size=1920,1080

# Tablet
npx playwright screenshot "$URL" tablet.png --viewport-size=768,1024

# Mobile
npx playwright screenshot "$URL" mobile.png --viewport-size=375,667
```

## Advanced Options

### Wait for Element

```javascript
// Script for waiting before screenshot
const { chromium } = require("playwright");

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto("$URL");

  // Wait for specific element
  await page.waitForSelector(".content-loaded");

  // Or wait for network idle
  await page.waitForLoadState("networkidle");

  await page.screenshot({
    path: ".claude/artifacts/browser/screenshots/capture.png",
    fullPage: true,
  });

  await browser.close();
})();
```

### Mask Dynamic Content

```javascript
await page.screenshot({
  path: "screenshot.png",
  fullPage: true,
  mask: [
    page.locator(".timestamp"),
    page.locator(".random-content"),
    page.locator(".ad-banner"),
  ],
});
```

### Clip Region

```javascript
await page.screenshot({
  path: "clipped.png",
  clip: {
    x: 0,
    y: 0,
    width: 800,
    height: 600,
  },
});
```

## Output

Screenshots saved to: `.claude/artifacts/browser/screenshots/`

```
.claude/artifacts/browser/screenshots/
├── capture.png           # Default name
├── homepage.png          # Named screenshot
├── mobile.png            # Mobile viewport
└── element.png           # Element screenshot
```

## Report Format

```markdown
## Screenshot Captured

**URL:** [URL]
**Viewport:** [width x height]
**Type:** [full-page / element / clipped]
**Output:** `.claude/artifacts/browser/screenshots/[filename].png`

### Preview

[Description of what was captured]

### Notes

[Any issues or observations]
```

---

**Capture screenshot of:** $ARGUMENTS
