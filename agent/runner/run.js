import fs from "fs";
import path from "path";
import { execSync, spawnSync } from "child_process";
import OpenAI from "openai";

/* ---------- helpers ---------- */
function sh(cmd) {
  return execSync(cmd, { stdio: "pipe", encoding: "utf-8" }).trim();
}
function must(cmd) {
  execSync(cmd, { stdio: "inherit" });
}
function readFileSafe(p) {
  try { return fs.readFileSync(p, "utf-8"); } catch { return ""; }
}
function ensureDir(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}
function runCapture(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { encoding: "utf-8", ...opts });
  return {
    code: r.status ?? 1,
    stdout: r.stdout || "",
    stderr: r.stderr || "",
  };
}
function nowId() {
  return Date.now();
}

/* ---------- args ---------- */
function parseArgs(argv) {
  const args = argv.slice(2);
  let role = "backend";
  let noBranch = false;
  let repair = 2;      // attempts after initial
  let review = true;

  const takeFlag = (name) => {
    const i = args.indexOf(name);
    if (i === -1) return false;
    args.splice(i, 1);
    return true;
  };

  const takeOpt = (name, def) => {
    const i = args.indexOf(name);
    if (i === -1) return def;
    const v = args[i + 1];
    args.splice(i, 2);
    return v ?? def;
  };

  noBranch = takeFlag("--no-branch");
  if (takeFlag("--no-review")) review = false;

  const roleIdx = args.indexOf("--role");
  if (roleIdx !== -1) {
    role = (args[roleIdx + 1] || "").trim();
    args.splice(roleIdx, 2);
  }

  const repairStr = takeOpt("--repair", String(repair));
  repair = Math.max(0, Number.parseInt(repairStr, 10) || 0);

  const task = args.join(" ").trim();
  return { role, task, noBranch, repair, review };
}

/* ---------- context narrowing ---------- */
function listRepoFiles() {
  return sh("git ls-files").split("\n").filter(Boolean);
}
function pickFilesByRole(role, files) {
  const keep = new Set();

  const addPrefix = (prefix) => {
    for (const f of files) if (f.startsWith(prefix)) keep.add(f);
  };

  // docs always relevant
  addPrefix("docs/");
  // core Makefile
  if (files.includes("Makefile")) keep.add("Makefile");
  
  if (role === "frontend") {
    addPrefix("apps/frontend/");
    addPrefix("docs/api.md");
  }
  if (role === "backend") addPrefix("apps/backend/");
  if (role === "qa") addPrefix("apps/load/");
  if (role === "architect" || role === "product") {
    // only docs + Makefile already added
  }
  if (role === "review") {
    // review uses docs + Makefile + diff (added separately)
  }

  return Array.from(keep).sort();
}

// prevent massive context: read only small/medium files; truncate huge
function readSelectedFiles(paths, maxCharsTotal = 120_000, maxCharsPerFile = 12_000) {
  let total = 0;
  const out = [];
  for (const p of paths) {
    let c = readFileSafe(p);
    if (!c) continue;

    if (c.length > maxCharsPerFile) c = c.slice(0, maxCharsPerFile) + "\n\n[TRUNCATED]\n";
    const nextTotal = total + c.length;
    if (nextTotal > maxCharsTotal) break;

    out.push({ path: p, content: c });
    total = nextTotal;
  }
  return out;
}

/* ---------- prompts ---------- */
function rolePrompt(role) {
  return readFileSafe(`agent/prompts/${role}.md`);
}
function systemPrompt(role) {
  return `
You are an implementation agent working in a monorepo.
Return ONLY valid JSON. No markdown. No prose.

Rules:
- Output MUST match the provided JSON schema (strict).
- Overwrite file content exactly as provided (full content).
- Keep changes minimal and scoped to the task.
- Do not include secrets/tokens.
- Prefer adding small files over rewriting large files.
- If task is "fix CI", change the smallest possible surface.

Role instructions:
${rolePrompt(role)}
`.trim();
}
function userPrompt({ role, task, selectedFiles, repoFileList, extra }) {
  const filesBlock = selectedFiles.map(f =>
`--- FILE: ${f.path} ---
${f.content}
`).join("\n");

  return `
Task: ${task}

Selected files (partial, may be truncated):
${filesBlock}

Repo file list:
${repoFileList.join("\n")}

Extra context:
${extra || ""}
`.trim();
}

/* ---------- OpenAI call (strict schema) ---------- */
async function callLLM({ client, role, task, selectedFiles, repoFileList, extra }) {
  const resp = await client.responses.create({
    model: process.env.OPENAI_MODEL || "gpt-5.1",
    max_output_tokens: 9000,
    input: [
      { role: "system", content: systemPrompt(role) },
      { role: "user", content: userPrompt({ role, task, selectedFiles, repoFileList, extra }) },
    ],
    text: {
      format: {
        type: "json_schema",
        name: "file_writes",
        strict: true,
        schema: {
          type: "object",
          additionalProperties: false,
          required: ["files"],
          properties: {
            files: {
              type: "array",
              minItems: 1,
              items: {
                type: "object",
                additionalProperties: false,
                required: ["path", "content"],
                properties: {
                  path: { type: "string" },
                  content: { type: "string" }
                }
              }
            }
          }
        }
      }
    }
  });

  const text = (resp.output_text || "").trim();
  return { text };
}

/* ---------- apply writes ---------- */
function applyFileWrites(obj) {
  if (!obj.files || !Array.isArray(obj.files) || obj.files.length === 0) {
    throw new Error("JSON missing 'files' array");
  }
  for (const f of obj.files) {
    if (!f.path || typeof f.path !== "string" || typeof f.content !== "string") {
      throw new Error(`Bad file entry: ${JSON.stringify(f).slice(0, 200)}`);
    }
    if (f.path.startsWith("/") || f.path.includes("..")) {
      throw new Error(`Unsafe path: ${f.path}`);
    }
    ensureDir(f.path);
    fs.writeFileSync(f.path, f.content, "utf-8");
  }
}

/* ---------- gates ---------- */
function runMakeCI() {
  // use spawn to capture logs
  const r = runCapture("make", ["ci"], { env: process.env });
  return r;
}
function gitDiff() {
  // show only current worktree diff
  return runCapture("git", ["diff"], {}).stdout;
}
function gitDiffStat() {
  return runCapture("git", ["diff", "--stat"], {}).stdout;
}
function hasWorktreeChanges() {
  return sh("git status --porcelain").length > 0;
}

/* ---------- branch mgmt ---------- */
function ensureBranch({ noBranch, role, task }) {
  if (noBranch) return;
  const safe = `${role}-${task}`.toLowerCase().replace(/[^a-z0-9]+/g, "-").slice(0, 40);
  const branch = `agent/${safe}-${nowId()}`;
  must(`git checkout -b ${branch}`);
  return branch;
}

/* ---------- main ---------- */
async function main() {
  const { role, task, noBranch, repair, review } = parseArgs(process.argv);

  const allowed = new Set(["product", "architect", "backend", "qa", "review", "frontend"]);
  if (!allowed.has(role)) {
    console.error(`Invalid role: ${role}. Allowed: ${Array.from(allowed).join(", ")}`);
    process.exit(1);
  }
  if (!task) {
    console.error('Usage: node agent/runner/run.js --role <product|architect|backend|qa|review> [--no-branch] [--repair N] [--no-review] "task"');
    process.exit(1);
  }

  const apiKey = process.env.OPENAI_API_KEY;
  if (!apiKey) {
    console.error("OPENAI_API_KEY is not set");
    process.exit(1);
  }
  const client = new OpenAI({ apiKey });

  const branch = ensureBranch({ noBranch, role, task });
  const repoFiles = listRepoFiles();
  const selectedPaths = pickFilesByRole(role, repoFiles);
  const selectedFiles = readSelectedFiles(selectedPaths);

  // 1) initial implementation
  const raw1 = await callLLM({ client, role, task, selectedFiles, repoFileList: repoFiles, extra: "" });
  fs.writeFileSync(".agent.raw.json", raw1.text, "utf-8");

  let obj1;
  try { obj1 = JSON.parse(raw1.text); } catch {
    console.error("Model did not return valid JSON. Saved to .agent.raw.json");
    process.exit(1);
  }
  applyFileWrites(obj1);

  // 2) self-healing loop
  let lastCI = runMakeCI();
  fs.writeFileSync(".agent.last_ci.txt", lastCI.stdout + "\n" + lastCI.stderr, "utf-8");

  for (let attempt = 1; attempt <= repair && lastCI.code !== 0; attempt++) {
    const extra = `
CI FAILED (attempt ${attempt}):
Exit code: ${lastCI.code}

STDOUT:
${lastCI.stdout}

STDERR:
${lastCI.stderr}

Current git diff --stat:
${gitDiffStat()}

Fix the CI failure with minimal changes.
`.trim();

    const fixRole = role === role;
    const fixTask = `fix CI failure for task: "${task}"`;

    const fixPaths = pickFilesByRole(fixRole, repoFiles);
    const fixFiles = readSelectedFiles(fixPaths);

    const rawFix = await callLLM({ client, role: fixRole, task: fixTask, selectedFiles: fixFiles, repoFileList: repoFiles, extra });
    fs.writeFileSync(".agent.raw.fix.json", rawFix.text, "utf-8");

    let objFix;
    try { objFix = JSON.parse(rawFix.text); } catch {
      console.error("Fix model did not return valid JSON. Saved to .agent.raw.fix.json");
      process.exit(1);
    }
    applyFileWrites(objFix);

    lastCI = runMakeCI();
    fs.writeFileSync(".agent.last_ci.txt", lastCI.stdout + "\n" + lastCI.stderr, "utf-8");
  }

  if (lastCI.code !== 0) {
    console.error("make ci failed after repair attempts. See .agent.last_ci.txt and .agent.raw*.json");
    process.exit(1);
  }

  // 3) review gate (optional)
  if (review) {
    const extra = `
Please review the CURRENT changes against DoD.
If changes are acceptable: create/update docs/review-notes.md with 3-7 bullets, and DO NOT change code.
If changes are not acceptable: apply minimal fixes needed to meet DoD.
Current git diff:
${gitDiff()}
`.trim();

    const reviewPaths = pickFilesByRole("review", repoFiles);
    const reviewFiles = readSelectedFiles(reviewPaths);

    const rawRev = await callLLM({ client, role: "review", task: `review changes for: "${task}"`, selectedFiles: reviewFiles, repoFileList: repoFiles, extra });
    fs.writeFileSync(".agent.raw.review.json", rawRev.text, "utf-8");

    let objRev;
    try { objRev = JSON.parse(rawRev.text); } catch {
      console.error("Review model did not return valid JSON. Saved to .agent.raw.review.json");
      process.exit(1);
    }
    applyFileWrites(objRev);

    // run gates again ONLY if review changed worktree
    if (hasWorktreeChanges()) {
      const ci2 = runMakeCI();
      fs.writeFileSync(".agent.last_ci.txt", ci2.stdout + "\n" + ci2.stderr, "utf-8");
      if (ci2.code !== 0) {
        console.error("make ci failed after review changes. See .agent.last_ci.txt");
        process.exit(1);
      }
    }
  }

  // 4) commit
  if (!hasWorktreeChanges()) {
    console.log("No changes to commit.");
    process.exit(0);
  }

  must("git add -A");
  must(`git commit -m "agent(${role}): ${task}"`);

  if (branch) {
    console.log(`OK. Branch created: ${branch}`);
    console.log(`Next: git push -u origin ${branch}`);
  } else {
    console.log("OK. Committed on current branch.");
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
