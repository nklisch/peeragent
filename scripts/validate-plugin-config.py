#!/usr/bin/env python3
"""Validate bundled MCP configuration and smoke-test the packaged server."""

from __future__ import annotations

import json
import selectors
import subprocess
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
EXPECTED_TOOLS = {"delegate", "job_status", "job_result", "job_cancel"}


def fail(message: str) -> None:
    raise AssertionError(message)


def load_json(path: Path) -> Any:
    if not path.is_file():
        fail(f"missing JSON file: {path.relative_to(ROOT)}")
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"invalid JSON in {path.relative_to(ROOT)}: {exc}")


def require_keys(value: Any, expected: set[str], label: str) -> None:
    if not isinstance(value, dict):
        fail(f"{label} must be an object")
    actual = set(value)
    if actual != expected:
        fail(f"{label} keys are {sorted(actual)}, expected {sorted(expected)}")


def validate_manifest(path: Path, config_name: str) -> None:
    manifest = load_json(path)
    if manifest.get("mcpServers") != config_name:
        fail(
            f"{path.relative_to(ROOT)} must point mcpServers at {config_name!r}; "
            f"got {manifest.get('mcpServers')!r}"
        )


def validate_server_config(
    path: Path, expected_command: str, expected_cwd: str | None = None
) -> None:
    config = load_json(path)
    require_keys(config, {"mcpServers"}, f"{path.relative_to(ROOT)} root")
    servers = config["mcpServers"]
    require_keys(servers, {"peeragent"}, f"{path.relative_to(ROOT)} servers")
    server = servers["peeragent"]
    expected_keys = {"command", "args"}
    if expected_cwd is not None:
        expected_keys.add("cwd")
    require_keys(server, expected_keys, f"{path.relative_to(ROOT)} peeragent server")
    if server["command"] != expected_command:
        fail(
            f"{path.relative_to(ROOT)} command is {server['command']!r}, "
            f"expected {expected_command!r}"
        )
    if server["args"] != ["mcp"]:
        fail(f"{path.relative_to(ROOT)} args must be ['mcp'], got {server['args']!r}")
    if expected_cwd is not None and server["cwd"] != expected_cwd:
        fail(
            f"{path.relative_to(ROOT)} cwd is {server['cwd']!r}, "
            f"expected {expected_cwd!r}"
        )


def validate_mirror(source: Path, packaged: Path) -> None:
    if source.read_bytes() != packaged.read_bytes():
        fail(
            f"packaged {packaged.relative_to(ROOT)} differs from source "
            f"{source.relative_to(ROOT)}; run scripts/package-plugin.sh"
        )


def protocol_smoke() -> None:
    shim = ROOT / "plugin" / "bin" / "peeragent"
    if not shim.is_file() or not shim.stat().st_mode & 0o111:
        fail(f"packaged shim is missing or not executable: {shim.relative_to(ROOT)}")

    initialize = json.dumps(
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2025-06-18",
                "capabilities": {},
                "clientInfo": {"name": "validation-smoke", "version": "1.0.0"},
            },
        }
    )
    initialized = json.dumps(
        {"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}}
    )
    list_tools = json.dumps(
        {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}
    )

    # Send requests interactively. The Go SDK may defer a response until the
    # next request is read, so writing all frames and closing stdin at once can
    # make a valid server look like an EOF failure instead of a protocol reply.
    process: subprocess.Popen[str] | None = None
    try:
        process = subprocess.Popen(
            [str(shim), "mcp"],
            cwd=shim.parent.parent,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
        )
        assert process.stdin is not None
        assert process.stdout is not None
        def read_response(label: str) -> str:
            selector = selectors.DefaultSelector()
            try:
                selector.register(process.stdout, selectors.EVENT_READ)
                if not selector.select(timeout=10):
                    fail(f"packaged MCP protocol smoke timed out waiting for {label}")
                line = process.stdout.readline()
            finally:
                selector.close()
            if not line:
                fail(f"packaged MCP protocol smoke received no {label} response")
            return line

        process.stdin.write(initialize + "\n")
        process.stdin.flush()
        initialize_line = read_response("initialize")
        process.stdin.write(initialized + "\n" + list_tools + "\n")
        process.stdin.flush()
        tools_line = read_response("tools/list")
        if not tools_line:
            fail("packaged MCP protocol smoke received no tools/list response")
        process.stdin.close()
        returncode = process.wait(timeout=3)
        stderr = process.stderr.read() if process.stderr is not None else ""
    except (OSError, subprocess.TimeoutExpired) as exc:
        fail(f"packaged MCP protocol smoke could not complete: {exc}")
    finally:
        if process is not None and process.poll() is None:
            process.kill()
            process.wait()

    if returncode != 0:
        fail(f"packaged MCP protocol smoke exited {returncode}; stderr: {stderr.strip()}")
    lines = [line for line in (initialize_line, tools_line) if line.strip()]

    frames: list[dict[str, Any]] = []
    for line in lines:
        try:
            frame = json.loads(line)
        except json.JSONDecodeError as exc:
            fail(f"packaged MCP stdout contained non-JSON output {line!r}: {exc}")
        if frame.get("jsonrpc") != "2.0":
            fail(f"packaged MCP stdout contained a non-JSON-RPC frame: {frame!r}")
        frames.append(frame)

    by_id = {frame.get("id"): frame for frame in frames}
    if 1 not in by_id or 2 not in by_id:
        fail(f"packaged MCP smoke responses missing initialize/tools-list ids: {frames!r}")
    tools = by_id[2].get("result", {}).get("tools")
    if not isinstance(tools, list):
        fail(f"packaged MCP tools/list response has no tools array: {by_id[2]!r}")
    names = {tool.get("name") for tool in tools if isinstance(tool, dict)}
    if names != EXPECTED_TOOLS or len(tools) != len(EXPECTED_TOOLS):
        fail(f"packaged MCP tools are {sorted(names)}, expected {sorted(EXPECTED_TOOLS)}")


def main() -> int:
    try:
        source_claude_config = ROOT / ".mcp.claude.json"
        source_codex_config = ROOT / ".mcp.json"
        packaged_claude_config = ROOT / "plugin" / ".mcp.claude.json"
        packaged_codex_config = ROOT / "plugin" / ".mcp.json"

        validate_manifest(ROOT / ".claude-plugin" / "plugin.json", "./.mcp.claude.json")
        validate_manifest(ROOT / ".codex-plugin" / "plugin.json", "./.mcp.json")
        validate_manifest(
            ROOT / "plugin" / ".claude-plugin" / "plugin.json", "./.mcp.claude.json"
        )
        validate_manifest(
            ROOT / "plugin" / ".codex-plugin" / "plugin.json", "./.mcp.json"
        )
        validate_server_config(
            source_claude_config, "${CLAUDE_PLUGIN_ROOT}/bin/peeragent"
        )
        validate_server_config(source_codex_config, "./bin/peeragent", expected_cwd=".")
        validate_server_config(
            packaged_claude_config, "${CLAUDE_PLUGIN_ROOT}/bin/peeragent"
        )
        validate_server_config(
            packaged_codex_config, "./bin/peeragent", expected_cwd="."
        )

        validate_mirror(source_claude_config, packaged_claude_config)
        validate_mirror(source_codex_config, packaged_codex_config)
        validate_mirror(
            ROOT / ".claude-plugin" / "plugin.json",
            ROOT / "plugin" / ".claude-plugin" / "plugin.json",
        )
        validate_mirror(
            ROOT / ".codex-plugin" / "plugin.json",
            ROOT / "plugin" / ".codex-plugin" / "plugin.json",
        )
        protocol_smoke()
    except AssertionError as exc:
        print(f"MCP plugin validation failed: {exc}", file=sys.stderr)
        return 1

    print("MCP plugin configurations and packaged protocol smoke passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
