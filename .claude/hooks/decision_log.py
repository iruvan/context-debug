#!/usr/bin/env python3
"""Stop-hook: append each turn's raw (unsummarized) exchange to docs/decision_log_<date>.md.

Reads the Claude Code transcript JSONL, extracts only the new lines since the
last run (tracked per session in .claude/hooks/.state/), and appends the real
user prompt + assistant text/tool-use for this turn verbatim. No LLM rewriting.
"""
import json
import os
import sys
from datetime import datetime, timezone

def load_state(state_file):
    if os.path.exists(state_file):
        try:
            return int(open(state_file).read().strip() or "0")
        except ValueError:
            return 0
    return 0

def save_state(state_file, offset):
    os.makedirs(os.path.dirname(state_file), exist_ok=True)
    with open(state_file, "w") as f:
        f.write(str(offset))

def extract_user_text(content):
    if isinstance(content, str):
        return content if content.strip() else None
    if isinstance(content, list):
        # Real user prompts are text/image blocks; tool_result entries are
        # synthetic "user" turns fed back from tool execution, not decisions.
        if any(block.get("type") == "tool_result" for block in content):
            return None
        texts = [b.get("text", "") for b in content if b.get("type") == "text"]
        joined = "\n".join(t for t in texts if t.strip())
        return joined if joined.strip() else None
    return None

def extract_assistant_blocks(content):
    parts = []
    if not isinstance(content, list):
        return parts
    for block in content:
        btype = block.get("type")
        if btype == "text" and block.get("text", "").strip():
            parts.append(("text", block["text"]))
        elif btype == "thinking" and block.get("thinking", "").strip():
            parts.append(("thinking", block["thinking"]))
        elif btype == "tool_use":
            name = block.get("name", "tool")
            try:
                inp = json.dumps(block.get("input", {}), ensure_ascii=False)
            except Exception:
                inp = str(block.get("input", {}))
            if len(inp) > 800:
                inp = inp[:800] + "...(truncated)"
            parts.append(("tool_use", f"{name}({inp})"))
    return parts

def format_timestamp(ts):
    try:
        dt = datetime.fromisoformat(ts.replace("Z", "+00:00")).astimezone()
        return dt.strftime("%H:%M:%S")
    except Exception:
        return ts or ""

def main():
    try:
        payload = json.load(sys.stdin)
    except Exception:
        return 0

    transcript_path = payload.get("transcript_path")
    session_id = payload.get("session_id", "unknown-session")
    if payload.get("stop_hook_active"):
        return 0
    if not transcript_path or not os.path.isfile(transcript_path):
        return 0

    project_root = os.getcwd()
    state_file = os.path.join(project_root, ".claude", "hooks", ".state", f"{session_id}.offset")
    offset = load_state(state_file)

    with open(transcript_path, "r", encoding="utf-8") as f:
        lines = f.readlines()

    new_lines = lines[offset:]
    if not new_lines:
        return 0

    entries = []
    for line in new_lines:
        line = line.strip()
        if not line:
            continue
        try:
            obj = json.loads(line)
        except json.JSONDecodeError:
            continue
        if obj.get("isSidechain") or obj.get("isMeta"):
            continue
        etype = obj.get("type")
        ts = format_timestamp(obj.get("timestamp"))
        if etype == "user":
            text = extract_user_text(obj.get("message", {}).get("content"))
            if text:
                entries.append((ts, "User", [("text", text)]))
        elif etype == "assistant":
            blocks = extract_assistant_blocks(obj.get("message", {}).get("content"))
            if blocks:
                model = obj.get("message", {}).get("model", "")
                role = f"Assistant ({model})" if model else "Assistant"
                entries.append((ts, role, blocks))

    save_state(state_file, len(lines))

    if not entries:
        return 0

    docs_dir = os.path.join(project_root, "docs/decision_log")
    os.makedirs(docs_dir, exist_ok=True)
    date_str = datetime.now().strftime("%d_%b_%Y").lower()
    short_session_id = session_id.split("-")[0]
    log_file = os.path.join(docs_dir, f"{date_str}_{short_session_id}.md")

    is_new = not os.path.exists(log_file)
    with open(log_file, "a", encoding="utf-8") as f:
        if is_new:
            f.write(f"# Decision Log — {datetime.now().strftime('%Y-%m-%d')} (session {session_id})\n\n")
        for ts, role, blocks in entries:
            f.write(f"## {ts} — {role}\n\n")
            for kind, content in blocks:
                if kind == "text":
                    f.write(f"{content}\n\n")
                elif kind == "thinking":
                    f.write(f"<details><summary>thinking</summary>\n\n{content}\n\n</details>\n\n")
                elif kind == "tool_use":
                    f.write(f"→ `{content}`\n\n")
        f.write("---\n\n")

    return 0

if __name__ == "__main__":
    sys.exit(main())
