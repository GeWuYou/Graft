#!/usr/bin/env python3
"""Tests for the Graft Quality Policy eff-u-code evaluator."""

from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("evaluate_eff_u_code_gate.py")
MODULE_SPEC = importlib.util.spec_from_file_location("evaluate_eff_u_code_gate", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")

MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)


def make_metric(name: str, score: float, details: str = "detail") -> dict[str, object]:
    """
    构造一条指标记录。
    
    Parameters:
    	name (str): 指标名称。
    	score (float): 归一化分数。
    	details (str): 指标详情。
    
    Returns:
    	dict[str, object]: 包含 `name`、`normalizedScore` 和 `details` 的指标字典。
    """
    return {
        "name": name,
        "normalizedScore": score,
        "details": details,
    }


def make_file(path: str, metrics: list[dict[str, object]]) -> dict[str, object]:
    """
    构造一个包含路径和指标列表的文件记录。
    
    Parameters:
    	path (str): 文件路径。
    	metrics (list[dict[str, object]]): 该文件的指标列表。
    
    Returns:
    	dict[str, object]: 包含 `path` 和 `metrics` 的文件字典。
    """
    return {"path": path, "metrics": metrics}


class EvaluateRuleTests(unittest.TestCase):
    def test_new_file_below_threshold_fails(self) -> None:
        evaluation = MODULE.evaluate_rule(
            "server/internal/runtime.go",
            "complexity",
            {"metrics": ["cyclomatic_complexity"], "threshold": 75, "regression": 5, "newFileThreshold": 75},
            "cyclomatic_complexity",
            make_metric("cyclomatic_complexity", 70),
            None,
            is_new_file=True,
        )

        self.assertEqual(evaluation.status, "fail")

    def test_existing_file_below_threshold_without_regression_is_legacy_warning(self) -> None:
        evaluation = MODULE.evaluate_rule(
            "server/modules/foo/service.go",
            "duplication",
            {"metrics": ["code_duplication"], "threshold": 75, "regression": 5},
            "code_duplication",
            make_metric("code_duplication", 72),
            make_metric("code_duplication", 74),
            is_new_file=False,
        )

        self.assertEqual(evaluation.status, "legacy-warning")

    def test_display_only_rule_never_blocks(self) -> None:
        evaluation = MODULE.evaluate_rule(
            "web/src/utils/foo.ts",
            "documentation",
            {"metrics": ["comment_ratio"], "mode": "display-only"},
            "comment_ratio",
            make_metric("comment_ratio", 0),
            None,
            is_new_file=True,
        )

        self.assertEqual(evaluation.status, "display-only")

    def test_project_mode_uses_threshold_without_baseline_regression(self) -> None:
        evaluation = MODULE.evaluate_rule(
            "web/src/utils/foo.ts",
            "structure",
            {"metrics": ["structure_analysis"], "threshold": 75, "regression": 5, "newFileThreshold": 70},
            "structure_analysis",
            make_metric("structure_analysis", 74),
            make_metric("structure_analysis", 40),
            is_new_file=False,
            scan_mode="project",
        )

        self.assertEqual(evaluation.status, "fail")

    def test_duplication_noise_exclude_suppresses_declarative_surface(self) -> None:
        evaluation = MODULE.evaluate_rule(
            "server/modules/scheduler/route_registration.go",
            "duplication",
            {
                "metrics": ["code_duplication"],
                "threshold": 75,
                "regression": 5,
                "newFileThreshold": 75,
                "noiseExcludes": ["server/modules/**/route_registration.go"],
            },
            "code_duplication",
            make_metric("code_duplication", 58.5, "10/62 duplicated"),
            make_metric("code_duplication", 90),
            is_new_file=False,
        )

        self.assertEqual(evaluation.status, "suppressed-noise")
        self.assertEqual(evaluation.noise_reason, "declarative duplication surface")

    def test_duplication_structural_mirror_candidates_are_configured_as_noise(self) -> None:
        gate_config = json.loads((MODULE.REPO_ROOT / "scripts" / "eff-u-code.gate.json").read_text(encoding="utf-8"))
        duplication_rule = gate_config["gateRules"]["duplication"]
        candidates = [
            "server/modules/audit/service_normalization.go",
            "server/modules/rbac/storeent/repository_records.go",
            "server/modules/rbac/storeent/repository_scan.go",
        ]

        for path in candidates:
            with self.subTest(path=path):
                evaluation = MODULE.evaluate_rule(
                    path,
                    "duplication",
                    duplication_rule,
                    "code_duplication",
                    make_metric("code_duplication", 52.0, "8/40 duplicated"),
                    make_metric("code_duplication", 90),
                    is_new_file=False,
                )

                self.assertEqual(evaluation.status, "suppressed-noise")
                self.assertEqual(evaluation.noise_reason, "declarative duplication surface")

    def test_reactive_tracking_noise_suppresses_error_handling(self) -> None:
        target = MODULE.REPO_ROOT / "web" / "src" / "modules" / "container" / "pages" / "detail" / "log-view-store.ts"
        target.parent.mkdir(parents=True, exist_ok=True)
        created = False
        if not target.exists():
            target.write_text("export function sample(version: { value: number }): void { void version.value; }\n", encoding="utf-8")
            created = True

        try:
            evaluation = MODULE.evaluate_rule(
                "web/src/modules/container/pages/detail/log-view-store.ts",
                "error_handling",
                {
                    "metrics": ["error_handling"],
                    "threshold": 60,
                    "regression": 5,
                    "newFileThreshold": 60,
                    "noiseExcludes": ["web/src/modules/container/pages/detail/log-view-store.ts"],
                },
                "error_handling",
                make_metric("error_handling", 1.2, "1/1 个错误被忽略 (100.0%)"),
                make_metric("error_handling", 100),
                is_new_file=False,
            )
        finally:
            if created:
                target.unlink(missing_ok=True)

        self.assertEqual(evaluation.status, "suppressed-noise")
        self.assertEqual(evaluation.noise_reason, "reactive-tracking read pattern")

    def test_error_handling_noise_exclude_uses_bounded_policy_noise_reason(self) -> None:
        evaluation = MODULE.evaluate_rule(
            "server/internal/realtime/hub.go",
            "error_handling",
            {
                "metrics": ["error_handling"],
                "threshold": 60,
                "regression": 5,
                "newFileThreshold": 60,
                "noiseExcludes": ["server/internal/realtime/hub.go"],
            },
            "error_handling",
            make_metric("error_handling", 1.2, "5/5 个错误被忽略 (100.0%)"),
            make_metric("error_handling", 100),
            is_new_file=False,
        )

        self.assertEqual(evaluation.status, "suppressed-noise")
        self.assertEqual(evaluation.noise_reason, "bounded policy noise")

    def test_error_handling_runtime_policy_candidates_are_configured_as_bounded_noise(self) -> None:
        gate_config = json.loads((MODULE.REPO_ROOT / "scripts" / "eff-u-code.gate.json").read_text(encoding="utf-8"))
        error_rule = gate_config["gateRules"]["error_handling"]
        candidates = [
            "server/internal/httpx/server.go",
            "server/internal/scheduler/runtime.go",
            "server/modules/auth/storeent/auth_repository.go",
            "server/modules/container/docker_exec_session.go",
            "server/modules/container/docker_runtime_stats.go",
            "server/modules/container/docker_runtime.go",
            "server/modules/container/log_topic_streamer.go",
            "server/modules/container/mount_usage.go",
            "server/modules/container/resource_stats_cache.go",
            "server/modules/container/runtime_event_manager.go",
            "server/modules/container/stats_collector.go",
            "server/modules/container/terminal/websocket_bridge.go",
            "server/modules/monitor/module.go",
            "server/modules/user/storeent/auth_repository.go",
            "web/src/shared/realtime/ws-client.ts",
            "web/src/store/modules/setting.ts",
        ]

        for path in candidates:
            with self.subTest(path=path):
                evaluation = MODULE.evaluate_rule(
                    path,
                    "error_handling",
                    error_rule,
                    "error_handling",
                    make_metric("error_handling", 1.2, "1/1 个错误被忽略 (100.0%)"),
                    make_metric("error_handling", 100),
                    is_new_file=False,
                )

                self.assertEqual(evaluation.status, "suppressed-noise")
                self.assertEqual(evaluation.noise_reason, "bounded policy noise")


class CuratedScoreTests(unittest.TestCase):
    def test_curated_score_ignores_zero_weight_rules(self) -> None:
        gate_config = {
            "curatedScore": {
                "participatesInGate": False,
                "weights": {
                    "complexity": 0.5,
                    "documentation": 0.0,
                },
            }
        }

        score = MODULE.curated_score(
            [
                MODULE.RuleEvaluation("complexity", "cyclomatic_complexity", "pass", 75, 5, 80, 90, "detail"),
                MODULE.RuleEvaluation("documentation", "comment_ratio", "display-only", None, None, 0, None, "detail"),
            ],
            gate_config,
        )

        self.assertEqual(score, 80)

    def test_curated_score_configuration_cannot_participate_in_gate(self) -> None:
        gate_config = {
            "curatedScore": {
                "participatesInGate": True,
                "weights": {"complexity": 1.0},
            }
        }

        with self.assertRaises(MODULE.GateConfigError):
            MODULE.curated_score([], gate_config)


class ScoreGateConfigTests(unittest.TestCase):
    def test_score_gate_enabled_for_matching_profile_and_scan_mode(self) -> None:
        gate_config = {
            "scoreGate": {
                "profiles": {
                    "score-project": {
                        "enabled": True,
                        "enabledScanModes": ["project"],
                        "threshold": 85,
                    }
                }
            }
        }

        self.assertTrue(MODULE.score_gate_enabled(gate_config, "score-project", "project"))
        self.assertFalse(MODULE.score_gate_enabled(gate_config, "score-project", "changed"))

    def test_build_file_diagnostics_tracks_impact_and_severity(self) -> None:
        gate_config = {
            "curatedScore": {
                "participatesInGate": False,
                "weights": {
                    "complexity": 0.5,
                    "duplication": 0.5,
                },
            }
        }
        diagnostics = MODULE.build_file_diagnostics(
            "server/modules/foo/service.go",
            [
                MODULE.RuleEvaluation("complexity", "cognitive_complexity", "fail", 75, 5, 60, 80, "collectFoo()"),
                MODULE.RuleEvaluation("duplication", "code_duplication", "fail", 75, 5, 70, 90, "duplicate block"),
            ],
            gate_config,
            ("complexity", "duplication"),
        )

        self.assertGreater(diagnostics["impact"], 0)
        self.assertEqual(diagnostics["highestSeverity"], "critical")
        self.assertEqual(diagnostics["severitySummary"]["critical"]["count"], 1)


class OverrideConfigTests(unittest.TestCase):
    def test_build_eff_u_code_overrides_preserves_defaults_and_expands_scope_top(self) -> None:
        base_eff_config = {
            "defaults": {"locale": "zh", "format": "console", "top": 20},
            "targets": {
                "server": {"path": "server", "exclude": ["**/*_test.go"]},
            },
        }
        gate_config = {
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/*.go", "server/**/*.go"],
                    "exclude": ["server/**/*_test.go"],
                }
            }
        }
        scoped_candidates = {"server": [f"server/file-{index}.go" for index in range(25)]}

        overrides = MODULE.build_eff_u_code_overrides(base_eff_config, gate_config, scoped_candidates, ["server"])

        self.assertEqual(overrides["defaults"]["locale"], "zh")
        self.assertEqual(overrides["targets"]["server"]["path"], "server")
        self.assertEqual(overrides["targets"]["server"]["top"], 25)
        self.assertEqual(overrides["targets"]["server"]["exclude"], ["**/*_test.go"])

    def test_build_snapshot_eff_config_keeps_required_defaults(self) -> None:
        base_eff_config = {
            "defaults": {"locale": "zh", "format": "console", "top": 20},
            "targets": {
                "server": {"path": "server", "exclude": ["**/*_test.go"]},
            },
        }
        gate_config = {
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/*.go", "server/**/*.go"],
                    "exclude": ["server/**/*_test.go"],
                }
            }
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            snapshot = MODULE.build_snapshot_eff_config(
                base_eff_config,
                gate_config,
                Path(tmp_dir),
                {"server": ["server/main.go"]},
                ["server"],
            )

        self.assertEqual(snapshot["defaults"]["locale"], "zh")
        self.assertEqual(snapshot["targets"]["server"]["top"], 20)
        self.assertTrue(snapshot["targets"]["server"]["path"].endswith("/server"))


class ChangedFileResolutionTests(unittest.TestCase):
    def test_resolve_changed_mode_files_fetches_base_ref_before_merge_base(self) -> None:
        calls: list[tuple[str, ...]] = []

        def fake_run_git(args: list[str]) -> str:
            calls.append(tuple(args))
            if args == ["merge-base", "HEAD", "refs/remotes/origin/main"]:
                return "base-sha"
            if args == ["diff", "--name-only", "--diff-filter=ACMR", "base-sha...HEAD"]:
                return "server/main.go\n"
            raise AssertionError(f"unexpected git args: {args}")

        with mock.patch.object(MODULE, "staged_or_changed_files", return_value=[]), \
            mock.patch.object(MODULE, "run_git", side_effect=fake_run_git), \
            mock.patch.object(MODULE.subprocess, "run") as mock_run, \
            mock.patch.dict("os.environ", {"GRAFT_LINT_BASE_REF": "main"}, clear=False):
            changed, baseline = MODULE.resolve_changed_mode_files("ACMR")

        self.assertEqual(changed, ["server/main.go"])
        self.assertEqual(baseline.revision, "base-sha")
        self.assertEqual(baseline.source, "merge-base")
        mock_run.assert_called_once_with(
            ["git", "fetch", "--no-tags", "--prune", "origin", "main"],
            cwd=MODULE.REPO_ROOT,
            check=False,
            stdout=MODULE.subprocess.PIPE,
            stderr=MODULE.subprocess.PIPE,
            text=True,
        )
        self.assertEqual(calls[0], ("merge-base", "HEAD", "refs/remotes/origin/main"))

    def test_resolve_changed_mode_files_prefers_explicit_sha_for_files_and_baseline(self) -> None:
        def fake_run_git(args: list[str]) -> str:
            if args == ["diff", "--name-only", "--diff-filter=ACMR", "explicit-base...HEAD"]:
                return "server/main.go\nscripts/evaluate_eff_u_code_gate.py\n"
            raise AssertionError(f"unexpected git args: {args}")

        with mock.patch.object(MODULE, "staged_or_changed_files", return_value=[]), \
            mock.patch.object(MODULE, "run_git", side_effect=fake_run_git), \
            mock.patch.dict(
                "os.environ",
                {
                    "GRAFT_QUALITY_BASE_SHA": "explicit-base",
                    "GRAFT_LINT_BASE_REF": "refs/remotes/origin/main",
                },
                clear=False,
            ):
            changed, baseline = MODULE.resolve_changed_mode_files("ACMR")

        self.assertEqual(changed, ["server/main.go", "scripts/evaluate_eff_u_code_gate.py"])
        self.assertEqual(baseline.revision, "explicit-base")
        self.assertEqual(baseline.source, "explicit-sha")
        self.assertEqual(baseline.normalized_base_ref, "refs/remotes/origin/main")

    def test_ci_changed_files_uses_graft_lint_base_ref_before_falling_back_to_tracked_files(self) -> None:
        def fake_run_git(args: list[str]) -> str:
            if args == ["merge-base", "HEAD", "refs/remotes/origin/main"]:
                return "base-sha"
            if args == ["diff", "--name-only", "--diff-filter=ACMR", "base-sha...HEAD"]:
                return "server/main.go\nscripts/evaluate_eff_u_code_gate.py\n"
            raise AssertionError(f"unexpected git args: {args}")

        with mock.patch.object(MODULE, "staged_or_changed_files", return_value=[]), \
            mock.patch.object(MODULE, "run_git", side_effect=fake_run_git), \
            mock.patch.dict("os.environ", {"GRAFT_LINT_BASE_REF": "refs/remotes/origin/main"}, clear=False):
            changed = MODULE.ci_changed_files("ACMR")

        self.assertEqual(changed, ["server/main.go", "scripts/evaluate_eff_u_code_gate.py"])

    def test_normalize_remote_base_ref_accepts_common_forms(self) -> None:
        self.assertEqual(MODULE.normalize_remote_base_ref("main"), "refs/remotes/origin/main")
        self.assertEqual(MODULE.normalize_remote_base_ref("origin/main"), "refs/remotes/origin/main")
        self.assertEqual(MODULE.normalize_remote_base_ref("refs/heads/main"), "refs/remotes/origin/main")
        self.assertEqual(MODULE.normalize_remote_base_ref("refs/remotes/origin/main"), "refs/remotes/origin/main")

    def test_fetch_target_from_base_ref_accepts_common_forms(self) -> None:
        self.assertEqual(MODULE.fetch_target_from_base_ref("main"), "main")
        self.assertEqual(MODULE.fetch_target_from_base_ref("origin/main"), "main")
        self.assertEqual(MODULE.fetch_target_from_base_ref("refs/heads/main"), "main")
        self.assertEqual(MODULE.fetch_target_from_base_ref("refs/remotes/origin/main"), "main")


class ScoreDiagnosticsTests(unittest.TestCase):
    def test_build_file_diagnostics_only_accumulates_contributing_categories(self) -> None:
        gate_config = {
            "curatedScore": {
                "participatesInGate": False,
                "weights": {"complexity": 1.0, "documentation": 1.0},
            }
        }
        diagnostics = MODULE.build_file_diagnostics(
            "server/internal/runtime.go",
            [
                MODULE.RuleEvaluation("complexity", "cyclomatic_complexity", "fail", 75, 5, 60, 90, "complex code"),
                MODULE.RuleEvaluation("documentation", "comment_ratio", "display-only", None, None, 0, None, "missing comments"),
            ],
            gate_config,
            ("complexity", "documentation"),
        )

        self.assertEqual(diagnostics["issueCount"], 1)
        self.assertEqual(diagnostics["categoryImpacts"], [{"rule": "complexity", "impact": 40.0}])


class ScoreGateConfigTests(unittest.TestCase):
    def test_score_gate_gain_steps_returns_sorted_unique_values(self) -> None:
        gate_config = {
            "scoreGate": {
                "profiles": {
                    "score-project": {
                        "potentialGainSteps": [10, 3, 5, 3],
                    }
                }
            }
        }

        steps = MODULE.score_gate_gain_steps(gate_config, "score-project")

        self.assertEqual(steps, [3, 5, 10])


class MainFlowTests(unittest.TestCase):
    def test_changed_mode_reuses_resolved_baseline_for_snapshot_export(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/**/*.go"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {"metrics": ["cyclomatic_complexity"], "threshold": 75, "regression": 5, "newFileThreshold": 75},
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
        }
        head_report = {"files": [make_file("internal/runtime.go", [make_metric("cyclomatic_complexity", 90, "collectFoo()")])]}
        baseline_report = {"files": [make_file("internal/runtime.go", [make_metric("cyclomatic_complexity", 88, "collectFoo()")])]}
        baseline_resolution = MODULE.ChangedBaselineResolution(
            revision="explicit-base",
            compare_revision="HEAD",
            base_ref="main",
            normalized_base_ref="refs/remotes/origin/main",
            fetch_target="main",
            source="explicit-sha",
        )

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            report_path = Path(tmp_dir) / "report.json"
            baseline_snapshot = Path(tmp_dir) / "baseline-snapshot"
            baseline_snapshot.mkdir(parents=True, exist_ok=True)
            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(json.dumps({"defaults": {}, "targets": {"server": {"path": "server", "exclude": []}}}), encoding="utf-8")

            seen_base_refs: list[str | None] = []

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                seen_base_refs.append(base_ref)
                payload = baseline_report if "baseline-reports" in str(output_dir) else head_report
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(payload), encoding="utf-8")
                return path

            with mock.patch.object(MODULE, "resolve_changed_mode_files", return_value=(["server/internal/runtime.go"], baseline_resolution)), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "export_git_snapshot", return_value=baseline_snapshot) as export_snapshot, \
                mock.patch("subprocess.run") as subprocess_run:
                subprocess_run.return_value = mock.Mock(returncode=0, stdout="", stderr="")
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--output-json",
                    str(report_path),
                    "--scopes",
                    "server",
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 0)
            export_snapshot.assert_called_once_with("explicit-base", mock.ANY)
            self.assertEqual(seen_base_refs, [None, "refs/remotes/origin/main"])
            payload = json.loads(report_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["changedBaseline"]["revision"], "explicit-base")
            self.assertEqual(payload["changedBaseline"]["source"], "explicit-sha")

    def test_gate_passes_when_only_documentation_is_low(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "web": {
                    "root": "web/src",
                    "include": ["web/src/**/*.ts"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {"metrics": ["cyclomatic_complexity"], "threshold": 75, "regression": 5, "newFileThreshold": 75},
                "documentation": {"metrics": ["comment_ratio"], "mode": "display-only"},
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0, "documentation": 0.0}},
        }
        report = {
            "files": [
                make_file(
                    "utils/foo.ts",
                    [
                        make_metric("cyclomatic_complexity", 90),
                        make_metric("comment_ratio", 0),
                    ],
                )
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(json.dumps({"defaults": {}, "targets": {"web": {"path": "web/src", "exclude": []}}}), encoding="utf-8")

            report_path = Path(tmp_dir) / "report.json"

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定 scope 写入模拟的 eff-u-code 报告文件。
                
                Parameters:
                	scope (str): 作用域名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置文件路径。
                	base_ref (str | None): 基准引用。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            def fake_run_git(args: list[str]) -> str:
                """
                模拟 `git` 命令的返回值。
                
                Parameters:
                	args (list[str]): 传入的 Git 参数。
                
                Returns:
                	str: 当参数为 `rev-parse HEAD` 时返回的提交哈希。
                
                Raises:
                	RuntimeError: 当参数不匹配预设调用时抛出。
                """
                if args[:2] == ["rev-parse", "HEAD"]:
                    return "head-sha"
                raise RuntimeError("skip base")

            with mock.patch.object(
                MODULE,
                "resolve_changed_mode_files",
                return_value=(
                    ["web/src/utils/foo.ts"],
                    MODULE.ChangedBaselineResolution(None, "HEAD", "", "", "", "local-changes"),
                ),
            ), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "run_git", side_effect=fake_run_git), \
                mock.patch.object(MODULE, "export_git_snapshot", return_value=Path(tmp_dir) / "baseline-snapshot"):
                (Path(tmp_dir) / "baseline-snapshot").mkdir(parents=True, exist_ok=True)
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--output-json",
                    str(report_path),
                    "--scopes",
                    "web",
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 0)
            payload = json.loads(report_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["status"], "pass")

    def test_score_changed_gate_fails_when_scope_score_below_threshold(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/**/*.go"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {"metrics": ["cyclomatic_complexity"], "threshold": 75, "regression": 5, "newFileThreshold": 75},
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
            "scoreGate": {
                "profiles": {
                    "score-changed": {
                        "enabled": True,
                        "enabledScanModes": ["changed"],
                        "threshold": 90,
                        "topContributors": 5,
                        "detailLimit": 3,
                    }
                }
            },
        }
        report = {"files": [make_file("internal/runtime.go", [make_metric("cyclomatic_complexity", 82, "collectFoo()")])]}

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            report_path = Path(tmp_dir) / "report.json"
            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(json.dumps({"defaults": {}, "targets": {"server": {"path": "server", "exclude": []}}}), encoding="utf-8")

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定范围写入模拟的 eff-u-code 报告文件。
                
                Parameters:
                	scope (str): 范围名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置路径。
                	base_ref (str | None): 基准引用。
                
                Returns:
                	Path: 生成的报告文件路径。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            def fake_run_git(args: list[str]) -> str:
                if args[:2] == ["rev-parse", "HEAD"]:
                    return "head-sha"
                raise RuntimeError("skip base")

            with mock.patch.object(
                MODULE,
                "resolve_changed_mode_files",
                return_value=(
                    ["server/internal/runtime.go"],
                    MODULE.ChangedBaselineResolution(None, "HEAD", "", "", "", "local-changes"),
                ),
            ), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "run_git", side_effect=fake_run_git), \
                mock.patch.object(MODULE, "export_git_snapshot", return_value=Path(tmp_dir) / "baseline-snapshot"):
                (Path(tmp_dir) / "baseline-snapshot").mkdir(parents=True, exist_ok=True)
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--gate-profile",
                    "score-changed",
                    "--output-json",
                    str(report_path),
                    "--scopes",
                    "server",
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 1)
            payload = json.loads(report_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["gateProfile"], "score-changed")
            self.assertEqual(payload["status"], "fail")
            self.assertEqual(payload["summary"]["failures"], 1)
            self.assertEqual(payload["summary"]["rawFailures"], 0)
            self.assertEqual(payload["summary"]["scoreGateFailures"], 1)
            self.assertEqual(payload["scopes"]["server"]["scoreGateStatus"], "fail")
            self.assertGreater(len(payload["scopes"]["server"]["topContributors"]), 0)

    def test_score_project_gate_passes_with_project_score_fields(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "web": {
                    "root": "web/src",
                    "include": ["web/src/**/*.ts"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {"metrics": ["cyclomatic_complexity"], "threshold": 75, "regression": 5, "newFileThreshold": 75},
                "duplication": {"metrics": ["code_duplication"], "threshold": 75, "regression": 5, "newFileThreshold": 75},
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 0.5, "duplication": 0.5}},
            "scoreGate": {
                "profiles": {
                    "score-project": {
                        "enabled": True,
                        "enabledScanModes": ["project"],
                        "threshold": 85,
                        "topContributors": 10,
                        "detailLimit": 5,
                        "potentialGainSteps": [3, 5],
                    }
                }
            },
        }
        report = {
            "files": [
                make_file("pages/home.ts", [make_metric("cyclomatic_complexity", 92, "buildHome()"), make_metric("code_duplication", 90, "duplicate block")]),
                make_file("pages/list.ts", [make_metric("cyclomatic_complexity", 88, "buildList()"), make_metric("code_duplication", 87, "duplicate list block")]),
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            output_path = Path(tmp_dir) / "out" / "report.json"
            repo_root = Path(tmp_dir) / "repo"
            first = repo_root / "web" / "src" / "pages" / "home.ts"
            second = repo_root / "web" / "src" / "pages" / "list.ts"
            first.parent.mkdir(parents=True, exist_ok=True)
            second.parent.mkdir(parents=True, exist_ok=True)
            first.write_text("// test\n", encoding="utf-8")
            second.write_text("// test\n", encoding="utf-8")

            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(json.dumps({"defaults": {"top": 20}, "targets": {"web": {"path": "web/src", "exclude": []}}}), encoding="utf-8")

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定范围写入模拟的 eff-u-code 报告文件。
                
                Parameters:
                	scope (str): 范围名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置路径。
                	base_ref (str | None): 基准引用。
                
                Returns:
                	Path: 生成的报告文件路径。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            with mock.patch.object(MODULE, "REPO_ROOT", repo_root), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "list_scope_files_on_disk", return_value=["web/src/pages/home.ts", "web/src/pages/list.ts"]):
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--gate-profile",
                    "score-project",
                    "--scan-mode",
                    "project",
                    "--scopes",
                    "web",
                    "--output-json",
                    str(output_path),
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 0)
            payload = json.loads(output_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["status"], "pass")
            self.assertEqual(payload["gateProfile"], "score-project")
            self.assertIsNotNone(payload["overallQualityScore"])
            self.assertIn("potentialScoreGain", payload["scopes"]["web"])
            self.assertIn("categorySummary", payload["scopes"]["web"])

    def test_score_project_gate_reports_unreported_files_without_blocking_score(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/**/*.go"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {"metrics": ["cyclomatic_complexity"], "threshold": 75, "regression": 5, "newFileThreshold": 75},
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
            "scoreGate": {
                "profiles": {
                    "score-project": {
                        "enabled": True,
                        "enabledScanModes": ["project"],
                        "threshold": 85,
                        "topContributors": 10,
                        "detailLimit": 5,
                    }
                }
            },
        }
        report = {
            "files": [
                make_file("internal/runtime.go", [make_metric("cyclomatic_complexity", 96, "collectFoo()")]),
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            output_path = Path(tmp_dir) / "out" / "report.json"
            repo_root = Path(tmp_dir) / "repo"
            runtime_path = repo_root / "server" / "internal" / "runtime.go"
            extra_path = repo_root / "server" / "internal" / "extra.go"
            runtime_path.parent.mkdir(parents=True, exist_ok=True)
            runtime_path.write_text("package test\n", encoding="utf-8")
            extra_path.write_text("package test\n", encoding="utf-8")

            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(
                json.dumps({"defaults": {"top": 20}, "targets": {"server": {"path": "server", "exclude": []}}}),
                encoding="utf-8",
            )

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定范围写入模拟的 eff-u-code 报告文件。
                
                Parameters:
                	scope (str): 范围名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置路径。
                	base_ref (str | None): 基准引用。
                
                Returns:
                	Path: 生成的报告文件路径。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            with mock.patch.object(MODULE, "REPO_ROOT", repo_root), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(
                    MODULE,
                    "list_scope_files_on_disk",
                    return_value=["server/internal/runtime.go", "server/internal/extra.go"],
                ):
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--gate-profile",
                    "score-project",
                    "--scan-mode",
                    "project",
                    "--scopes",
                    "server",
                    "--output-json",
                    str(output_path),
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 0)
            payload = json.loads(output_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["status"], "pass")
            self.assertEqual(payload["scopes"]["server"]["scoreGateStatus"], "pass")
            self.assertEqual(payload["scopes"]["server"]["coverageStatus"], "pass")
            self.assertEqual(payload["scopes"]["server"]["unreportedFiles"], ["server/internal/extra.go"])

    def test_changed_mode_excluded_files_do_not_count_as_coverage_failures(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/**/*.go"],
                    "exclude": ["server/**/internal/contract/openapi/**"],
                }
            },
            "gateRules": {
                "complexity": {
                    "metrics": ["cyclomatic_complexity"],
                    "threshold": 75,
                    "regression": 5,
                    "newFileThreshold": 75,
                }
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
            "scoreGate": {
                "profiles": {
                    "score-changed": {
                        "enabled": True,
                        "enabledScanModes": ["changed"],
                        "threshold": 90,
                        "topContributors": 5,
                        "detailLimit": 5,
                        "potentialGainSteps": [3],
                        "categoryOrder": ["complexity"],
                    }
                }
            },
        }
        report = {
            "files": [
                make_file(
                    "internal/runtime.go",
                    [
                        make_metric("cyclomatic_complexity", 90),
                    ],
                )
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            output_path = Path(tmp_dir) / "result.json"
            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(
                json.dumps({"defaults": {"top": 20}, "targets": {"server": {"path": "server", "exclude": []}}}),
                encoding="utf-8",
            )

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                写入并返回模拟的 eff-u-code 报告文件路径。
                
                Parameters:
                	scope (str): 作用域名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): eff 配置覆盖文件路径。
                	base_ref (str | None): 基线引用。
                
                Returns:
                	Path: 生成的报告文件路径。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            baseline_resolution = MODULE.ChangedBaselineResolution(
                revision=None,
                compare_revision="HEAD",
                base_ref="",
                normalized_base_ref="",
                fetch_target="",
                source="local-changes",
            )

            with mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(
                    MODULE,
                    "resolve_changed_mode_files",
                    return_value=(
                        [
                            "server/internal/runtime.go",
                            "server/internal/contract/openapi/generated/types.gen.go",
                        ],
                        baseline_resolution,
                    ),
                ):
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--gate-profile",
                    "score-changed",
                    "--scan-mode",
                    "changed",
                    "--scopes",
                    "server",
                    "--output-json",
                    str(output_path),
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 0)
            payload = json.loads(output_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["status"], "pass")
            self.assertEqual(payload["scopes"]["server"]["candidateFiles"], ["server/internal/runtime.go"])
            self.assertEqual(payload["scopes"]["server"]["coverageStatus"], "pass")
            self.assertEqual(payload["scopes"]["server"]["unreportedFiles"], [])

    def test_changed_mode_web_vue_files_outside_eff_u_code_coverage_are_ignored(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "web": {
                    "root": "web/src",
                    "include": ["web/src/**/*.ts", "web/src/**/*.tsx"],
                    "exclude": ["web/src/**/*.test.ts", "web/src/**/*.d.ts"],
                }
            },
            "gateRules": {
                "complexity": {
                    "metrics": ["cyclomatic_complexity"],
                    "threshold": 75,
                    "regression": 5,
                    "newFileThreshold": 75,
                }
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
            "scoreGate": {
                "profiles": {
                    "score-changed": {
                        "enabled": True,
                        "enabledScanModes": ["changed"],
                        "threshold": 90,
                        "topContributors": 5,
                        "detailLimit": 5,
                        "potentialGainSteps": [3],
                        "categoryOrder": ["complexity"],
                    }
                }
            },
        }
        report = {
            "files": [
                make_file(
                    "modules/project/shared/display.ts",
                    [
                        make_metric("cyclomatic_complexity", 90),
                    ],
                )
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            output_path = Path(tmp_dir) / "result.json"
            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(
                json.dumps({"defaults": {"top": 20}, "targets": {"web": {"path": "web/src", "exclude": []}}}),
                encoding="utf-8",
            )

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                写入并返回模拟的 eff-u-code 报告文件路径。
                
                Parameters:
                	scope (str): 作用域名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): eff 配置覆盖文件路径。
                	base_ref (str | None): 基线引用。
                
                Returns:
                	Path: 生成的报告文件路径。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            baseline_resolution = MODULE.ChangedBaselineResolution(
                revision=None,
                compare_revision="HEAD",
                base_ref="",
                normalized_base_ref="",
                fetch_target="",
                source="local-changes",
            )

            with mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(
                    MODULE,
                    "resolve_changed_mode_files",
                    return_value=(
                        [
                            "web/src/modules/project/shared/display.ts",
                            "web/src/modules/project/pages/list/index.vue",
                        ],
                        baseline_resolution,
                    ),
                ):
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--gate-profile",
                    "score-changed",
                    "--scan-mode",
                    "changed",
                    "--scopes",
                    "web",
                    "--output-json",
                    str(output_path),
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 0)
            payload = json.loads(output_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["status"], "pass")
            self.assertEqual(payload["scopes"]["web"]["candidateFiles"], ["web/src/modules/project/shared/display.ts"])
            self.assertEqual(payload["scopes"]["web"]["coverageStatus"], "pass")
            self.assertEqual(payload["scopes"]["web"]["unreportedFiles"], [])

    def test_gate_fails_when_new_file_breaks_complexity_rule(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/**/*.go"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {
                    "metrics": ["cyclomatic_complexity"],
                    "threshold": 75,
                    "regression": 5,
                    "newFileThreshold": 75,
                }
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
        }
        report = {
            "files": [
                make_file(
                    "internal/runtime.go",
                    [
                        make_metric("cyclomatic_complexity", 70),
                    ],
                )
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(json.dumps({"defaults": {}, "targets": {"server": {"path": "server", "exclude": []}}}), encoding="utf-8")

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定 scope 写入模拟的 eff-u-code 报告文件。
                
                Parameters:
                	scope (str): 作用域名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置文件路径。
                	base_ref (str | None): 基准引用。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            def fake_run_git(args: list[str]) -> str:
                """
                模拟 `git` 命令的返回值。
                
                Parameters:
                	args (list[str]): 传入的 Git 参数。
                
                Returns:
                	str: 当参数为 `rev-parse HEAD` 时返回的提交哈希。
                
                Raises:
                	RuntimeError: 当参数不匹配预设调用时抛出。
                """
                if args[:2] == ["rev-parse", "HEAD"]:
                    return "head-sha"
                raise RuntimeError("skip base")

            with mock.patch.object(
                MODULE,
                "resolve_changed_mode_files",
                return_value=(
                    ["server/internal/runtime.go"],
                    MODULE.ChangedBaselineResolution(None, "HEAD", "", "", "", "local-changes"),
                ),
            ), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "run_git", side_effect=fake_run_git), \
                mock.patch.object(MODULE, "export_git_snapshot", return_value=Path(tmp_dir) / "baseline-snapshot"):
                (Path(tmp_dir) / "baseline-snapshot").mkdir(parents=True, exist_ok=True)
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--scopes",
                    "server",
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 1)

    def test_project_mode_evaluates_all_scope_files_without_changed_file_filter(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/**/*.go"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {
                    "metrics": ["cyclomatic_complexity"],
                    "threshold": 75,
                    "regression": 5,
                    "newFileThreshold": 75,
                }
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
        }
        report = {
            "files": [
                make_file("internal/runtime.go", [make_metric("cyclomatic_complexity", 80)]),
                make_file("modules/foo/service.go", [make_metric("cyclomatic_complexity", 70)]),
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            output_path = Path(tmp_dir) / "out" / "report.json"
            repo_root = Path(tmp_dir) / "repo"
            server_root = repo_root / "server"
            runtime_path = server_root / "internal" / "runtime.go"
            service_path = server_root / "modules" / "foo" / "service.go"
            runtime_path.parent.mkdir(parents=True, exist_ok=True)
            service_path.parent.mkdir(parents=True, exist_ok=True)
            for path in (runtime_path, service_path):
                path.write_text("package test\n", encoding="utf-8")

            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(
                json.dumps({"defaults": {"top": 20}, "targets": {"server": {"path": "server", "exclude": []}}}),
                encoding="utf-8",
            )

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定 scope 写入模拟的 eff-u-code 报告文件。
                
                Parameters:
                	scope (str): 作用域名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置文件路径。
                	base_ref (str | None): 基准引用。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            with mock.patch.object(MODULE, "REPO_ROOT", repo_root), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "list_scope_files_on_disk", return_value=["server/internal/runtime.go", "server/modules/foo/service.go"]):
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--scan-mode",
                    "project",
                    "--output-json",
                    str(output_path),
                    "--scopes",
                    "server",
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 1)
            payload = json.loads(output_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["scanMode"], "project")
            self.assertEqual(payload["summary"]["filesEvaluated"], 2)
            self.assertEqual(payload["summary"]["failures"], 1)
            self.assertEqual(payload["summary"]["rawFailures"], 1)
            self.assertEqual(
                [item["path"] for item in payload["scopes"]["server"]["files"]],
                ["server/internal/runtime.go", "server/modules/foo/service.go"],
            )

    def test_project_mode_scopes_are_respected(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/**/*.go"],
                    "exclude": [],
                },
                "web": {
                    "root": "web/src",
                    "include": ["web/src/**/*.ts"],
                    "exclude": [],
                },
            },
            "gateRules": {
                "complexity": {
                    "metrics": ["cyclomatic_complexity"],
                    "threshold": 75,
                    "regression": 5,
                    "newFileThreshold": 75,
                }
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
        }
        reports = {
            "server": {"files": [make_file("internal/runtime.go", [make_metric("cyclomatic_complexity", 80)])]},
            "web": {"files": [make_file("pages/home.ts", [make_metric("cyclomatic_complexity", 80)])]},
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            output_path = Path(tmp_dir) / "report.json"
            repo_root = Path(tmp_dir) / "repo"
            server_file = repo_root / "server" / "internal" / "runtime.go"
            web_file = repo_root / "web" / "src" / "pages" / "home.ts"
            server_file.parent.mkdir(parents=True, exist_ok=True)
            web_file.parent.mkdir(parents=True, exist_ok=True)
            for path in (server_file, web_file):
                path.write_text("// test\n", encoding="utf-8")

            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(
                json.dumps(
                    {
                        "defaults": {"top": 20},
                        "targets": {
                            "server": {"path": "server", "exclude": []},
                            "web": {"path": "web/src", "exclude": []},
                        },
                    }
                ),
                encoding="utf-8",
            )

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定范围写入模拟的 eff-u-code 报告。
                
                Parameters:
                	scope (str): 范围名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置路径。
                	base_ref (str | None): 基线引用。
                
                Returns:
                	Path: 生成的报告文件路径。
                """
                report_path = output_dir / f"eff-u-code-{scope}.json"
                report_path.parent.mkdir(parents=True, exist_ok=True)
                report_path.write_text(json.dumps(reports[scope]), encoding="utf-8")
                return report_path

            with mock.patch.object(MODULE, "REPO_ROOT", repo_root), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "list_scope_files_on_disk", return_value=["web/src/pages/home.ts"]):
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--scan-mode",
                    "project",
                    "--scopes",
                    "web",
                    "--output-json",
                    str(output_path),
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 0)
            payload = json.loads(output_path.read_text(encoding="utf-8"))
            self.assertEqual(sorted(payload["scopes"].keys()), ["web"])

    def test_gate_fails_when_candidate_file_is_unreported(self) -> None:
        gate_config = {
            "changedFiles": {"mode": "git-diff", "diffFilter": "ACMR"},
            "targets": {
                "server": {
                    "root": "server",
                    "include": ["server/*.go", "server/**/*.go"],
                    "exclude": [],
                }
            },
            "gateRules": {
                "complexity": {
                    "metrics": ["cyclomatic_complexity"],
                    "threshold": 75,
                    "regression": 5,
                    "newFileThreshold": 75,
                }
            },
            "curatedScore": {"participatesInGate": False, "weights": {"complexity": 1.0}},
        }
        report = {
            "files": [
                make_file("main.go", [make_metric("cyclomatic_complexity", 85)]),
            ]
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            config_path = Path(tmp_dir) / "gate.json"
            eff_path = Path(tmp_dir) / "eff.json"
            report_path = Path(tmp_dir) / "report.json"
            config_path.write_text(json.dumps(gate_config), encoding="utf-8")
            eff_path.write_text(
                json.dumps({"defaults": {"locale": "zh", "format": "console", "top": 20}, "targets": {"server": {"path": "server", "exclude": []}}}),
                encoding="utf-8",
            )

            def fake_run(scope: str, *, output_dir: Path, eff_config_override: Path | None, base_ref: str | None = None) -> Path:
                """
                为指定 scope 写入模拟的 eff-u-code 报告文件。
                
                Parameters:
                	scope (str): 作用域名称。
                	output_dir (Path): 报告输出目录。
                	eff_config_override (Path | None): 覆盖配置文件路径。
                	base_ref (str | None): 基准引用。
                """
                path = output_dir / f"eff-u-code-{scope}.json"
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(report), encoding="utf-8")
                return path

            def fake_run_git(args: list[str]) -> str:
                """
                模拟 `git` 命令的返回值。
                
                Parameters:
                	args (list[str]): 传入的 Git 参数。
                
                Returns:
                	str: 当参数为 `rev-parse HEAD` 时返回的提交哈希。
                
                Raises:
                	RuntimeError: 当参数不匹配预设调用时抛出。
                """
                if args[:2] == ["rev-parse", "HEAD"]:
                    return "head-sha"
                raise RuntimeError("skip base")

            with mock.patch.object(
                MODULE,
                "resolve_changed_mode_files",
                return_value=(
                    ["server/main.go", "server/extra.go"],
                    MODULE.ChangedBaselineResolution(None, "HEAD", "", "", "", "local-changes"),
                ),
            ), \
                mock.patch.object(MODULE, "run_eff_u_code", side_effect=fake_run), \
                mock.patch.object(MODULE, "run_git", side_effect=fake_run_git), \
                mock.patch.object(MODULE, "export_git_snapshot", return_value=Path(tmp_dir) / "baseline-snapshot"):
                (Path(tmp_dir) / "baseline-snapshot").mkdir(parents=True, exist_ok=True)
                argv = [
                    "evaluate_eff_u_code_gate.py",
                    "--config",
                    str(config_path),
                    "--eff-u-code-config",
                    str(eff_path),
                    "--output-json",
                    str(report_path),
                    "--scopes",
                    "server",
                ]
                with mock.patch.object(sys, "argv", argv):
                    result = MODULE.main()

            self.assertEqual(result, 1)
            payload = json.loads(report_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["status"], "fail")
            self.assertEqual(payload["summary"]["ruleFailures"], 0)
            self.assertEqual(payload["summary"]["coverageFailures"], 1)
            self.assertEqual(payload["scopes"]["server"]["unreportedFiles"], ["server/extra.go"])


if __name__ == "__main__":
    unittest.main()
