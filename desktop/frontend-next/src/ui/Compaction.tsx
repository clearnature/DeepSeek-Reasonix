import { useEffect, useId, useRef, useState } from "react";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";
import type { AgentPort, CompactionSettings } from "../port/port";

const tokens = (n: number) => (n >= 1000 ? `${Math.round(n / 1000)}k` : String(n));

// Two bounds decide when a session folds and only the lower one fires, so the
// number in force is shown beside the field rather than left to be worked out:
// a 1M window against the default soft limit folds at 160k, and a screen that
// showed the setting alone would read as broken.
export function Compaction({ port, onChanged }: { port: AgentPort; onChanged: () => void }) {
  const [box, setBox] = useState<CompactionSettings | null>(null);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const field = useId();

  useEffect(() => {
    port
      .compaction()
      .then((s) => {
        setBox(s);
        setDraft(s.soft_limit_tokens > 0 ? String(s.soft_limit_tokens) : "");
      })
      .catch(() => setBox(null));
  }, [port]);

  if (!box) return <div className="empty">{t("读不到压缩配置。")}</div>;

  const save = async (value: number) => {
    setBusy(true);
    setError("");
    try {
      const saved = await port.saveCompaction(value);
      setBox(saved);
      setDraft(saved.soft_limit_tokens > 0 ? String(saved.soft_limit_tokens) : "");
      onChanged();
    } catch (e) {
      setError(reason(e));
    } finally {
      setBusy(false);
    }
  };

  const sent = useRef<number | null>(null);
  const commit = () => {
    const text = draft.trim();
    // An empty field is the default, which is a value and not a missing one.
    const next = text === "" ? 0 : Number(text);
    if (!Number.isFinite(next) || !Number.isInteger(next)) {
      setError(t("请填一个整数，或者留空用默认值。"));
      return;
    }
    // Enter commits and then blurs, so this runs twice for one keystroke.
    // Declaring the window rebuilds the runtime; doing it twice is not free.
    if (next !== sent.current && next !== box.soft_limit_tokens) {
      sent.current = next;
      void save(next);
    }
  };

  const off = box.soft_limit_tokens < 0;
  // Zero window is the kernel's own "nobody declared one", and it stops
  // maintenance regardless of what either bound says.
  const why = box.context_window === 0
    ? t("这个来源没有声明上下文窗口，所以不会自动压缩。")
    : off
      ? t("经济边界已关闭，只按窗口比例折叠：{n}。", { n: tokens(box.trigger) })
      : box.trigger < box.context_window * box.ratio
        ? t("当前生效的是这个阈值：{n}。", { n: tokens(box.trigger) })
        : t("窗口比例更早到达，当前折叠点是 {n}，这个阈值不会触发。", { n: tokens(box.trigger) });

  return (
    <>
      <div className="lrow">
        <span className="tx">
          <label className="lb" htmlFor={field}>{t("压缩阈值")}</label>
          <span className="ds">
            {t("可见输入到这个大小就折叠，与模型声明的窗口无关。留空使用默认 {n}，填负数关闭。", {
              n: tokens(box.default_soft_limit),
            })}
          </span>
        </span>
        <input
          id={field}
          className="in"
          type="text"
          inputMode="numeric"
          value={draft}
          disabled={busy}
          placeholder={String(box.default_soft_limit)}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          // Enter is the aimed-at commit and carries the identity. Blur still
          // saves — clicking away is an answer too — but a write that only
          // ever hangs off blur has no gesture anyone made on purpose, and a
          // census built on user input cannot see it at all.
          data-action-keydown="compaction.threshold"
          onKeyDown={(e) => {
            if (e.key !== "Enter") return;
            commit();
            e.currentTarget.blur();
          }}
        />
      </div>
      <p className="note">{why}</p>
      {error && <p className="note" data-lvl="warn">{error}</p>}
    </>
  );
}
