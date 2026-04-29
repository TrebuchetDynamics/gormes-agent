---
name: touchdesigner-mcp
description: "Control a running TouchDesigner instance via twozero MCP — create operators, set parameters, wire connections, execute Python, build real-time visuals. 36 native tools."
version: 1.0.0
author: kshitijk4poor
license: MIT
triggers:
  - TouchDesigner network or operator work
  - Building real-time visuals, generative art, or audio-reactive patches in TouchDesigner
  - Calling twozero MCP tools such as td_create_operator, td_set_operator_pars, or td_execute_python
exclusions:
  - Running TouchDesigner setup scripts, downloading the twozero plugin, or starting an MCP server
  - Loading this skill while TouchDesigner or the twozero MCP server is unreachable
  - Sending TouchDesigner traffic outside localhost
review:
  state: reviewed
  source: hermes:853ed609
prerequisites:
  credential_groups:
    - any_of: [TOUCHDESIGNER_MCP_URL, TWOZERO_MCP_URL]
metadata:
  hermes:
    tags: [TouchDesigner, MCP, twozero, creative-coding, real-time-visuals, generative-art, audio-reactive, VJ, installation, GLSL]
    related_skills: [native-mcp, ascii-video, manim-video, hermes-video]
  gormes:
    source: upstream-hermes
    trust_class: [operator, system]
---

# TouchDesigner Integration (twozero MCP)

Use this cookbook only after Gormes has marked the skill available; if `TOUCHDESIGNER_MCP_URL` or `TWOZERO_MCP_URL` is missing, the skill stays out of prompts and Gormes does not start TouchDesigner, the twozero plugin, or any MCP server on the user's behalf.

## CRITICAL RULES

1. **NEVER guess parameter names.** Call `td_get_par_info` for the operator type FIRST. Training data is wrong for current TD versions.
2. **If `tdAttributeError` fires, STOP.** Call `td_get_operator_info` on the failing node before continuing.
3. **NEVER hardcode absolute paths** in script callbacks. Use `me.parent()` or `scriptOp.parent()`.
4. **Prefer native MCP tools over `td_execute_python`.** Use `td_create_operator`, `td_set_operator_pars`, `td_get_errors` first; only fall back to `td_execute_python` for complex multi-step logic.
5. **Call `td_get_hints` before building.** It returns patterns specific to the operator type.

## Architecture

```
Gormes Agent -> MCP (Streamable HTTP) -> twozero.tox (port 40404) -> TD Python
```

36 native tools. The hub health check `GET http://localhost:40404/mcp` returns JSON with the instance PID, project name, and TD version.

## Availability

This skill is bundled creative content. Gormes never installs the twozero plugin, drags `.tox` files into TouchDesigner, or starts the MCP server. Setup is the user's responsibility, performed exactly once per machine outside of Gormes.

After the user finishes manual setup they expose the MCP URL through one of:

- `TOUCHDESIGNER_MCP_URL` — preferred Gormes name.
- `TWOZERO_MCP_URL` — accepted alias matching the twozero plugin docs.

If neither variable is set, Gormes lists the skill as unavailable with `missing-prerequisite` evidence and excludes it from prompts.

## Workflow

### Step 0: Discover (before building anything)

```
Call td_get_par_info with op_type for each type you plan to use.
Call td_get_hints with the topic you're building (e.g. "glsl", "audio reactive", "feedback").
Call td_get_focus to see where the user is and what's selected.
Call td_get_network to see what already exists.
```

### Step 1: Clean + Build

Use `td_create_operator` for each node:

```
td_create_operator(type="noiseTOP", parent="/project1", name="bg", parameters={"resolutionw": 1280, "resolutionh": 720})
td_create_operator(type="levelTOP", parent="/project1", name="brightness")
td_create_operator(type="nullTOP", parent="/project1", name="out")
```

For bulk creation or wiring, use `td_execute_python`:

```python
# td_execute_python script:
root = op('/project1')
nodes = []
for name, optype in [('bg', noiseTOP), ('fx', levelTOP), ('out', nullTOP)]:
    n = root.create(optype, name)
    nodes.append(n.path)
for i in range(len(nodes)-1):
    op(nodes[i]).outputConnectors[0].connect(op(nodes[i+1]).inputConnectors[0])
result = {'created': nodes}
```

### Step 2: Set Parameters

Prefer the native tool, which validates parameter names:

```
td_set_operator_pars(path="/project1/bg", parameters={"roughness": 0.6, "monochrome": true})
```

For expressions or modes, use `td_execute_python`:

```python
op('/project1/time_driver').par.colorr.expr = "absTime.seconds % 1000.0"
```

### Step 3: Wire

```python
op('/project1/bg').outputConnectors[0].connect(op('/project1/fx').inputConnectors[0])
```

### Step 4: Verify

```
td_get_errors(path="/project1", recursive=true)
td_get_perf()
td_get_operator_info(path="/project1/out", detail="full")
```

### Step 5: Display / Capture

```
td_get_screenshot(path="/project1/out")
```

## MCP Tool Quick Reference

**Core:**
| Tool | What |
|------|------|
| `td_execute_python` | Run arbitrary Python in TD. Full API access. |
| `td_create_operator` | Create node with params and auto-positioning. |
| `td_set_operator_pars` | Set params safely (validates, won't crash). |
| `td_get_operator_info` | Inspect one node: connections, params, errors. |
| `td_get_operators_info` | Inspect multiple nodes in one call. |
| `td_get_network` | See network structure at a path. |
| `td_get_errors` | Find errors and warnings recursively. |
| `td_get_par_info` | Get parameter names for an operator type. |
| `td_get_hints` | Get patterns and tips before building. |
| `td_get_focus` | What network is open and what is selected. |

**Read/Write:** `td_read_dat`, `td_write_dat`, `td_read_chop`, `td_read_textport`.

**Visual:** `td_get_screenshot`, `td_get_screenshots`, `td_get_screen_screenshot`, `td_navigate_to`.

**Search:** `td_find_op`, `td_search`.

**System:** `td_get_perf`, `td_list_instances`, `td_get_docs`, `td_agents_md`, `td_reinit_extension`, `td_clear_textport`.

**Input automation:** `td_input_execute`, `td_input_status`, `td_input_clear`, `td_op_screen_rect`, `td_click_screen_point`.

The complete inert tool catalog is in `references/mcp-tools.md`. That reference is bundled documentation only and is not parsed or executed as a live MCP descriptor by Gormes.

## Operator Quick Reference

| Family | Color | Python class / MCP type | Suffix |
|--------|-------|-------------------------|--------|
| TOP | Purple | noiseTOP, glslTOP, compositeTOP, levelTOP, blurTOP, textTOP, nullTOP | TOP |
| CHOP | Green | audiofileinCHOP, audiospectrumCHOP, mathCHOP, lfoCHOP, constantCHOP | CHOP |
| SOP | Blue | gridSOP, sphereSOP, transformSOP, noiseSOP | SOP |
| DAT | White | textDAT, tableDAT, scriptDAT, webserverDAT | DAT |
| MAT | Yellow | phongMAT, pbrMAT, glslMAT, constMAT | MAT |
| COMP | Gray | geometryCOMP, containerCOMP, cameraCOMP, lightCOMP, windowCOMP | COMP |

## Safety Boundaries

- Do not invoke twozero MCP tools unless `TOUCHDESIGNER_MCP_URL` or `TWOZERO_MCP_URL` is present and the user asks for TouchDesigner work.
- Do not infer parameter names from partial responses; call `td_get_par_info` first.
- Do not request commercial-license codecs (H.264, H.265, AV1) on Non-Commercial TouchDesigner. Prefer `prores` on macOS or `mjpa` as a fallback.
- TouchDesigner MCP listens on localhost only. Never proxy or forward port 40404 to non-local hosts.
- `td_execute_python` has unrestricted access to the TouchDesigner Python environment. Treat it as a privileged tool.
