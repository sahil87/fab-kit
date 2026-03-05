#!/usr/bin/env node
// Node benchmark contender for statusman operations.
// Uses js-yaml for YAML parsing/serialization.

import { readFileSync, writeFileSync, renameSync } from 'fs';
import { dirname, join } from 'path';
import yaml from 'js-yaml';

const STAGES = ['intake', 'spec', 'tasks', 'apply', 'review', 'hydrate', 'ship', 'review-pr'];

function readStatus(filePath) {
  return yaml.load(readFileSync(filePath, 'utf8'));
}

function writeStatusAtomic(filePath, data) {
  const dir = dirname(filePath);
  const tmp = join(dir, `.status.yaml.${process.pid}.${Date.now()}`);
  writeFileSync(tmp, yaml.dump(data, { lineWidth: -1, noRefs: true }));
  renameSync(tmp, filePath);
}

function now() {
  return new Date().toISOString();
}

// ─── progress-map ─────────────────────────────────────────────────────────────
function progressMap(statusFile) {
  const data = readStatus(statusFile);
  const progress = data.progress || {};
  for (const stage of STAGES) {
    console.log(`${stage}:${progress[stage] || 'pending'}`);
  }
}

// ─── set-change-type ──────────────────────────────────────────────────────────
function setChangeType(statusFile, type) {
  const validTypes = ['feat', 'fix', 'refactor', 'docs', 'test', 'ci', 'chore'];
  if (!validTypes.includes(type)) {
    process.stderr.write(`ERROR: Invalid change type '${type}'\n`);
    process.exit(1);
  }
  const data = readStatus(statusFile);
  data.change_type = type;
  data.last_updated = now();
  writeStatusAtomic(statusFile, data);
}

// ─── finish ───────────────────────────────────────────────────────────────────
function finish(statusFile, stage) {
  const data = readStatus(statusFile);
  const progress = data.progress || {};
  const currentState = progress[stage] || 'pending';

  if (currentState !== 'active' && currentState !== 'ready') {
    process.stderr.write(`ERROR: Cannot finish stage '${stage}' — current state is '${currentState}'\n`);
    process.exit(1);
  }

  const ts = now();

  // Set stage to done
  progress[stage] = 'done';

  // Stage metrics — completed_at
  if (!data.stage_metrics) data.stage_metrics = {};
  if (!data.stage_metrics[stage]) data.stage_metrics[stage] = {};
  data.stage_metrics[stage].completed_at = ts;

  // Auto-activate next pending stage
  const stageIdx = STAGES.indexOf(stage);
  if (stageIdx >= 0 && stageIdx < STAGES.length - 1) {
    const nextStage = STAGES[stageIdx + 1];
    if ((progress[nextStage] || 'pending') === 'pending') {
      progress[nextStage] = 'active';
      data.stage_metrics[nextStage] = {
        started_at: ts,
        driver: 'benchmark',
        iterations: 1
      };
    }
  }

  data.progress = progress;
  data.last_updated = ts;
  writeStatusAtomic(statusFile, data);
}

// ─── CLI ──────────────────────────────────────────────────────────────────────
const [,, cmd, ...args] = process.argv;

switch (cmd) {
  case '--help':
  case '-h':
    console.log('statusman-node: Node benchmark contender');
    console.log('Usage: node statusman.mjs {progress-map|set-change-type|finish} <status_file> [args...]');
    break;
  case 'progress-map':
    progressMap(args[0]);
    break;
  case 'set-change-type':
    setChangeType(args[0], args[1]);
    break;
  case 'finish':
    finish(args[0], args[1]);
    break;
  default:
    process.stderr.write(`Unknown command: ${cmd || ''}\n`);
    process.exit(1);
}
