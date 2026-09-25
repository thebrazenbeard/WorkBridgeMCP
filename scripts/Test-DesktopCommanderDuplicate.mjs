import { spawn } from "node:child_process";
import path from "node:path";
import process from "node:process";

const root = path.resolve("upstream", "DesktopCommanderMCP");
const entrypoint = path.join(root, "dist", "index.js");
const child = spawn(process.execPath, [entrypoint, "--no-onboarding"], {
  cwd: root,
  stdio: ["pipe", "pipe", "pipe"],
  env: { ...process.env, DC_REMOTE_DEVICE: "true" },
  windowsHide: true,
});

let buffer = "";
let stderr = "";
let nextId = 1;
const pending = new Map();

function fail(message) {
  try { child.kill(); } catch {}
  throw new Error(message + (stderr ? "\nstderr:\n" + stderr.slice(-4000) : ""));
}

child.stderr.on("data", (chunk) => {
  stderr += chunk.toString();
});

child.stdout.on("data", (chunk) => {
  buffer += chunk.toString();
  const lines = buffer.split("\n");
  buffer = lines.pop() || "";
  for (const raw of lines) {
    const line = raw.trim();
    if (!line) continue;
    let msg;
    try {
      msg = JSON.parse(line);
    } catch {
      continue;
    }
    if (msg.id !== undefined && pending.has(msg.id)) {
      const { resolve, reject, timer } = pending.get(msg.id);
      clearTimeout(timer);
      pending.delete(msg.id);
      if (msg.error) reject(new Error(JSON.stringify(msg.error)));
      else resolve(msg.result);
    }
  }
});

child.on("error", (err) => {
  for (const { reject, timer } of pending.values()) {
    clearTimeout(timer);
    reject(err);
  }
  pending.clear();
});

function request(method, params = {}, timeoutMs = 15000) {
  const id = nextId++;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      pending.delete(id);
      reject(new Error("timeout waiting for " + method));
    }, timeoutMs);
    pending.set(id, { resolve, reject, timer });
    child.stdin.write(JSON.stringify({ jsonrpc: "2.0", id, method, params }) + "\n");
  });
}

function notify(method, params = {}) {
  child.stdin.write(JSON.stringify({ jsonrpc: "2.0", method, params }) + "\n");
}

try {
  const init = await request("initialize", {
    protocolVersion: "2024-11-05",
    capabilities: {},
    clientInfo: { name: "workbridge-duplicate-ci", version: "1.0.0" },
  });
  if (!init?.serverInfo?.name) fail("Desktop Commander initialize returned no serverInfo");
  notify("notifications/initialized", {});

  const listed = await request("tools/list", {});
  const tools = listed?.tools || [];
  const byName = new Map(tools.map((tool) => [tool.name, tool]));
  const required = [
    "read_file",
    "read_multiple_files",
    "write_file",
    "create_directory",
    "list_directory",
    "move_file",
    "start_search",
    "get_more_search_results",
    "stop_search",
    "list_searches",
    "get_file_info",
    "edit_block",
    "start_process",
    "read_process_output",
    "interact_with_process",
    "force_terminate",
    "list_sessions",
    "list_processes",
    "kill_process",
    "get_recent_tool_calls",
  ];
  const missing = required.filter((name) => !byName.has(name));
  if (missing.length) fail("missing Desktop Commander tools: " + missing.join(", "));

  const start = byName.get("start_process");
  const props = start?.inputSchema?.properties || {};
  const requiredArgs = new Set(start?.inputSchema?.required || []);
  if (!props.command || !requiredArgs.has("command")) {
    fail("start_process is not the unrestricted command-string API");
  }

  const marker = "WORKBRIDGE_DUPLICATE_OK";
  const command = "node -e \"process.stdout.write('" + marker + "')\"";
  const started = await request("tools/call", {
    name: "start_process",
    arguments: { command, timeout_ms: 5000 },
  }, 20000);
  const text = (started?.content || [])
    .filter((item) => item?.type === "text")
    .map((item) => item.text || "")
    .join("\n");

  if (!text.includes(marker)) {
    const match = text.match(/PID\s+(-?\d+)/i);
    if (!match) fail("start_process returned neither marker nor PID: " + text);
    const pid = Number(match[1]);
    const output = await request("tools/call", {
      name: "read_process_output",
      arguments: { pid, offset: -20, length: 20, timeout_ms: 2000 },
    }, 10000);
    const outputText = (output?.content || [])
      .filter((item) => item?.type === "text")
      .map((item) => item.text || "")
      .join("\n");
    if (!outputText.includes(marker)) {
      fail("unrestricted command-string execution did not return marker");
    }
  }

  console.log(JSON.stringify({
    status: "PASS",
    upstream: "wonderwhy-er/DesktopCommanderMCP@550a0b3e31da18b7cf25e87ed840e3d953b6da42",
    server: init.serverInfo,
    tool_count: tools.length,
    unrestricted_command_string_shell: true,
  }));
} finally {
  try { child.stdin.end(); } catch {}
  try { child.kill(); } catch {}
}
