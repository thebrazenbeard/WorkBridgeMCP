from __future__ import annotations

import json
import re
import tempfile
import unittest
from pathlib import Path

from tools.render_workbridge_desktop_commander_plugin import RenderError, render, validate_url

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "plugin" / "workbridge-desktop-commander"
UPSTREAM = ROOT / "upstream" / "DesktopCommanderMCP"


class PluginContractTests(unittest.TestCase):
    def test_contract_matches_exact_pinned_upstream_tool_names(self):
        contract = json.loads((SOURCE / "tool-contract.json").read_text(encoding="utf-8"))
        server = (UPSTREAM / "src" / "server.ts").read_text(encoding="utf-8")
        observed = set(re.findall(r'name:\s*"([^"]+)"', server))
        observed.discard("desktop-commander")
        observed.discard("*.js")
        self.assertEqual(set(contract["required_tools"]), observed)
        self.assertEqual(
            "550a0b3e31da18b7cf25e87ed840e3d953b6da42",
            contract["upstream"]["commit"],
        )
        self.assertEqual("0.2.51", contract["upstream"]["version"])
        self.assertFalse(contract["transport"]["semantic_translation"])

    def test_start_process_contract_is_native_command_string(self):
        contract = json.loads((SOURCE / "tool-contract.json").read_text(encoding="utf-8"))
        proc = contract["required_process_contract"]
        self.assertEqual("start_process", proc["tool"])
        self.assertEqual("command", proc["required_argument"])
        self.assertEqual("string", proc["argument_type"])
        self.assertTrue(proc["unrestricted_command_string_shell"])
        server = (UPSTREAM / "src" / "server.ts").read_text(encoding="utf-8")
        schemas = (UPSTREAM / "src" / "tools" / "schemas.ts").read_text(encoding="utf-8")
        self.assertIn('name: "start_process"', server)
        self.assertRegex(schemas, r"StartProcessArgsSchema[\s\S]*command")

    def test_plugin_is_direct_mcp_and_has_no_v2_app_dependency(self):
        manifest = json.loads((SOURCE / "plugin.json").read_text(encoding="utf-8"))
        self.assertNotIn("apps", manifest["extensions"]["com.openai"])
        overlay = json.loads(
            (SOURCE / ".codex-plugin" / "plugin.json").read_text(encoding="utf-8")
        )
        self.assertEqual("./.mcp.json", overlay["mcpServers"])
        source_text = "\n".join(
            p.read_text(encoding="utf-8")
            for p in (
                SOURCE / "plugin.json",
                SOURCE / "mcp.template.json",
                SOURCE / "README.md",
            )
        )
        self.assertIn("__WORKBRIDGE_DESKTOP_COMMANDER_MCP_URL__", source_text)
        self.assertNotIn("trycloudflare.com", source_text)

    def test_private_render_preserves_only_transport_secret(self):
        with tempfile.TemporaryDirectory() as td:
            out = Path(td) / "rendered"
            url = "https://example.invalid/capability/mcp"
            render(SOURCE, out, mcp_url=url)
            portable = json.loads((out / "mcp.json").read_text(encoding="utf-8"))
            self.assertEqual(
                url,
                portable["mcpServers"]["workbridge_desktop_commander"]["url"],
            )
            self.assertFalse((out / "mcp.template.json").exists())
            self.assertTrue((out / "tool-contract.json").is_file())

    def test_url_policy(self):
        self.assertEqual(
            "https://mesh.example/cap/mcp",
            validate_url("https://mesh.example/cap/mcp"),
        )
        for value in (
            "http://mesh.example/cap/mcp",
            "https://mesh.example/",
            "https://mesh.example/cap",
            "https://u@mesh.example/cap/mcp",
            "https://mesh.example/cap/mcp?q=1",
        ):
            with self.subTest(value=value):
                with self.assertRaises(RenderError):
                    validate_url(value)


if __name__ == "__main__":
    unittest.main()
