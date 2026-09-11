// 按下要有回答。悬停会亮、按下什么都不发生的控件，手感是「软」的：从按下去到
// 状态变化之间那一段是空的，而那一段正是人在问「它收到了吗」。
//
// 成员资格从真实 DOM 读，不在样式表里手抄一份类名 —— 抄来的名单会腐烂，而
// 这份问的是「屏幕上此刻每个吃指针的控件」。子元素不算：button 里的 svg 继承了
// pointer，按在它上面和按在按钮上是同一次按下。
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

// 打开几处常驻面板，让这一页上的控件种类多一些。
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
  // 吃指针的元素里，自己是控件的那些：语义元素或带交互 role，且不是另一个
  // 控件的后代。
  const ROLES = ["button", "tab", "option", "menuitem", "menuitemcheckbox", "switch", "checkbox", "radio", "treeitem", "link"];
  const isControl = (el) => {
    const tag = el.tagName.toLowerCase();
    if (["button", "summary", "select"].includes(tag) || (tag === "a" && el.hasAttribute("href"))) return true;
    return ROLES.includes(el.getAttribute("role") ?? "");
  };
  const all = [...document.querySelectorAll("*")].filter((el) => getComputedStyle(el).cursor === "pointer" && visible(el));
  // 禁用的控件本来就该没有回答，它不欠一个按下态。
  const live = (el) => !el.disabled && el.getAttribute("aria-disabled") !== "true";
  const controls = all.filter((el) => isControl(el) && live(el) && !all.some((o) => o !== el && isControl(o) && o.contains(el)));
  const orphans = all.filter((el) => !isControl(el) && !controls.some((c) => c.contains(el)));

  // 每条带 :active 的规则，去掉 :active 之后还能匹配谁 —— 不用真按下去问，
  // 既不触发点击，也不看运行时的偶然状态。
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
        // 逗号按括号深度切：:is(a, b) 里的那个逗号不是选择器之间的逗号，照字面
        // 切会把一条规则碎成几个无效选择器，然后这份守卫报告它不存在。
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

console.log(`吃指针的控件 ${survey.controls} 个   带 :active 的规则 ${survey.pressedRules} 条`);
if (!survey.controls || !survey.pressedRules) {
  console.log("\n没扫到控件或没扫到规则：这份守卫会永远绿着，先确认页面起来了。");
  process.exit(1);
}

const silentTotal = survey.silent.reduce((a, [, n]) => a + n, 0);
if (silentTotal) {
  console.log(`\n按下去没有任何回答的 ${silentTotal} 个（${survey.silent.length} 种）：`);
  for (const [id, n] of survey.silent) console.log(`  ${String(n).padStart(3)}  ${id}`);
}
check("每个吃指针的控件都对按下有回答", silentTotal === 0, silentTotal ? `${silentTotal} 个没有` : "");

if (survey.orphans.length) {
  console.log(`\n吃指针但既不是语义元素也没有 role 的 ${survey.orphans.length} 种（键盘和读屏都到不了）：`);
  for (const [id, n] of survey.orphans) console.log(`  ${String(n).padStart(3)}  ${id}`);
}
check("吃指针的东西都是控件", survey.orphans.length === 0, survey.orphans.length ? `${survey.orphans.length} 种不是` : "");

await browser.close();
console.log(fails.length ? `\n${fails.length} 项不过` : "\n全过");
process.exit(fails.length ? 1 : 0);
