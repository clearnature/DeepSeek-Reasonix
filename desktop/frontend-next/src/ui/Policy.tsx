import { useRef, useState } from "react";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";
import type { AgentPort, ApprovalMode, Preset, SessionStatus } from "../port/port";
import { useDismiss } from "./dismiss";
import { Seg } from "./Seg";

// 这一轮怎么跑，是三个变量共同回答的一个问题。它们原来分处窗口顶栏（执行方式）
// 和输入框底栏（强度、批准），要知道下一轮会怎么执行得看两个地方。
const PRESETS: [Preset, string, string][] = [
  ["balanced", "均衡", "做到模型认为做完为止。日常用这档"],
  ["delivery", "交付", "改了东西就得验证、复核、签收，少一样都不算做完"],
];

// 从紧到松排，和闸门环的缺口一个方向。「不打扰」在内核是 permission.Deny：
// 它比「询问」更严，不是更松 —— 排在询问前面才不会被读反。
const APPROVALS: [ApprovalMode, string, string][] = [
  ["dontAsk", "不打扰", "不弹审批；要批准才能做的一概不做。"],
  ["ask", "询问", "每次动手前问你。"],
  ["auto", "自动", "低风险自己过，写操作仍然问。"],
  ["yolo", "全放行", "不问了。只在你完全信任这个工作区时用。"],
];

type Field = "preset" | "effort" | "approval";

interface Props {
  port: AgentPort;
  status: SessionStatus | null;
  // The rungs this model actually has, decided upstream from the kernel's own
  // list. An empty ladder means the endpoint names none, and the field is not
  // drawn rather than drawn dead.
  efforts: string[];
  onChanged: () => void;
}

// 内核是三个独立的 effect，界面把它们画在一起不该把它们并成一个调用 —— 那会让
// 「强度没换成」连坐掉已经换好的执行方式。
export function Policy({ port, status, efforts, onChanged }: Props) {
  const [open, setOpen] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);
  const btn = useRef<HTMLButtonElement>(null);
  useDismiss(open, wrap, () => setOpen(false));

  // 每个字段各自记「问出去的是哪一档」和「上一次为什么被拒」。合成一个就回到了
  // 整排一起变灰：一个字段在等，另外两个仍然能改。
  const [asking, setAsking] = useState<Partial<Record<Field, string>>>({});
  const [refused, setRefused] = useState<Partial<Record<Field, string>>>({});

  const apply = (field: Field, value: string, call: () => Promise<void>) => {
    if (asking[field]) return;
    setAsking((a) => ({ ...a, [field]: value }));
    setRefused((r) => ({ ...r, [field]: "" }));
    call()
      .then(onChanged)
      .catch((e: unknown) => setRefused((r) => ({ ...r, [field]: reason(e) })))
      .finally(() => setAsking((a) => ({ ...a, [field]: "" })));
  };

  const preset = status?.preset;
  const eff = status?.effort || "auto";
  const apv = status?.toolApprovalMode ?? "ask";
  // 事实本身，不是「设置」。摘要念的是内核当前的三个值，档位一律用内核自己的
  // 拼法 —— 给它另配一张显示名表，就是又一处能跟内核走散的措辞。
  const summary = [t(PRESETS.find(([id]) => id === preset)?.[1] ?? "") || "—", eff, t(APPROVALS.find(([m]) => m === apv)?.[1] ?? "")]
    .filter(Boolean)
    .join(" · ");

  return (
    <div className="picker policy" ref={wrap}>
      <button
        ref={btn}
        className="mode plain"
        data-action="chrome.policy"
        aria-expanded={open}
        title={t("这一轮怎么跑：执行方式、思考强度、工具权限")}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="vl">{summary}</span>
      </button>
      <div className="menu modemenu polmenu" role="group" aria-label={t("本轮执行策略")} hidden={!open}>
        <Seg
          label={t("执行方式")}
          data-action="chrome.preset"
          text
          options={PRESETS.map(([value, lb, note]) => ({ value, label: t(lb), note: t(note) }))}
          current={preset}
          asking={asking.preset}
          refused={refused.preset}
          onClick={(v) => apply("preset", v, () => port.setPreset(v as Preset))}
        />
        {/* An endpoint that names no levels gets no ladder: a control whose
            every rung is refused downstream reads as a dead one. */}
        {efforts.length > 0 && (
          <Seg
            label={t("思考强度")}
            data-action="reasoning.effort"
            options={efforts.map((value) => ({ value, label: value }))}
            current={eff}
            asking={asking.effort}
            refused={refused.effort}
            onClick={(v) => apply("effort", v, () => port.setEffort(v))}
          />
        )}
        <Seg
          label={t("工具权限")}
          data-action="tool-approval.mode"
          text
          options={APPROVALS.map(([value, lb, note]) => ({ value, label: t(lb), note: t(note) }))}
          current={apv}
          asking={asking.approval}
          refused={refused.approval}
          onClick={(v) => apply("approval", v, () => port.setApprovalMode(v as ApprovalMode))}
        />
      </div>
    </div>
  );
}
