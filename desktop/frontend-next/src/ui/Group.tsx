import { t } from "../i18n";
import { SETTING_AT } from "./prefsnav";
import type { ReactNode } from "react";

// id is required, and that is the whole enforcement: a settings block cannot
// be written without naming itself, and a name the catalogue does not carry
// fails the gate. So "which setting is this, and when does changing it take
// effect" has no default and no way to be left unanswered.
export function Group({
  id, title, hint, now, action, children,
}: {
  id: string; title: string; hint?: string; now?: string; action?: ReactNode; children?: ReactNode;
}) {
  return (
    <section className="grp" id={`set-${id}`} data-setting={id}>
      <div className="grp-hd">
        <h2>{title}</h2>
        {now && <span className="now">{now}</span>}
        {action}
      </div>
      {hint && <p className="hint">{hint}</p>}
      {children && <div className="grp-items">{children}</div>}
      {/* A fact about this block, at the weight of a fact: what it costs to
          change is not a warning, and three of them on a page should not read
          as three alarms. */}
      <ApplyNote id={id} />
    </section>
  );
}

/** What it costs to change what is in this block. Separate from Group so a
 *  block that draws its own frame still cannot go without one — the appearance
 *  page builds its sections by hand, and the question is the same there. */
export function ApplyNote({ id }: { id: string }) {
  const apply = SETTING_AT(id)?.apply;
  if (!apply || apply === "none") return null;
  return <p className="apply" data-apply={apply}>{t(APPLY_SAID[apply])}</p>;
}

// 「已保存」和「生效」是两件事，重启那一档必须把它们分开说 —— 否则关掉设置的人
// 不知道刚填的东西还在不在。

// 「已保存」和「生效」是两件事，重启那一档必须把它们分开说 —— 否则关掉设置的人
// 不知道刚填的东西还在不在。
const APPLY_SAID: Record<"immediate" | "runtime-rebuild" | "restart", string> = {
  immediate: "立即生效",
  "runtime-rebuild": "重建运行时",
  restart: "已保存 · 重启后生效",
};
