# Agent Completion 合并分析文档

## 概述

本文档分析 agent completion (#3)、CLI Elicitation UI (#4)、配置/fetch (#5) 合并所需的文件变更。

**核心结论**：agent/ 包有181个文件与 main-v2 存在差异，其中：
- 56个文件是我们独有的（main-v2 没有）
- 46个文件是上游旧版本（可安全 checkout）
- 79个文件有我们的自定义代码（需逐函数对比合并）

## 已完成的合并

### ✅ #1 MCP Apps 控制器
- control/mcp_apps.go, mcp_interaction.go
- boot.Options.MCPHostProfile
- event.MCPInteractionRequest + MCPInteraction

### ✅ #2 MCP Apps Desktop
- desktop/mcp_apps_app.go, mcp_apps_sandbox.go
- bridge.ts + mcpAppBridge.ts + mcpAppProtocol.ts
- @modelcontextprotocol/ext-apps 1.7.5

### ✅ MCP 2026 Apps 核心
- plugin/ 包完整更新（profile.go, initialize.go, capability_view.go, apps_meta.go, appregistry.go, result.go, protocol_metrics.go, toolresult_projection.go, sdk_legacy_elicitation.go）
- tool/ 包更新（mcp_app.go, remote_dispatch.go, arguments.go）

## 待合并的文件

### A. 可安全 checkout 的上游文件（46个）

```
agent_config.go, argserror_test.go, ask.go, ask_test.go, capability_gate.go,
capability_gate_test.go, capability_list_filter.go, contextual_tool_test.go,
coordinator.go, delivery_hardening_test.go, delivery_visible_final_test.go,
errors.go, execute_batch.go, execute_one.go, extensions.go,
final_readiness_recovery.go, goal_run_boundary.go, interrupted_recovery.go,
loop_e2e_test.go, mcp_concurrency_test.go, mcp_dynamic_tools.go,
progress_guard.go, recovery_gc.go, recovery_gc_test.go, review_report.go,
save.go, services.go, session_display_index.go, session_events_test.go,
standard_todo_continuation.go, storm_breaker.go, storm_test.go,
subagent_progress.go, text_observation.go, title_test.go, tool_call_plan.go,
tool_result_capability.go, turn_phase.go, turnruntime.go, usecapability.go,
usecapability_arguments_test.go, usecapability_list.go, usecapability_mcp_arguments.go,
usecapability_test.go, user_input_test.go, workspace_lease_regression_test.go
```

### B. 我们独有的文件（56个，不需要合并）

```
backgroundize.go, budget.go, calibration.go, compact_prefix_cache_test.go,
compact_ratio_price_test.go, compact_sidecar_fresh_test.go,
compact_sidecar_lossless_test.go, compact_turn_guard.go, empty_session_gc.go,
empty_session_gc_test.go, fork_cache_real_test.go, fork_gate.go,
fork_gate_test.go, plan_approval.go, plan_approval_test.go, skill_fork.go,
steer_job_test.go, steer_throughput_test.go, subagent_fork.go,
subagent_fork_bench_test.go, subagent_fork_test.go, team_ask_gate.go,
team_ask_gate_test.go, team_race_test.go, team_reliability.go,
team_reliability_test.go, team_token_test.go, team_worktree.go,
team_worktree_test.go, teammate_done_e2e_test.go, teammate_ref_test.go,
teammate_store.go, teammate_store_test.go
```

### C. 需要逐函数对比合并的文件（79个）

这些文件有我们的自定义代码，主要是压缩/投影核心：
- compact.go, compact_commit.go, compact_fold_input.go, compact_projection.go
- context_manager.go, context_receipt.go
- preflight.go, projection.go, prune.go
- run_loop.go, session.go, sessionstate.go
- agent.go, task.go
- 等等

## #3 Agent Completion 集成

### 依赖链

```
completion_validation.go → agent.go (Options, Agent struct)
                         → task.go (TaskToolOptions)
                         → turnruntime.go (terminal field, completionPhase)
                         → errors.go (CompletionUncertainError)
                         → run_loop.go (validateCandidateCompletion 调用)
                         → config/completion.go (config.AgentConfig 字段)
                         → completioneval/ 包
                         → event/completion_validation.go
                         → provider/mcp_app.go
                         → capability/audit.go
                         → event/notice_codes.go
                         → evidence/review_report.go
```

### 需要添加的类型/字段

1. **config.AgentConfig**：CompletionValidation, CompletionEvaluatorModel
2. **agent.Options**：CompletionValidation, CompletionEvaluator, CompletionEvaluatorFactory
3. **agent.TaskToolOptions**：CompletionValidation, CompletionEvaluatorFactory
4. **agent.Agent**：completionValidation *completionValidationState
5. **turnruntime.turnRuntime**：terminal terminalProtocolState
6. **tool.Tool**：CallClass, BatchClassifier
7. **新增类型**：CompletionUncertainError, completionPhase, terminalProtocolState, completionValidationState, CompletionEvaluator, CompletionEvaluatorFactory

### 合并策略

**路径A：逐字段添加（推荐）**
1. 在 config.AgentConfig 末尾添加2个字段
2. 在 agent.Options 末尾添加3个字段
3. 在 agent.TaskToolOptions 末尾添加2个字段
4. 在 tool.Tool 接口后添加 CallClass/BatchClassifier
5. 添加新类型定义
6. checkout completioneval/ 包
7. checkout event/completion_validation.go
8. checkout provider/mcp_app.go
9. 更新 run_loop.go 的 terminal 子状态引用
10. 添加 IsHostGeneratedUserMessage 到 preview.go

**风险**：run_loop.go 和 turnruntime.go 有我们的压缩/投影自定义代码，需要逐行对比。

## #4 CLI Elicitation UI

### 依赖链

```
cli/elicit.go → chatTUI.elicit field
             → control.SessionAPI.AnswerMCPInteraction
             → event.MCPInteraction (已添加)
```

### 需要添加的字段

1. **chatTUI**：elicit mcpElicitState
2. **control.SessionAPI**：AnswerMCPInteraction(id string, accepted bool, answer any) error

## #5 配置/fetch 扩展

### 依赖链

```
config/fetch.go → openai.FetchModelsOptions.Proxy
config/completion.go → config.AgentConfig.CompletionValidation
boot/completion_eval.go → agent.CompletionEvaluatorFactory
```

### 需要添加的字段

1. **openai.FetchModelsOptions**：Proxy netclient.ProxySpec
2. **config.AgentConfig**：CompletionValidation, CompletionEvaluatorModel

