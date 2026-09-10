import type { Tool } from "../../port/wire";
import { t } from "../../i18n";

export function toolFailed(tool: Tool): boolean {
  if (tool.err) return true;
  const execution = tool.execution;
  return !!execution && ((!!execution.state && execution.state !== "completed") || (execution.exitCode ?? 0) !== 0);
}

export function toolFailureLabel(tool: Tool): string {
  const execution = tool.execution;
  if ((execution?.exitCode ?? 0) !== 0) return `exit ${execution?.exitCode}`;
  if (execution?.state && execution.state !== "completed") return execution.state;
  return t("失败");
}
