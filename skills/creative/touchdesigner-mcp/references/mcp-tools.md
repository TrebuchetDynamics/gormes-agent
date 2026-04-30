# twozero MCP Tools Reference (Inert)

This reference documents the 36 native twozero MCP tools that the upstream
TouchDesigner skill calls when the operator's TouchDesigner instance is
running and `TOUCHDESIGNER_MCP_URL` (or `TWOZERO_MCP_URL`) is reachable.

It is bundled inert documentation. Gormes does not start a TouchDesigner
process, install the twozero plugin, dial the MCP server, or treat this file
as a live MCP descriptor. It is text content only.

## Execution & Scripting

### td_execute_python

Run Python inside TouchDesigner and return the result. Full TD Python API
access (op, project, app, etc). Print statements and the last expression
value are captured.

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | string | yes | Python code to execute in TouchDesigner. |

### td_create_operator

Create one operator with parameters and automatic viewport positioning.

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | yes | Operator type, e.g. `noiseTOP`, `levelTOP`, `nullTOP`. |
| `parent` | string | yes | Parent COMP path, e.g. `/project1`. |
| `name` | string | yes | New operator name. |
| `parameters` | object | no | Parameter values to apply after creation. |

### td_set_operator_pars

Set parameters on an existing operator with validation.

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | yes | Operator path. |
| `parameters` | object | yes | Parameter name to value map. |

## Network & Inspection

- `td_get_network` — return compact network structure at a path.
- `td_get_operator_info` — inspect one node: connections, params, errors.
- `td_get_operators_info` — inspect multiple nodes in one call.
- `td_get_errors` — find errors and warnings recursively.
- `td_get_par_info` — return parameter names and types for an operator type.
- `td_get_hints` — return patterns and tips for a topic.
- `td_get_focus` — return the open network and current selection.

## Read & Write

- `td_read_dat` — read DAT text content.
- `td_write_dat` — write or patch DAT content.
- `td_read_chop` — read CHOP channel values.
- `td_read_textport` — read TouchDesigner console output.

## Visual

- `td_get_screenshot` — capture one operator viewer to file.
- `td_get_screenshots` — capture multiple operators at once.
- `td_get_screen_screenshot` — capture the actual screen via TouchDesigner.
- `td_navigate_to` — jump the network editor to an operator.

## Search

- `td_find_op` — find operators by name or type across the project.
- `td_search` — search code, expressions, and string parameters.

## System

- `td_get_perf` — performance profiling (FPS, slow operators).
- `td_list_instances` — list all running TouchDesigner instances.
- `td_get_docs` — return in-depth docs on a TouchDesigner topic.
- `td_agents_md` — read or write per-COMP markdown docs.
- `td_reinit_extension` — reload an extension after a code edit.
- `td_clear_textport` — clear the console before a debug session.

## Input Automation

- `td_input_execute` — send mouse or keyboard input to TouchDesigner.
- `td_input_status` — poll the input queue status.
- `td_input_clear` — stop input automation.
- `td_op_screen_rect` — return screen coordinates of a node.
- `td_click_screen_point` — click a point in a screenshot.

## Notes

- Every tool accepts an optional `target_instance` parameter for multi-TD
  scenarios.
- All tools reach TouchDesigner exclusively through the local twozero MCP
  endpoint (default `http://localhost:40404/mcp`). Gormes never proxies that
  endpoint or forwards traffic outside the operator's machine.
- This file is descriptive prose. It is not a runtime MCP manifest, an OpenAPI
  document, or a Gormes SKILL.md descriptor. Tools and prerequisites are
  defined exclusively by `SKILL.md` frontmatter.
