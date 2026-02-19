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

function parseArgs(argv) {
  const args = argv.slice(2);
  let role = "backend";

  const roleIdx = args.indexOf("--role");
  if (roleIdx !== -1) {
    role = (args[roleIdx + 1] || "").trim();
    args.splice(roleIdx, 2);
  }

  const task = args.join(" ").trim();
  return { role, task };
}

function repoContext() {
  const files = sh("git ls-files").split("\n").filter(Boolean);
  return {
    dod: readFileSafe("docs/dod.md"),
    arch: readFileSafe("docs/architecture.md"),
    api: readFileSafe("docs/api.md"),
    events: readFileSafe("docs/events.md"),
    docsReadme: readFileSafe("docs/README.md"),
    tree: files,
  };
}

function rolePrompt(role) {
  const p = `agent/prompts/${role}.md`;
  return readFileSafe(p);
}

function systemPrompt(role) {
  return `
You are an implementation agent working in a monorepo.
Return ONLY valid JSON. No markdown. No prose.

You MUST follow the JSON schema provided by the caller (strict).

Rules:
- Overwrite file content exactly as provided (full content).
- Keep changes minimal and scoped to the task.
- Do not include secrets or tokens.
- Prefer small targeted changes; avoid rewriting large unrelated files.

Role instructions:
${rolePrompt(role)}
`.trim();
}

function userPrompt(task, ctx) {
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

Existing docs/README.md (if any):
${ctx.docsReadme}

Repo file list:
${ctx.tree.join("\n")}
`.trim();
}

async function main() {
  const { role, task } = parseArgs(process.argv);

  if (!task) {
    console.error('Usage: node agent/runner/run.js --role <product|architect|backend|qa|review> "task"');
    process.exit(1);
  }

  const allowed = new Set(["product", "architect", "backend", "qa", "review"]);
  if (!allowed.has(role)) {
    console.error(`Invalid role: ${role}. Allowed: ${Array.from(allowed).join(", ")}`);
    process.exit(1);
  }

  const apiKey = process.env.OPENAI_API_KEY;
  if (!apiKey) {
    console.error("OPENAI_API_KEY is not set");
    process.exit(1);
  }

  // create branch
  const safe = `${role}-${task}`.toLowerCase().replace(/[^a-z0-9]+/g, "-").slice(0, 40);
  const branch = `agent/${safe}-${Date.now()}`;
  must(`git checkout -b ${branch}`);

  const ctx = repoContext();
  const client = new OpenAI({ apiKey });

  const resp = await client.responses.create({
    model: process.env.OPENAI_MODEL || "gpt-4.1-mini",
    max_output_tokens: 8000,
    input: [
      { role: "system", content: systemPrompt(role) },
      { role: "user", content: userPrompt(task, ctx) },
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
  fs.writeFileSync(".agent.raw.json", text, "utf-8");

  let obj;
  try {
    obj = JSON.parse(text);
  } catch {
    console.error("Model did not return valid JSON. Raw saved to .agent.raw.json");
    console.error(text.slice(0, 500));
    process.exit(1);
  }

  if (!obj.files || !Array.isArray(obj.files) || obj.files.length === 0) {
    console.error("JSON missing 'files' array");
    console.error(obj);
    process.exit(1);
  }

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

  try {
    must("make ci");
  } catch {
    console.error("make ci failed. Inspect diff and fix manually.");
    process.exit(1);
  }

  must("git add -A");
  must(`git commit -m "agent(${role}): ${task}"`);

  console.log(`OK. Branch created: ${branch}`);
  console.log(`Next: git push -u origin ${branch}`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
