import fs from "fs";
import path from "path";
import { execSync } from "child_process";
import OpenAI from "openai";

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

function repoContext() {
  const files = sh("git ls-files").split("\n").filter(Boolean);

  // беремо тільки важливі доки (щоб не роздувати контекст)
  const ctx = {
    dod: readFileSafe("docs/dod.md"),
    arch: readFileSafe("docs/architecture.md"),
    api: readFileSafe("docs/api.md"),
    events: readFileSafe("docs/events.md"),
    tree: files,
    // додаємо існуючий docs/README.md, бо саме він часто конфліктує
    docsReadme: readFileSafe("docs/README.md"),
  };
  return ctx;
}

const SYSTEM = `
You are an implementation agent working in a monorepo.
Return ONLY valid JSON. No markdown. No prose.

JSON schema:
{
  "files": [
    { "path": "relative/path", "content": "full file content" }
  ]
}

Rules:
- Overwrite file content exactly as provided (full content).
- Keep changes minimal and scoped to the task.
- Do not include secrets or tokens.
- Ensure changes will not break "make ci".
`;

function USER(task, ctx) {
  return `
Task: ${task}

Repo constraints (DoD):
${ctx.dod}

Architecture:
${ctx.arch}

API:
${ctx.api}

Events:
${ctx.events}

Existing docs/README.md content (if any):
${ctx.docsReadme}

Repo file list:
${ctx.tree.join("\n")}

Return JSON only.
`;
}

async function main() {
  const task = process.argv.slice(2).join(" ").trim();
  if (!task) {
    console.error('Usage: node agent/runner/run.js "task description"');
    process.exit(1);
  }

  const apiKey = process.env.OPENAI_API_KEY;
  if (!apiKey) {
    console.error("OPENAI_API_KEY is not set");
    process.exit(1);
  }

  // create branch
  const safe = task.toLowerCase().replace(/[^a-z0-9]+/g, "-").slice(0, 40);
  const branch = `agent/${safe}-${Date.now()}`;
  must(`git checkout -b ${branch}`);

  const ctx = repoContext();
  const client = new OpenAI({ apiKey });

  const resp = await client.responses.create({
    model: process.env.OPENAI_MODEL || "gpt-4.1-mini",
    max_output_tokens: 2000,
    input: [
      { role: "system", content: SYSTEM },
      { role: "user", content: USER(task, ctx) },
    ],
  });

  const text = resp.output_text?.trim();
  if (!text) {
    console.error("Empty model output");
    process.exit(1);
  }

  let obj;
  try {
    obj = JSON.parse(text);
  } catch (e) {
    console.error("Model did not return valid JSON. Output was:");
    console.error(text);
    process.exit(1);
  }

  if (!obj.files || !Array.isArray(obj.files) || obj.files.length === 0) {
    console.error("JSON missing 'files' array");
    console.error(obj);
    process.exit(1);
  }

  // write files
  for (const f of obj.files) {
    if (!f.path || typeof f.path !== "string" || typeof f.content !== "string") {
      console.error("Bad file entry:", f);
      process.exit(1);
    }
    if (f.path.startsWith("/") || f.path.includes("..")) {
      console.error("Unsafe path:", f.path);
      process.exit(1);
    }
    ensureDir(f.path);
    fs.writeFileSync(f.path, f.content, "utf-8");
  }

  // gates
  try {
    must("make ci");
  } catch {
    console.error("make ci failed. Inspect diff and fix manually.");
    process.exit(1);
  }

  must("git add -A");
  must(`git commit -m "agent: ${task}"`);

  console.log(`OK. Branch created: ${branch}`);
  console.log("Next:");
  console.log(`  git push -u origin ${branch}`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
