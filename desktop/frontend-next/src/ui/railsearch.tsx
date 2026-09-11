import { type ReactNode, createContext, useContext, useEffect, useRef, useState } from "react";
import { t } from "../i18n";
import { chord } from "./keys";

// 栏里现在是一份机器的列表，搜索框问的是整份列表。它自己拥有这个词，两个列表
// 都读它 —— 把状态搬进 App 会让「栏顶那个框」和「谁在用它」分成两处维护，而
// 让本机那半边持有它，远程那半边就只能从旁边要。
const RailQuery = createContext("");

/** 此刻栏里在找的词，已去掉首尾空白；空串表示没有在找。 */
export const useRailQuery = () => useContext(RailQuery).trim().toLowerCase();

export function RailSearch({ children }: { children: ReactNode }) {
  const [q, setQ] = useState("");
  const find = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key !== "k" || !(ev.metaKey || ev.ctrlKey)) return;
      ev.preventDefault();
      find.current?.focus();
      find.current?.select();
    };
    addEventListener("keydown", onKey);
    return () => removeEventListener("keydown", onKey);
  }, []);

  return (
    <>
      <div className="wsfind">
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M7.2 3.1a4.1 4.1 0 1 0 0 8.2 4.1 4.1 0 0 0 0-8.2M10.3 10.3 13 13" />
        </svg>
        <input
          ref={find}
          type="search"
          value={q}
          onChange={(ev) => setQ(ev.target.value)}
          onKeyDown={(ev) => ev.key === "Escape" && setQ("")}
          placeholder={t("搜索会话 / 项目")}
          aria-label={t("搜索会话 / 项目")}
        />
        <kbd hidden={!!q}>{chord("K")}</kbd>
        <button className="clr" hidden={!q} onClick={() => setQ("")} aria-label={t("清空")}>
          ×
        </button>
      </div>
      <RailQuery.Provider value={q}>{children}</RailQuery.Provider>
    </>
  );
}
