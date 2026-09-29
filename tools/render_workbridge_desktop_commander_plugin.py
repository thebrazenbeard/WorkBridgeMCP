from __future__ import annotations

import argparse
import json
import shutil
from pathlib import Path
from urllib.parse import urlsplit


class RenderError(RuntimeError):
    pass


def validate_url(value: str) -> str:
    if not value:
        raise RenderError("MCP URL is required")
    parsed=urlsplit(value)
    if parsed.scheme != "https" or not parsed.hostname:
        raise RenderError("MCP URL must be absolute HTTPS")
    if not parsed.path or not parsed.path.endswith("/mcp"):
        raise RenderError("MCP URL path must end with /mcp")
    if parsed.query or parsed.fragment or parsed.username or parsed.password:
        raise RenderError("MCP URL contains unsupported URL components")
    return value.rstrip("/")


def render(source_dir: str|Path, output_dir: str|Path, *, mcp_url: str) -> Path:
    source=Path(source_dir).resolve()
    output=Path(output_dir).resolve()
    mcp_url=validate_url(mcp_url)
    if output.exists():
        raise RenderError(f"refusing to overwrite existing output: {output}")
    for required in ("plugin.json",".codex-plugin/plugin.json","tool-contract.json","skills/workbridge-desktop-commander/SKILL.md"):
        if not (source/required).is_file():
            raise RenderError(f"required source missing: {required}")
    shutil.copytree(source,output,ignore=shutil.ignore_patterns("mcp.template.json","__pycache__"))
    portable={
        "$schema":"https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
        "mcpServers":{"workbridge_desktop_commander":{"type":"streamable-http","url":mcp_url}},
    }
    legacy={"mcpServers":{"workbridge_desktop_commander":{"type":"streamable-http","url":mcp_url,"headers":{}}}}
    (output/"mcp.json").write_text(json.dumps(portable,indent=2)+"\n",encoding="utf-8")
    (output/".mcp.json").write_text(json.dumps(legacy,indent=2)+"\n",encoding="utf-8")
    return output


def main():
    p=argparse.ArgumentParser(description="Render WorkBridge Desktop Commander ChatGPT plugin")
    p.add_argument("--source",required=True);p.add_argument("--output",required=True);p.add_argument("--mcp-url",required=True)
    a=p.parse_args()
    print(render(a.source,a.output,mcp_url=a.mcp_url))


if __name__=="__main__":
    main()
