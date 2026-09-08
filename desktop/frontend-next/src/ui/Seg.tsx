// 一段带标签的互斥选择：树里已有的 .seg 槽，外面套上字段名和「选中这档意味着
// 什么」那一行，再加上它自己的等待与拒绝。
//
// 和 Switch 一样是个基元：它自己不是入口，动作的身份写在把 onClick 交给它的那
// 个调用点上，data-* 原样透传到每一颗按钮，答案落在 data-value 上。
export interface SegOption {
  value: string;
  // Already in the reader's language: the call site owns the wording, because
  // that is the only place a catalogue check can see it.
  label: string;
  // What picking it means, where there is something to say. A rung on a ladder
  // explains itself by its place on it.
  note?: string;
}

export function Seg({
  label, options, current, asking, refused, text, onClick, ...id
}: {
  label: string;
  options: SegOption[];
  current?: string;
  // The one value asked for and not yet answered. While it is out this group
  // takes no second answer — a set of one-of-N buttons has no meaning for two
  // in flight — and every other group beside it stays live.
  asking?: string;
  // Why the last answer did not land. It sits with the group it was about and
  // does not lock it: picking again is the only thing to do here.
  refused?: string;
  // 标签是中文：mono 栈里汉字会掉到回退字体，跟旁边的界面字对不齐。
  text?: boolean;
  // Named for the DOM event it stands for, as Switch is. The action census
  // places a primitive's call site by that name; a prop the DOM has no event
  // for makes the call site invisible to it, which is why every Picker in the
  // tree is uncounted today.
  onClick: (value: string) => void;
} & { [K in `data-${string}`]?: string }) {
  const note = options.find((o) => o.value === current)?.note;
  return (
    <div className="segf">
      <div className="seglb">{label}</div>
      <div className="seg" data-text={text ? "" : undefined} role="group" aria-label={label}>
        {options.map((o) => (
          <button
            key={o.value}
            {...id}
            data-value={o.value}
            aria-pressed={current === o.value}
            data-asking={asking === o.value ? "" : undefined}
            disabled={!!asking}
            onClick={() => onClick(o.value)}
          >
            {o.label}
          </button>
        ))}
      </div>
      {refused ? <p className="note segbad" role="alert">{refused}</p> : note ? <p className="note">{note}</p> : null}
    </div>
  );
}
