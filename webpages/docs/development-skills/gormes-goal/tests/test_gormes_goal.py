import json
import os
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "gormes_goal.py"


def run_goal(tmp_path, *args, session="test-session"):
    env = os.environ.copy()
    env["GORMES_GOAL_DB"] = str(tmp_path / "goals.sqlite")
    env["GORMES_GOAL_SESSION_ID"] = session
    return subprocess.run(
        [sys.executable, str(SCRIPT), *args],
        env=env,
        text=True,
        capture_output=True,
        check=False,
    )


def test_set_status_pause_resume_complete(tmp_path):
    result = run_goal(tmp_path, "invoke", "--tokens", "98.5K", "improve benchmark coverage")
    assert result.returncode == 0, result.stderr
    assert "Action: set" in result.stdout
    assert "Token budget: 98.5K" in result.stdout
    assert "<objective>" in result.stdout
    assert "~/.agents/skills/gormes-goal/scripts/gormes_goal.py complete" in result.stdout

    result = run_goal(tmp_path, "pause")
    assert result.returncode == 0, result.stderr
    assert "Status: paused" in result.stdout

    result = run_goal(tmp_path, "resume")
    assert result.returncode == 0, result.stderr
    assert "Status: active" in result.stdout

    result = run_goal(tmp_path, "complete")
    assert result.returncode == 0, result.stderr
    assert "Status: complete" in result.stdout


def test_rejects_empty_and_duplicate_without_replace(tmp_path):
    result = run_goal(tmp_path, "set")
    assert result.returncode == 1
    assert "goal objective must not be empty" in result.stderr

    assert run_goal(tmp_path, "set", "first objective").returncode == 0
    result = run_goal(tmp_path, "set", "second objective")
    assert result.returncode == 1
    assert "already has a goal" in result.stderr


def test_json_output(tmp_path):
    assert run_goal(tmp_path, "set", "ship the thing").returncode == 0
    result = run_goal(tmp_path, "json")
    assert result.returncode == 0, result.stderr
    data = json.loads(result.stdout)
    assert data["objective"] == "ship the thing"
    assert data["status"] == "active"
    assert data["source"] == "gormes-goal"


def test_stop_hook_blocks_active_goal(tmp_path):
    assert run_goal(tmp_path, "set", "keep going").returncode == 0
    env = os.environ.copy()
    env["GORMES_GOAL_DB"] = str(tmp_path / "goals.sqlite")
    env["GORMES_GOAL_SESSION_ID"] = "test-session"
    result = subprocess.run(
        [sys.executable, str(SCRIPT), "stop-hook"],
        input=json.dumps({"session_id": "test-session", "stop_hook_active": False}),
        env=env,
        text=True,
        capture_output=True,
        check=False,
    )
    assert result.returncode == 0, result.stderr
    data = json.loads(result.stdout)
    assert data["decision"] == "block"
    assert "<objective>" in data["reason"]
    assert "~/.agents/skills/gormes-goal/scripts/gormes_goal.py complete" in data["reason"]


def test_stop_hook_allows_paused_goal(tmp_path):
    assert run_goal(tmp_path, "set", "keep going").returncode == 0
    assert run_goal(tmp_path, "pause").returncode == 0
    env = os.environ.copy()
    env["GORMES_GOAL_DB"] = str(tmp_path / "goals.sqlite")
    env["GORMES_GOAL_SESSION_ID"] = "test-session"
    result = subprocess.run(
        [sys.executable, str(SCRIPT), "stop-hook"],
        input=json.dumps({"session_id": "test-session"}),
        env=env,
        text=True,
        capture_output=True,
        check=False,
    )
    assert result.returncode == 0, result.stderr
    assert result.stdout == ""


def test_gormes_session_id_isolated(tmp_path):
    db = str(tmp_path / "goals.sqlite")
    env_a = os.environ.copy()
    env_a["GORMES_GOAL_DB"] = db
    env_a.pop("GORMES_GOAL_SESSION_ID", None)
    env_a["GORMES_SESSION_ID"] = "gormes-a"
    assert subprocess.run([sys.executable, str(SCRIPT), "set", "A goal"], env=env_a).returncode == 0

    env_b = os.environ.copy()
    env_b["GORMES_GOAL_DB"] = db
    env_b.pop("GORMES_GOAL_SESSION_ID", None)
    env_b["GORMES_SESSION_ID"] = "gormes-b"
    status_b = subprocess.run(
        [sys.executable, str(SCRIPT), "status"],
        env=env_b,
        text=True,
        capture_output=True,
        check=False,
    )
    assert status_b.returncode == 0, status_b.stderr
    assert "No goal is currently set" in status_b.stdout
    assert "A goal" not in status_b.stdout
