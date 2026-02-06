# Visual Diff: Screenshot Comparison

Compare screenshots to detect visual regressions.

## Target

$ARGUMENTS

Format: `baseline.png current.png` or `URL --baseline=baseline.png`

Examples:

- `baseline.png current.png` - Compare two images
- `https://example.com --baseline=expected.png` - Capture and compare
- `--update-baseline https://example.com` - Update baseline image

## Comparison Methods

### Method 1: Using pixelmatch (Node.js)

```bash
# Install pixelmatch
npm install pixelmatch pngjs

# Compare images
node -e "
const fs = require('fs');
const PNG = require('pngjs').PNG;
const pixelmatch = require('pixelmatch');

const img1 = PNG.sync.read(fs.readFileSync('baseline.png'));
const img2 = PNG.sync.read(fs.readFileSync('current.png'));
const {width, height} = img1;
const diff = new PNG({width, height});

const numDiffPixels = pixelmatch(img1.data, img2.data, diff.data, width, height, {
  threshold: 0.1
});

fs.writeFileSync('.claude/artifacts/browser/diff.png', PNG.sync.write(diff));
console.log('Different pixels:', numDiffPixels);
console.log('Difference %:', (numDiffPixels / (width * height) * 100).toFixed(2));
"
```

### Method 2: Using Playwright

```javascript
const { test, expect } = require("@playwright/test");

test("visual regression", async ({ page }) => {
  await page.goto("https://example.com");

  // Compare with baseline
  await expect(page).toHaveScreenshot("baseline.png", {
    maxDiffPixels: 100, // Allow up to 100 different pixels
    // OR
    maxDiffPixelRatio: 0.01, // Allow 1% difference
  });
});
```

### Method 3: Using reg-cli

```bash
# Install reg-cli
npm install -g reg-cli

# Compare directories
reg-cli ./actual ./expected ./diff -R report.html

# View report
open report.html
```

## Workflow

### 1. Capture Baseline

```bash
# First run: capture baseline
npx playwright screenshot https://example.com .claude/artifacts/browser/baselines/homepage.png --full-page
```

### 2. Capture Current

```bash
# After changes: capture current state
npx playwright screenshot https://example.com .claude/artifacts/browser/current/homepage.png --full-page
```

### 3. Compare

```bash
# Generate diff
node compare.js baseline.png current.png diff.png
```

### 4. Review & Update

```bash
# If changes are intentional, update baseline
cp current/homepage.png baselines/homepage.png
```

## Threshold Settings

| Threshold | Use Case                                |
| --------- | --------------------------------------- |
| `0.0`     | Pixel-perfect match required            |
| `0.1`     | Minor anti-aliasing differences allowed |
| `0.2`     | Small color variations allowed          |
| `0.5`     | Significant changes highlighted only    |

## Output Structure

```
.claude/artifacts/browser/
├── baselines/          # Expected screenshots
│   ├── homepage.png
│   └── dashboard.png
├── current/            # Current screenshots
│   ├── homepage.png
│   └── dashboard.png
├── diffs/              # Difference images
│   ├── homepage-diff.png
│   └── dashboard-diff.png
└── reports/
    └── visual-regression.html
```

## Report Format

```markdown
## Visual Diff Report

**Baseline:** [baseline path]
**Current:** [current path]
**Threshold:** [0.1]

### Results

| Page      | Status  | Diff Pixels | Diff % |
| --------- | ------- | ----------- | ------ |
| Homepage  | ✅ PASS | 42          | 0.01%  |
| Dashboard | ❌ FAIL | 15,234      | 2.3%   |
| Settings  | ✅ PASS | 0           | 0%     |

### Failed Comparisons

#### Dashboard

- **Diff Image:** `.claude/artifacts/browser/diffs/dashboard-diff.png`
- **Changes Detected:**
  - Header layout shifted
  - New notification badge
  - Color change in sidebar

### Recommendations

1. Review dashboard changes - intentional update?
2. If intentional: `cp current/dashboard.png baselines/dashboard.png`
3. If regression: investigate recent commits
```

## Integration with CI

```yaml
# .github/workflows/visual-regression.yml
- name: Visual Regression Test
  run: |
    npx playwright test --grep @visual
    if [ -f diff-report.html ]; then
      echo "Visual differences detected!"
      exit 1
    fi
```

## Handling Dynamic Content

### Mask Dynamic Elements

```javascript
await page.screenshot({
  path: "screenshot.png",
  mask: [
    page.locator(".timestamp"),
    page.locator(".random-id"),
    page.locator(".ad-container"),
  ],
});
```

### Wait for Stability

```javascript
// Wait for animations to complete
await page.waitForTimeout(1000);

// Or wait for specific condition
await page.waitForFunction(() => {
  return !document.querySelector(".loading-spinner");
});
```

---

**Compare visuals:** $ARGUMENTS
