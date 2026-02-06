#!/usr/bin/env node
/**
 * UserPromptSubmit Hook - Skill Activation
 *
 * Automatically injects relevant skill context based on the user's prompt.
 * This eliminates the need to manually activate skills - they auto-trigger
 * based on keywords and patterns in what you're asking Claude to do.
 *
 * Skills are loaded from .claude/skills/<skill>/SKILL.md and provide
 * domain expertise without bloating the main context.
 */

import { readFileSync, readdirSync, existsSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Read user prompt from stdin
let input = '';
try {
  input = readFileSync(0, 'utf-8');
} catch (e) {
  // CRITICAL: Exit with non-zero code on error
  // Exit code 0 with no output would blank the user's prompt
  // Non-zero exit tells Claude to ignore hook output and preserve original prompt
  process.exit(1);
}

let promptData;
try {
  promptData = JSON.parse(input);
} catch (e) {
  // CRITICAL: Exit with non-zero code on parse error
  // This preserves the original prompt instead of blanking it
  process.exit(1);
}

// Get the original prompt (preserve case for output)
const originalPrompt = promptData.prompt || promptData.user_prompt || '';
const userPromptLower = originalPrompt.toLowerCase();

// CRITICAL: For UserPromptSubmit hooks, we MUST always output the prompt
// If we exit without output, the user's prompt gets blanked out
if (!userPromptLower) {
  // Empty prompt - just pass through (output nothing, let default behavior occur)
  process.exit(0);
}

// Skill definitions with trigger patterns
const skillTriggers = {
  'tdd': {
    patterns: [
      /\btest.?first\b/,
      /\btdd\b/,
      /\bred.?green.?refactor\b/,
      /\bwrite.*test.*before/,
      /\btest.?driven/,
      /\bfailing.*test.*first/,
    ],
    priority: 1
  },
  'security-review': {
    patterns: [
      /\bsecurity\b/,
      /\bvulnerab/,
      /\bowasp\b/,
      /\bauth(entication|orization)?\b/,
      /\bxss\b/,
      /\bsql.?injection\b/,
      /\binput.?valid(ation|ate)\b/,
      /\bsanitiz/,
      /\bcredential/,
      /\bsecret/,
      /\bencrypt/,
    ],
    priority: 1
  },
  'api-design': {
    patterns: [
      /\bapi\b/,
      /\bendpoint/,
      /\brest(ful)?\b/,
      /\bgraphql\b/,
      /\broute/,
      /\bhttp.*method/,
      /\brequest.*response/,
      /\bpayload/,
      /\bschema.*design/,
    ],
    priority: 2
  },
  'async-patterns': {
    patterns: [
      /\basync\b/,
      /\bawait\b/,
      /\bpromise/,
      /\bconcurren/,
      /\bparallel/,
      /\brace.?condition/,
      /\bevent.?loop/,
      /\bcallback/,
      /\bdeadlock/,
    ],
    priority: 2
  },
  'debugging': {
    patterns: [
      /\bdebug/,
      /\bbug\b/,
      /\berror\b/,
      /\bfix\b/,
      /\bstack.?trace/,
      /\bexception/,
      /\bissue\b/,
      /\bbroken\b/,
      /\bfailing\b/,
      /\bcrash/,
      /\bnot.?work/,
      /\bwhy.*doesn.?t/,
      /\bwhat.?s.*wrong/,
      /\binvestigat/,
      /\btroubleshoot/,
    ],
    priority: 1
  },
  'refactoring': {
    patterns: [
      /\brefactor/,
      /\bclean.?up/,
      /\bsimplif/,
      /\bimprove.*code/,
      /\breduce.*complex/,
      /\bcode.?smell/,
      /\btechnical.?debt/,
      /\bextract.*method/,
      /\bextract.*function/,
      /\brename\b/,
    ],
    priority: 2
  },
  'testing-patterns': {
    patterns: [
      /\btest/,
      /\bunit\b/,
      /\bintegration\b/,
      /\be2e\b/,
      /\bend.?to.?end/,
      /\bmock/,
      /\bstub\b/,
      /\bfixture/,
      /\bcoverage/,
      /\bassert/,
      /\bjest\b/,
      /\bpytest\b/,
      /\bmocha\b/,
      /\bplaywright\b/,
      /\bcypress\b/,
    ],
    priority: 2
  },
  'k8s-operations': {
    patterns: [
      /\bkubernetes\b/,
      /\bk8s\b/,
      /\bkubectl\b/,
      /\bhelm\b/,
      /\bpod\b/,
      /\bdeployment\b/,
      /\bservice\b.*\b(mesh|type)/,
      /\bingress\b/,
      /\bconfigmap\b/,
      /\bnamespace\b/,
      /\bargo\b/,
      /\bflux\b/,
    ],
    priority: 2
  },
  'cicd-automation': {
    patterns: [
      /\bci\/?cd\b/,
      /\bpipeline/,
      /\bgithub.?action/,
      /\bworkflow/,
      /\bdeploy/,
      /\bbuild.*automat/,
      /\bjenkins\b/,
      /\bcircle.?ci\b/,
      /\btravis\b/,
      /\bgitlab.?ci\b/,
    ],
    priority: 2
  },
  'observability': {
    patterns: [
      /\blog(ging|s)?\b/,
      /\bmetric/,
      /\btrac(ing|e)\b/,
      /\bmonitor/,
      /\balert/,
      /\bdashboard/,
      /\bgrafana\b/,
      /\bprometheus\b/,
      /\bdatadog\b/,
      /\bopentelemetry\b/,
      /\bspan\b/,
    ],
    priority: 2
  },
  'deslop': {
    patterns: [
      /\bdeslop\b/,
      /\bremove.?slop\b/,
      /\bai.?slop\b/,
      /\baggressive.?refactor/,
      /\bsimplify.?drastic/,
      /\bcut.?code\b/,
      /\breduce.?complex/,
      /\bover.?engineer/,
      /\btoo.?verbose\b/,
      /\byagni\b/,
      /\bdelete.*dead.?code/,
      /\binline.*function/,
    ],
    priority: 1
  },
  'systematic-debugging': {
    patterns: [
      /\bsystematic.?debug/,
      /\bmethodical.?debug/,
      /\bstructured.?debug/,
      /\broot.?cause.?analysis\b/,
      /\brca\b/,
      /\b5.?whys?\b/,
      /\bbisect\b/,
      /\bgit.?bisect\b/,
      /\bhypothesis/,
      /\breproduc.*bug/,
      /\bintermittent/,
    ],
    priority: 1
  },
  'ralph-coder': {
    patterns: [
      /\bcoder.?loop\b/,
      /\bralph.?mode\b/,
      /\bralph.?coder\b/,
      /\bautonomous.?cod/,
      /\bcode.?until.?done/,
      /\btdd.?loop\b/,
      /\bimplement.?and.?commit/,
      /\batomic.?commit/,
      /\bquality.?gate/,
    ],
    priority: 1
  },
  'autonomous-loop': {
    patterns: [
      /\bautonomous\b/,
      /\bkeep.?going/,
      /\bloop.?until/,
      /\biterat.*until/,
      /\bexit.?signal/,
      /\bcircuit.?breaker/,
    ],
    priority: 1
  },
  'surgical-analysis': {
    patterns: [
      /\bzeno\b/,
      /\bsurgical.?analysis/,
      /\bevidence.?based.?review/,
      /\bcite.?lines/,
      /\bfile.?line.?citation/,
      /\bcode.?audit/,
      /\bfind.?vulnerabilities/,
      /\bscan.?for.?issues/,
      /\bstatic.?analysis/,
    ],
    priority: 1
  },
  'browser-automation': {
    patterns: [
      /\bbrowser\b/,
      /\bpuppeteer\b/,
      /\bplaywright\b/,
      /\be2e.?test/,
      /\bend.?to.?end/,
      /\bscreenshot/,
      /\bvisual.?test/,
      /\bheadless/,
      /\bcdp\b/,
      /\bclick.*button/,
      /\bfill.*form/,
      /\bscrape/,
      /\bautomation/,
    ],
    priority: 2
  },
  'chatops': {
    patterns: [
      /\bchatops\b/,
      /\bslack.?bot\b/,
      /\bdiscord.?bot\b/,
      /\btelegram.?bot\b/,
      /\bremote.?command/,
      /\bchat.?trigger/,
      /\bwebhook/,
      /\bgateway\b/,
      /\bbidirectional/,
      /\bchat.?platform/,
      /\bslash.?command/,
      /\bbot.?integration/,
    ],
    priority: 2
  },
  'workflow-orchestration': {
    patterns: [
      /\bworkflow\b/,
      /\bpipeline\b/,
      /\borchestrat/,
      /\bapproval.?gate/,
      /\bmulti.?step/,
      /\blobster\b/,
      /\btyped.?workflow/,
      /\bpause.*approval/,
      /\bwait.*review/,
      /\bstep.?by.?step.?execut/,
    ],
    priority: 2
  }
};

// Find matching skills
const matchedSkills = [];

for (const [skillName, config] of Object.entries(skillTriggers)) {
  for (const pattern of config.patterns) {
    if (pattern.test(userPromptLower)) {
      matchedSkills.push({
        name: skillName,
        priority: config.priority
      });
      break; // Only match each skill once
    }
  }
}

// Sort by priority (lower = higher priority)
matchedSkills.sort((a, b) => a.priority - b.priority);

// Limit to top 2 skills to avoid context bloat
const activeSkills = matchedSkills.slice(0, 2);

if (activeSkills.length === 0) {
  // No skills matched - pass through original prompt unchanged
  // CRITICAL: Must output prompt or it gets blanked
  console.log(originalPrompt);
  process.exit(0);
}

// Load skill content
const skillsDir = join(__dirname, '..', '..', 'skills');
let skillContext = '';

for (const skill of activeSkills) {
  const skillPath = join(skillsDir, skill.name, 'SKILL.md');

  if (existsSync(skillPath)) {
    try {
      const content = readFileSync(skillPath, 'utf-8');
      // Remove frontmatter
      const cleanContent = content.replace(/^---[\s\S]*?---\n*/, '');
      skillContext += `\n## Activated Skill: ${skill.name}\n`;
      skillContext += cleanContent + '\n';
    } catch (e) {
      // Silently skip unreadable skills
    }
  }
}

// CRITICAL: Output the original prompt first, then append skill context
// UserPromptSubmit hooks replace the user's input, so we must preserve it
if (skillContext) {
  // Output original prompt first (this is what the user actually asked)
  console.log(originalPrompt);
  console.log('');
  console.log('<skill-context>');
  console.log('The following skill context was auto-activated based on your prompt:');
  console.log(skillContext);
  console.log('</skill-context>');
} else {
  // No skill content loaded - pass through original prompt
  // CRITICAL: Must output prompt or it gets blanked
  console.log(originalPrompt);
  process.exit(0);
}
