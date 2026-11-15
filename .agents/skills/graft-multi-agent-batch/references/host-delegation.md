# Host Delegation Mapping

Apply [root delegation policy](../../../../AGENTS.md#delegation). Model choice is autonomous within the direct-parent ceiling; repeat the check for nested dispatch, replacements, and retries.

For the current Codex collaboration API:

- `fork_turns="all"` or omission inherits the parent's model and reasoning effort; omit explicit overrides.
- For a verified same/lower explicit model, use `fork_turns="none"` or a positive turn count and the tool's supported `model`/`reasoning_effort` fields.
- Record the parent and selected model, relation, and comparison source supplied by the host or current official documentation. Full configuration inheritance proves `same` without a ranking table.
- If the relative rank cannot be verified, inherit the parent and continue. Do not infer rank from model names, availability, or reasoning effort, and do not pause merely to ask for a model choice.
- Never dispatch a higher model under the current policy.

Use the actual enabled tool schema on another host; keep its parameter mapping here rather than in general repository governance. Do not maintain a static cross-model ranking in the repository.
