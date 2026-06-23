#!/usr/bin/env python3

from __future__ import annotations

import re
import sys
from dataclasses import dataclass
from pathlib import Path


REPO_SENTINEL = "AGENTS.md"
GO_TYPE_RE = re.compile(r"^type\s+([A-Za-z_][A-Za-z0-9_]*)\s*(=)?\s*(.*)$")
DTO_TOKENS = ("Request", "Response", "Payload", "DTO", "Params")
REQUEST_TOKENS = ("Request", "Payload", "DTO", "Params")
RESPONSE_TOKENS = ("Response",)
NON_DTO_SUFFIXES = ("Config", "Options", "Result", "Reader", "Manager", "Bridge", "Guard", "Registrar", "Handler")
FORBIDDEN_GENERATED_RUNTIME_PATTERNS = (
    "NewStrictHandler(",
    "RegisterHandlers(",
    "RegisterHandlersWithOptions(",
    "HandlerFromMux(",
    "HandlerWithOptions(",
    "ServerInterfaceWrapper",
    "StrictHandlerFunc",
    "StrictHTTPServerOptions",
)

HANDLER_BOUNDARY_CHECKS = {
    "server/modules/auth/route_handlers.go": (
        "authopenapi.PostAuthLoginJSONRequestBody",
        "authopenapi.GetAuthSessionsParams",
        "authopenapi.PostAuthChangePasswordJSONRequestBody",
    ),
    "server/modules/user/route_user_handlers.go": (
        "useropenapi.GetUsersParams",
        "useropenapi.PostUsersJSONRequestBody",
        "useropenapi.PostUserUpdateJSONRequestBody",
    ),
    "server/modules/user/route_admin_session_handlers.go": (
        "useropenapi.GetUserSessionsParams",
        "useropenapi.PostUserSessionsRevokeAllParams",
        "useropenapi.PostUserSessionRevokeParams",
    ),
    "server/modules/rbac/route_read_handlers.go": (
        "rbacopenapi.GetPermissionsParams",
        "rbacopenapi.GetRolesParams",
        "rbacopenapi.GetRolePermissionsParams",
        "rbacopenapi.GetUserRolesParams",
    ),
    "server/modules/rbac/route_write_handlers.go": (
        "rbacopenapi.PostRolesJSONRequestBody",
        "rbacopenapi.PostRoleUpdateJSONRequestBody",
        "rbacopenapi.PostRoleStatusJSONRequestBody",
        "rbacopenapi.PostRolePermissionsReplaceJSONRequestBody",
        "rbacopenapi.PostRolePermissionsAddJSONRequestBody",
        "rbacopenapi.PostRolePermissionsRemoveJSONRequestBody",
        "rbacopenapi.PostUserRolesReplaceJSONRequestBody",
        "rbacopenapi.PostUserRolesAddJSONRequestBody",
        "rbacopenapi.PostUserRolesRemoveJSONRequestBody",
        "rbacopenapi.PostUsersRolesReplaceJSONRequestBody",
        "rbacopenapi.PostUsersRolesAddJSONRequestBody",
        "rbacopenapi.PostUsersRolesRemoveJSONRequestBody",
    ),
    "server/modules/monitor/module.go": (
        "monitoropenapi.GetMonitorServerStatusParams",
        "generated.ServerStatusResponse",
    ),
}

RESPONSE_MAPPER_CHECKS = {
    "server/modules/auth/mapper_http.go": (
        "generated.LoginResponse",
        "generated.BootstrapResponse",
        "generated.SessionSummary",
    ),
    "server/modules/user/mapper_http.go": (
        "generated.UserListResponse",
        "generated.UserListItem",
        "generated.SessionSummary",
    ),
    "server/modules/rbac/mapper_http.go": (
        "generated.RoleListResponse",
        "generated.RolePermissionBindingResponse",
        "generated.PermissionListResponse",
        "generated.UserRoleBindingResponse",
    ),
    "server/modules/monitor/module.go": (
        "generated.ServerStatusResponse",
        "generated.ServerStatusServer",
        "generated.ServerStatusDependencies",
    ),
    "server/internal/moduleruntime/snapshot.go": (
        "generated.ModuleRuntimeSnapshot",
        "generated.ModuleRuntimeItem",
        "generated.ModuleRuntimeDependency",
    ),
}

ALLOWED_EXACT_TYPES: dict[str, tuple[str, str]] = {
    "server/modules/user/bootstrap.go:bootstrapResponse": (
        "generated_mapper_allowed",
        "internal bootstrap bridge model consumed by authFlowBridge before auth mapper emits generated.BootstrapResponse",
    ),
    "server/modules/user/bootstrap.go:bootstrapMenuResponse": (
        "generated_mapper_allowed",
        "internal bootstrap menu bridge model consumed by authFlowBridge before auth mapper emits generated.BootstrapResponse",
    ),
    "server/modules/user/bootstrap.go:bootstrapLocaleSnapshot": (
        "generated_mapper_allowed",
        "internal bootstrap locale bridge model consumed by authFlowBridge before auth mapper emits generated.BootstrapResponse",
    ),
    "server/modules/user/session.go:loginUserResponse": (
        "service_command_allowed",
        "internal auth flow identity snapshot used before pluginapi/auth generated mappers",
    ),
    "server/modules/user/session.go:refreshResult": (
        "service_command_allowed",
        "internal auth flow result model used before pluginapi.AuthRefreshResult and generated.LoginResponse mapping",
    ),
    "server/modules/user/session.go:sessionSummary": (
        "service_command_allowed",
        "internal auth session summary model used before generated.SessionSummary mapping",
    ),
}


@dataclass
class AuditResult:
    generated_boundary_type: list[str]
    generated_mapper_allowed: list[str]
    service_command_allowed: list[str]
    domain_or_ent_model_allowed: list[str]
    httpx_runtime_allowed: list[str]
    stale_manual_api_request_dto: list[str]
    stale_manual_api_response_dto: list[str]
    warnings: list[str]
    violations: list[str]
    generated_boundary_preserved: bool
    httpx_runtime_preserved: bool
    generated_runtime_used: bool


def find_repo_root() -> Path:
    current = Path.cwd().resolve()
    for candidate in (current, *current.parents):
        if (candidate / REPO_SENTINEL).is_file() and (candidate / "server").is_dir():
            return candidate
    raise SystemExit(f"could not locate repository root containing {REPO_SENTINEL}")


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def is_candidate_type(name: str) -> bool:
    if name.endswith(NON_DTO_SUFFIXES):
        return False
    return any(token in name for token in DTO_TOKENS)


def is_request_type(name: str) -> bool:
    return any(token in name for token in REQUEST_TOKENS)


def is_response_type(name: str) -> bool:
    return any(token in name for token in RESPONSE_TOKENS)


def classify_manual_dto(rel_path: str, type_name: str) -> str:
    if is_request_type(type_name):
        return "stale_manual_api_request_dto"
    if is_response_type(type_name):
        return "stale_manual_api_response_dto"
    return "suspicious"


def append_category(result: AuditResult, category: str, value: str) -> None:
    getattr(result, category).append(value)


def audit_type_declarations(repo_root: Path, result: AuditResult) -> None:
    for path in sorted((repo_root / "server/modules").rglob("*.go")):
        rel_path = path.relative_to(repo_root).as_posix()
        if rel_path.endswith("_test.go"):
            continue
        if any(part in {"contract", "ent", "store", "storeent", "migrations"} for part in path.parts):
            continue

        for lineno, line in enumerate(read_text(path).splitlines(), start=1):
            match = GO_TYPE_RE.match(line.strip())
            if not match:
                continue

            type_name = match.group(1)
            if not is_candidate_type(type_name):
                continue

            key = f"{rel_path}:{type_name}"
            if key in ALLOWED_EXACT_TYPES:
                category, reason = ALLOWED_EXACT_TYPES[key]
                append_category(result, category, f"{rel_path}:{lineno} `{type_name}`: {reason}")
                continue

            alias = match.group(2) == "="
            rest = match.group(3)
            if alias and "generated." in rest:
                result.generated_boundary_type.append(
                    f"{rel_path}:{lineno} `{type_name}` aliases `{rest.strip()}`"
                )
                continue

            file_name = path.name
            if file_name.startswith("dto_http"):
                category = classify_manual_dto(rel_path, type_name)
                append_category(
                    result,
                    category,
                    f"{rel_path}:{lineno} `{type_name}` declared under `dto_http*` boundary file",
                )
                continue

            if any(token in file_name for token in ("handler", "api", "route")):
                category = classify_manual_dto(rel_path, type_name)
                append_category(
                    result,
                    category,
                    f"{rel_path}:{lineno} `{type_name}` declared inside HTTP boundary file `{file_name}`",
                )


def audit_generated_boundary_usage(repo_root: Path, result: AuditResult) -> None:
    """
    验证指定 Go 文件中预期的生成标记的存在性。
    
    检查处理器边界文件和响应映射器文件是否包含配置中要求的生成标记。当标记缺失时，向结果添加警告并将边界保持状态标记为假；当标记全部存在时，记录允许的发现。
    """
    for rel_path, patterns in HANDLER_BOUNDARY_CHECKS.items():
        text = read_text(repo_root / rel_path)
        missing = [pattern for pattern in patterns if pattern not in text]
        if missing:
            result.warnings.append(
                f"{rel_path}: expected generated boundary markers missing: {', '.join(missing)}"
            )
            result.generated_boundary_preserved = False
            continue
        result.generated_boundary_type.append(
            f"{rel_path}: handler boundary binds generated markers `{', '.join(patterns)}`"
        )

    for rel_path, patterns in RESPONSE_MAPPER_CHECKS.items():
        text = read_text(repo_root / rel_path)
        missing = [pattern for pattern in patterns if pattern not in text]
        if missing:
            result.warnings.append(
                f"{rel_path}: expected generated response mapper markers missing: {', '.join(missing)}"
            )
            result.generated_boundary_preserved = False
            continue
        result.generated_mapper_allowed.append(
            f"{rel_path}: response mapper emits generated markers `{', '.join(patterns)}`"
        )


def audit_httpx_runtime(repo_root: Path, result: AuditResult) -> None:
    """
    验证指定的 Go 文件包含所有预期的 httpx 运行时所有权标记。
    
    若任何必需的标记缺失，则记录警告并将运行时所有权状态标记为未保留；
    若所有标记存在，则记录该文件的运行时所有权已保留。
    
    Parameters:
        repo_root (Path): 代码仓库的根目录
        result (AuditResult): 用于记录审计发现的结果对象
    """
    runtime_checks = {
        "server/modules/auth/route_handlers.go": ("httpx.WriteSuccess", "writeLocalizedContractError"),
        "server/modules/user/route_user_handlers.go": ("httpx.WriteSuccess", "writeLocalizedContractError"),
        "server/modules/user/route_admin_session_handlers.go": ("httpx.WriteSuccess",),
        "server/modules/rbac/route_read_handlers.go": ("httpx.WriteSuccess", "httpx.AbortLocalizedError"),
        "server/modules/rbac/route_write_handlers.go": ("httpx.WriteSuccess",),
        "server/modules/monitor/module.go": ("httpx.WriteSuccess", "httpx.AbortLocalizedError", "ctx.Router.Group"),
        "server/internal/moduleruntime/registration.go": ("httpx.WriteSuccess", "httpx.AbortLocalizedError", "router.Group"),
    }

    for rel_path, patterns in runtime_checks.items():
        text = read_text(repo_root / rel_path)
        missing = [pattern for pattern in patterns if pattern not in text]
        if missing:
            result.warnings.append(
                f"{rel_path}: expected runtime ownership markers missing: {', '.join(missing)}"
            )
            result.httpx_runtime_preserved = False
            continue
        result.httpx_runtime_allowed.append(
            f"{rel_path}: runtime ownership stays on `{', '.join(patterns)}`"
        )


def audit_generated_runtime_takeover(repo_root: Path, result: AuditResult) -> None:
    """
    检测禁止的生成代码运行时接管标记，并在审计结果中记录发现的违规项。
    
    Parameters:
        repo_root (Path): 仓库根目录路径
        result (AuditResult): 审计结果对象，将被修改以记录发现的违规项
    """
    for path in sorted((repo_root / "server").rglob("*.go")):
        rel_path = path.relative_to(repo_root).as_posix()
        text = read_text(path)
        for pattern in FORBIDDEN_GENERATED_RUNTIME_PATTERNS:
            if pattern in text:
                result.generated_runtime_used = True
                result.violations.append(
                    f"{rel_path}: found generated runtime takeover marker `{pattern}`"
                )


def build_result() -> AuditResult:
    """
    创建初始化的审计结果对象。
    
    Returns:
        AuditResult: 包含空发现列表和默认保留状态的审计结果对象。
    """
    return AuditResult(
        generated_boundary_type=[],
        generated_mapper_allowed=[],
        service_command_allowed=[],
        domain_or_ent_model_allowed=[
            "server/modules/user/store/**: repository-owned user/auth write and read models remain internal",
            "server/modules/rbac/store/**: repository-owned RBAC domain/read models remain internal",
            "server/modules/*/ent/**: Ent models stay outside the HTTP API boundary",
        ],
        httpx_runtime_allowed=[],
        stale_manual_api_request_dto=[],
        stale_manual_api_response_dto=[],
        warnings=[],
        violations=[],
        generated_boundary_preserved=True,
        httpx_runtime_preserved=True,
        generated_runtime_used=False,
    )


def verdict(result: AuditResult) -> str:
    """
    根据审计结果确定最终的审查结论。
    
    参数:
        result (AuditResult): 包含审计发现的结果对象
    
    返回值:
        str: 审查结论，为以下之一：
            - "FAILED_VIOLATIONS"：存在违规项或过期的手工 DTO
            - "PASS_WITH_WARNINGS"：存在警告或边界保护失效
            - "PASS_ALLOWED_BOUNDARY_MODELS"：所有审计条件均满足
    """
    if result.stale_manual_api_request_dto or result.stale_manual_api_response_dto or result.violations:
        return "FAILED_VIOLATIONS"
    if result.warnings or not result.generated_boundary_preserved or not result.httpx_runtime_preserved:
        return "PASS_WITH_WARNINGS"
    return "PASS_ALLOWED_BOUNDARY_MODELS"


def print_category(title: str, items: list[str]) -> None:
    """
    打印格式化的分类列表。
    
    如果列表为空，打印占位符"none"；否则打印列表中的每一项，每项前缀为"  - "。
    """
    print(f"{title}:")
    if not items:
        print("  - none")
        return
    for item in items:
        print(f"  - {item}")


def main() -> int:
    """
    执行后端 DTO 边界审计并输出检查报告。
    
    遍历服务器源文件以分类 DTO 类型声明、验证生成边界标记、检查 HTTP 运行时所有权和禁用的生成代码接管模式。
    汇总审计结果并按类别打印允许项、警告和违规项。
    
    Returns:
        int: 1 如果发现违规项，0 否则
    """
    repo_root = find_repo_root()
    result = build_result()
    audit_type_declarations(repo_root, result)
    audit_generated_boundary_usage(repo_root, result)
    audit_httpx_runtime(repo_root, result)
    audit_generated_runtime_takeover(repo_root, result)

    final_verdict = verdict(result)
    print(f"Backend DTO Boundary Verdict: {final_verdict}")
    print("Allowed Findings:")
    print_category("generated_boundary_type", result.generated_boundary_type)
    print_category("generated_mapper_allowed", result.generated_mapper_allowed)
    print_category("service_command_allowed", result.service_command_allowed)
    print_category("domain_or_ent_model_allowed", result.domain_or_ent_model_allowed)
    print_category("httpx_runtime_allowed", result.httpx_runtime_allowed)
    print("Warnings:")
    print_category("warnings", result.warnings)
    print("Violations:")
    print_category("stale_manual_api_request_dto", result.stale_manual_api_request_dto)
    print_category("stale_manual_api_response_dto", result.stale_manual_api_response_dto)
    print_category("violations", result.violations)
    print(f"generated_boundary_preserved: {result.generated_boundary_preserved}")
    print(f"httpx_runtime_preserved: {result.httpx_runtime_preserved}")
    print(f"generated_runtime_used: {result.generated_runtime_used}")

    if final_verdict == "FAILED_VIOLATIONS":
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
