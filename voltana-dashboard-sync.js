#!/usr/bin/env node
'use strict';

const fs   = require('fs');
const path = require('path');

const ROOT          = __dirname;
const WORKFLOWS_DIR = path.join(ROOT, '.ai', 'workflows');
const DASHBOARD     = path.join(ROOT, 'voltana-dashboard.html');

// ── 1. Parse every TASK-*.md file, handling multi-task files ─────────────────

const taskStatuses = {};   // { 'TASK-0001': 'DONE', 'TASK-0002': 'READY', … }

const mdFiles = fs.readdirSync(WORKFLOWS_DIR)
  .filter(f => /^TASK-.*\.md$/i.test(f))
  .sort();

for (const file of mdFiles) {
  const content = fs.readFileSync(path.join(WORKFLOWS_DIR, file), 'utf8');

  // A single file may contain multiple task sections (e.g. TASK-0003-0008.md).
  // Each section starts with "# TASK-XXXX" and has a "**Status**: VALUE" line.
  const sectionRe = /^# (TASK-\d+)[^\n]*/gm;
  let sectionMatch;

  while ((sectionMatch = sectionRe.exec(content)) !== null) {
    const taskId    = sectionMatch[1];
    const fromHere  = content.slice(sectionMatch.index);

    // Find the FIRST **Status**: line within this section (≤600 chars ahead)
    const statusMatch = fromHere.match(/\*\*Status\*\*:\s*([A-Z_]+)/);
    if (statusMatch) {
      taskStatuses[taskId] = statusMatch[1].trim();
    }
  }
}

if (Object.keys(taskStatuses).length === 0) {
  console.error('No tasks found in', WORKFLOWS_DIR);
  process.exit(1);
}

// ── 2. Read dashboard HTML ────────────────────────────────────────────────────

let html = fs.readFileSync(DASHBOARD, 'utf8');

// ── 3. Update status field for each task in the TASKS JS array ───────────────
// Pattern: id: 'TASK-XXXX'  …up to 300 chars…  status: 'OLD_STATUS'

let updatedCount = 0;
for (const [taskId, status] of Object.entries(taskStatuses)) {
  const re = new RegExp(
    `(id:\\s*'${taskId}'[\\s\\S]{0,300}?status:\\s*')([A-Z_]+)(')`
  );
  html = html.replace(re, (_, pre, old, post) => {
    if (old !== status) updatedCount++;
    return `${pre}${status}${post}`;
  });
}

// ── 4. Tally counts ───────────────────────────────────────────────────────────

const counts = {
  DONE:        0,
  READY:       0,
  IN_PROGRESS: 0,
  REVIEW:      0,
  TESTING:     0,
  BLOCKED:     0,
  BACKLOG:     0,
};

for (const status of Object.values(taskStatuses)) {
  if (status in counts) counts[status]++;
  else                   counts.BACKLOG++;
}

// TESTING counts alongside REVIEW in the dashboard (same visual column)
const displayReview = counts.REVIEW + counts.TESTING;

// ── 5. Update hero stat cards ─────────────────────────────────────────────────
// <div class="stat-label">Label</div>
// <div class="stat-value">N</div>

const statUpdates = {
  'Done':        counts.DONE,
  'Ready':       counts.READY,
  'In Progress': counts.IN_PROGRESS,
  'Review':      displayReview,
  'Blocked':     counts.BLOCKED,
  'Backlog':     counts.BACKLOG,
};

for (const [label, count] of Object.entries(statUpdates)) {
  html = html.replace(
    new RegExp(
      `(<div class="stat-label">${label}<\\/div>\\s*<div class="stat-value">)\\d+(<\\/div>)`,
      'g'
    ),
    `$1${count}$2`
  );
}

// ── 6. Update kanban column header counts ─────────────────────────────────────
// <span class="col-count s-KEY">N</span>

const colUpdates = {
  backlog: counts.BACKLOG,
  ready:   counts.READY,
  inprog:  counts.IN_PROGRESS,
  review:  displayReview,
  done:    counts.DONE,
};

for (const [key, count] of Object.entries(colUpdates)) {
  html = html.replace(
    new RegExp(`(<span class="col-count s-${key}">)\\d+(<\\/span>)`, 'g'),
    `$1${count}$2`
  );
}

// ── 7. Update ACTIVITY feed from git log ─────────────────────────────────────

try {
  const { execSync } = require('child_process');
  const gitOut = execSync("git log --format='%h|%s|%cr' -14", { cwd: ROOT, encoding: 'utf8' }).trim();
  const lines  = gitOut.split('\n').filter(Boolean);

  const iconMap = [
    [/^feat:/,      '✅', 'rgba(16,185,129,0.2)'],
    [/^fix:/,       '🐛', 'rgba(245,158,11,0.2)'],
    [/^chore:/,     '🔧', 'rgba(100,116,139,0.2)'],
    [/^docs:/,      '📖', 'rgba(99,102,241,0.2)'],
    [/^refactor:/,  '♻️', 'rgba(14,165,233,0.2)'],
    [/^test:/,      '🧪', 'rgba(168,85,247,0.2)'],
  ];

  const entries = lines.map(line => {
    const parts   = line.split('|');
    const hash    = parts[0];
    const subject = parts.slice(1, -1).join('|');
    const relTime = parts[parts.length - 1];
    if (!hash || !subject) return null;

    let icon = '📝', color = 'rgba(99,102,241,0.2)';
    for (const [re, i, c] of iconMap) {
      if (re.test(subject)) { icon = i; color = c; break; }
    }

    const safe = subject.replace(/\\/g, '\\\\').replace(/'/g, "\\'")
                        .replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return `  { icon: '${icon}', color: '${color}', text: '<strong>${hash}</strong> — ${safe}', time: '${relTime.trim()}' }`;
  }).filter(Boolean);

  const newActivity = `const ACTIVITY = [\n${entries.join(',\n')},\n];`;
  html = html.replace(/const ACTIVITY = \[[\s\S]*?\];/, newActivity);
} catch (_) {
  // git not available — skip activity update
}

// ── 8. Write dashboard back ───────────────────────────────────────────────────

fs.writeFileSync(DASHBOARD, html, 'utf8');

// ── 9. Report ─────────────────────────────────────────────────────────────────

const total = Object.keys(taskStatuses).length;
const summary = Object.entries(counts)
  .filter(([, v]) => v > 0)
  .map(([k, v]) => `${k}:${v}`)
  .join('  ');

console.log(`voltana-dashboard synced — ${total} tasks  |  ${summary}`);
if (updatedCount > 0) {
  console.log(`  ${updatedCount} status change(s) written to dashboard`);
}
Object.entries(taskStatuses)
  .sort(([a], [b]) => a.localeCompare(b))
  .forEach(([id, s]) => console.log(`  ${id}  →  ${s}`));
