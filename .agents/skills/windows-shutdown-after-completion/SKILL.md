---
name: windows-shutdown-after-completion
description: Schedule a Windows shutdown only when the user explicitly invokes `$shutdown-after-completion` (or unambiguously names this exact skill) after asking to power down following fully completed work. Do not invoke implicitly, from installation or indirect mentions, or while requested work remains unfinished.
---

# Shutdown After Completion

Use this skill only after the active user-requested task has reached its normal final validation and closeout condition.

1. Confirm that all requested work is complete and that the applicable validation and closeout have finished. If work remains, do not schedule shutdown; finish or report the remaining work first.
2. Tell the user: `Windows will shut down in 60 seconds.`
3. Execute exactly this command from WSL:

```bash
cmd.exe /c "shutdown /s /t 60"
```

Do not execute the command merely because this skill is installed, read, referenced indirectly, or mentioned without an explicit invocation. Do not substitute, amend, cancel, or repeat the command unless the user explicitly requests it.
