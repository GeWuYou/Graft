---
name: graft-module-architecture-review
description: Review Graft module and interface design for ownership, depth, locality, dependency direction, and platform extensibility. Use during design before introducing packages, adapters, shared helpers, or runtime surfaces.
---

# Graft Module Architecture Review

Use this as the semantic architecture review for module shape. It complements `graft-platform-architecture-review`,
server/web module rules, and validation; it never creates a second architecture authority or review gate.

## Workflow

1. Establish the startup receipt and identify the product capability, owning module, task class, and canonical contract.
2. Map the module interface, implementation, adapters, dependencies, lifecycle, and callers. Use **depth**, **seam**,
   **locality**, and **leverage** to describe the design.
3. Apply the deletion test: if the module disappeared, would complexity vanish or return across callers? A pass-through
   wrapper, one-adapter seam, or generic helper needs concrete variation or policy to justify itself.
4. Check dependency direction: business modules use stable contracts and runtime capabilities, never another module's
   internal implementation. Startup, registration, goroutines, and shutdown must have one visible owner.
5. Check testability at the public interface. Prefer injected external dependencies and behavior tests at the highest
   useful seam; do not expose private seams merely for tests.
6. Check platform fit: compile-time module registration, Application First, existing Task/Submission and configuration
   authority, and no duplicate runtime, DI, shell, or source-of-truth system.

## Findings

Report `blocking`, `high`, `medium`, or `note` findings with evidence, affected callers, smallest repair, and validation.
Conclude with `ownership`, `interface_and_seam`, `depth_and_locality`, `dependency_direction`, `deletion_candidates`,
`platform_fit`, `validation`, and `decisions_needed`. Prefer deleting shallow abstractions over layering another adapter.

## Guardrails

- Do not add DDD, CQRS, ports, repositories, or generics without a concrete invariant, variation, or boundary benefit.
- Do not move business behavior into core merely to simplify one caller.
- Do not replace repository validation, closeout, or architecture documents with this skill's output.
