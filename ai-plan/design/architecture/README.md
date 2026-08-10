# Architecture

This directory is for repository-wide structural design truth.

## Use This Directory For

- backend/module baseline architecture
- frontend shell and module structure
- cross-repository structural design that explains how major layers fit together
- navigation and resource-route information architecture

## Do Not Use This Directory For

- capability-specific product design
- topic-local migration notes
- long-lived guardrails that fit better under `../governance/`

## Current Scope Note

The baseline architecture authorities now live in this directory. Keep new repository-wide structural design documents
here instead of restoring them to the `ai-plan/design/` root.

## Navigation And Routes

- [导航与资源路由信息架构规范.md](导航与资源路由信息架构规范.md) is the authority for Graft's visible navigation domains,
  resource-oriented UI routes, global entries, and runtime/source placement rules.
- [项目文件组织与扩展点设计.md](项目文件组织与扩展点设计.md) is the authority for the target server layout, Provider/
  Adapter/Integration boundaries, independent Agents, runners, deployments, and conformance fixtures.
