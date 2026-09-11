// A press must be answered. A control that lights on hover and does nothing on
// mouse-down leaves the interval between the press and the state change empty,
// and that interval is where the reader is asking whether it was heard.
//
// Membership is read from the running DOM rather than copied into the
// stylesheet as a list of class names, because the list is what rots. Children
// do not count: an svg inside a button inherits the pointer, and pressing it is
// the same press.
import { chromium } from "playwright";

const PAGE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?ws=1&sess=1&turns=4&pref=zh";
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, colorScheme: "dark" });
await page.goto(PAGE, { waitUntil: "networkidle" });
await page.waitForSelector(".compose");

// Open a few panels so the page carries more kinds of control than the
// transcript alone puts on screen.
for (const tab of ["任务", "上下文"]) {
  const el = page.getByRole("tab", { name: tab, exact: true });
  if (await el.count()) {
    await el.first().click();
    await page.waitForTimeout(260);
  }
}

const survey = await page.evaluate(() => {
  const visible = (el) => {
    const s = getComputedStyle(el);
    if (s.visibility === "hidden" || s.display === "none" || s.pointerEvents === "none") return false;
    const b = el.getBoundingClientRect();
    return b.width > 2 && b.height > 2;
  };
  // Of the elements taking a pointer, the ones that are controls themselves:
  // a semantic element or an interactive role, not a descendant of another.
  const ROLES = ["button", "tab", "option", "menuitem", "menuitemcheckbox", "switch", "checkbox", "radio", "treeitem", "link"];
  const isControl = (el) => {
    const tag = el.tagName.toLowerCase();
    if (["button", "summary", "select"].includes(tag) || (tag === "a" && el.hasAttribute("href"))) return true;
    return ROLES.includes(el.getAttribute("role") ?? "");
  };
  const all = [...document.querySelectorAll("*")].filter((el) => getComputedStyle(el).cursor === "pointer" && visible(el));
  // A disabled control owes no pressed state.
  const live = (el) => !el.disabled && el.getAttribute("aria-disabled") !== "true";
  const controls = all.filter((el) => isControl(el) && live(el) && !all.some((o) => o !== el && isControl(o) && o.contains(el)));
  const orphans = all.filter((el) => !isControl(el) && !controls.some((c) => c.contains(el)));

  // Who each :active rule would match with the pseudo-class removed. Asking
  // this instead of pressing fires no click and reads no incidental state.
  const pressed = [];
  for (const sheet of document.styleSheets) {
    let rules;
    try {
      rules = sheet.cssRules;
    } catch {
      continue;
    }
    const walk = (list) => {
      for (const r of list) {
        if (r.cssRules) walk(r.cssRules);
        if (!r.selectorText || !r.selectorText.includes(":active")) continue;
        // Split on commas at depth zero: the comma inside :is(a, b) does not
        // separate selectors, and splitting on it shreds one rule into
        // fragments that match nothing — which this guard then reports as an
        // absent rule.
        const parts = [];
        let depth = 0;
        let cur = "";
        for (const ch of r.selectorText) {
          if (ch === "(") depth++;
          else if (ch === ")") depth--;
          if (ch === "," && depth === 0) {
            parts.push(cur);
            cur = "";
          } else cur += ch;
        }
        parts.push(cur);
        for (const sel of parts) {
          if (!sel.includes(":active")) continue;
          const bare = sel.replace(/:active/g, "").trim();
          if (bare) pressed.push(bare);
        }
      }
    };
    walk(rules);
  }
  const answers = (el) => pressed.some((sel) => {
    try {
      return el.matches(sel) || el.querySelector(sel) !== null;
    } catch {
      return false;
    }
  });

  const name = (el) => {
    const cls = typeof el.className === "string" ? el.className.split(/\s+/).filter(Boolean)[0] : "";
    const role = el.getAttribute("role");
    return `${el.tagName.toLowerCase()}${role ? `[role=${role}]` : ""}${cls ? "." + cls : ""}`;
  };
  const tally = (els) => {
    const m = new Map();
    for (const el of els) m.set(name(el), (m.get(name(el)) ?? 0) + 1);
    return [...m].sort((a, b) => b[1] - a[1]);
  };
  return {
    controls: controls.length,
    silent: tally(controls.filter((el) => !answers(el))),
    orphans: tally(orphans),
    pressedRules: pressed.length,
  };
});

console.log(`接受指针的控件 ${survey.controls} 个   带 :active 的规则 ${survey.pressedRules} 条`);
if (!survey.controls || !survey.pressedRules) {
  console.log("\n没有扫到控件或规则：这份守卫会一直通过，请先确认页面已经就绪。");
  process.exit(1);
}

const silentTotal = survey.silent.reduce((a, [, n]) => a + n, 0);
if (silentTotal) {
  console.log(`\n按下后没有任何反馈的 ${silentTotal} 个（${survey.silent.length} 种）：`);
  for (const [id, n] of survey.silent) console.log(`  ${String(n).padStart(3)}  ${id}`);
}
check("每个接受指针的控件都对按下有反馈", silentTotal === 0, silentTotal ? `${silentTotal} 个缺少` : "");

if (survey.orphans.length) {
  console.log(`\n接受指针但既非语义元素、也没有 role 的 ${survey.orphans.length} 种（键盘与读屏无法到达）：`);
  for (const [id, n] of survey.orphans) console.log(`  ${String(n).padStart(3)}  ${id}`);
}
check("接受指针的元素都是控件", survey.orphans.length === 0, survey.orphans.length ? `${survey.orphans.length} 种不是控件` : "");

await browser.close();
console.log(fails.length ? `\n${fails.length} 项未通过` : "\n全部通过");
process.exit(fails.length ? 1 : 0);
