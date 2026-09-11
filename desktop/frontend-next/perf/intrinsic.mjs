// 转录用 content-visibility 跳过屏外卡片的排版，屏外那些按一个估算高度占位。
// 估算本身不必准，但它的误差必须是双向的：低于最矮的卡片时，每一张没读过的卡
// 片都被低估，误差同向累加 —— 滚动落点永远偏短，落下去再纠正一次，那一下就是
// 回包时看到的抽动。这份守卫量真实分布，再拿它对账样式表里的那个数。
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const HERE = dirname(fileURLToPath(import.meta.url));
const SRC = process.env.PERF_SRC ?? join(HERE, "..", "src");
const PAGE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?ws=1&sess=1&turns=40&pref=zh";

const css = readFileSync(join(SRC, "styles", "app.css"), "utf8");
const declared = Number(css.match(/contain-intrinsic-size:\s*auto\s+(\d+(?:\.\d+)?)px/)?.[1] ?? NaN);
if (!Number.isFinite(declared)) {
  console.log("没在样式表里找到 contain-intrinsic-size 的兜底值，这份守卫会一直通过。");
  process.exit(1);
}

const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, colorScheme: "dark" });
await page.goto(PAGE, { waitUntil: "networkidle" });
await page.waitForSelector(".compose");
await page.waitForTimeout(1000);

// 把整条转录走一遍，每张卡片至少排版一次，量到的才是读者会遇到的分布。
const stats = await page.evaluate(async () => {
  const sc = [...document.querySelectorAll("*")]
    .filter((el) => {
      const s = getComputedStyle(el);
      return /auto|scroll/.test(s.overflowY) && el.scrollHeight > el.clientHeight + 4 && el.querySelector(".call");
    })
    .pop();
  if (sc) {
    for (let y = 0; y <= sc.scrollHeight; y += 400) {
      sc.scrollTop = y;
      await new Promise((r) => requestAnimationFrame(r));
    }
  }
  const hs = [...document.querySelectorAll(".chunk > .enterbox > .call")]
    .map((c) => c.offsetHeight)
    .filter((x) => x > 0)
    .sort((a, b) => a - b);
  if (!hs.length) return null;
  const q = (p) => hs[Math.floor((hs.length - 1) * p)];
  return {
    n: hs.length,
    min: hs[0],
    median: q(0.5),
    mean: Math.round(hs.reduce((a, b) => a + b, 0) / hs.length),
    max: hs[hs.length - 1],
  };
});

if (!stats) {
  console.log("没量到卡片：这份守卫会一直通过，先确认页面上有转录。");
  process.exit(1);
}

console.log(`样式表里的兜底 ${declared}px   实测 n=${stats.n} 最矮 ${stats.min} 中位 ${stats.median} 均值 ${stats.mean} 最高 ${stats.max}`);
check(
  "占位估算不低于最矮的卡片",
  declared >= stats.min,
  declared < stats.min ? `${declared} < ${stats.min}，每张未读卡片都被低估` : "",
);
check(
  "占位估算落在真实分布之内",
  declared <= stats.max,
  declared > stats.max ? `${declared} > ${stats.max}，误差反向累加` : "",
);

await browser.close();
console.log(fails.length ? `\n${fails.length} 项未通过` : "\n全部通过");
process.exit(fails.length ? 1 : 0);
