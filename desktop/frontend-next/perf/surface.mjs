// 一个盒子要和它坐着的底分开，只有三种说法：描一条线、垫一层底色、抬一层影子。
// 三种都没有的盒子读起来是糊的；三种一起上的盒子读起来像表格。这份守卫问的是
// 每个圆角盒子选了哪一种，以及选了底色的那些，这一层到底差得够不够看得见。
//
// 差多少算够从人眼算起，不从设计稿抄：明度差低于 2%（OKLab L）在深色底上基本
// 看不出，那就不是一层，是一个没生效的决定。
import { chromium } from "playwright";

const PAGE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?ws=1&sess=1&turns=4&pref=zh";
const MIN_L = Number(process.env.PERF_MIN_L ?? 2);
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();

for (const scheme of ["dark", "light"]) {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, colorScheme: scheme });
  await page.goto(PAGE, { waitUntil: "networkidle" });
  await page.waitForSelector(".compose");
  await page.waitForTimeout(300);

  const found = await page.evaluate((minL) => {
    // sRGB -> OKLab L，只要明度这一项：分层靠的是明暗，不是色相。
    const lum = (rgb) => {
      const m = rgb.match(/[\d.]+/g);
      if (!m) return null;
      const [r, g, b] = m.slice(0, 3).map((v) => {
        const c = Number(v) / 255;
        return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
      });
      const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
      const m2 = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
      const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
      return (0.2104542553 * l + 0.793617785 * m2 - 0.0040720468 * s) * 100;
    };
    const opaque = (c) => c && c !== "transparent" && !/rgba\([^)]*,\s*0\s*\)/.test(c);

    const thin = [];
    for (const el of document.querySelectorAll("*")) {
      const s = getComputedStyle(el);
      if (s.display === "none" || s.visibility === "hidden") continue;
      const box = el.getBoundingClientRect();
      // 只看真的被画成盒子的东西：有圆角、有面积。
      if (box.width < 40 || box.height < 24) continue;
      if (parseFloat(s.borderTopLeftRadius) < 3) continue;
      if (!opaque(s.backgroundColor)) continue;
      const hasBorder = parseFloat(s.borderTopWidth) > 0 && opaque(s.borderTopColor);
      const hasShadow = s.boxShadow !== "none" && !/inset/.test(s.boxShadow.replace(/inset/g, "inset"))
        ? true
        : s.boxShadow !== "none" && /(^|,)\s*(?!inset)[^,]*\d/.test(s.boxShadow);
      if (hasBorder || hasShadow) continue;

      // 它坐在谁身上：最近的一个自己有底色的祖先。
      let p = el.parentElement;
      while (p && !opaque(getComputedStyle(p).backgroundColor)) p = p.parentElement;
      if (!p) continue;
      const a = lum(s.backgroundColor);
      const b = lum(getComputedStyle(p).backgroundColor);
      if (a === null || b === null) continue;
      const d = Math.abs(a - b);
      // 完全相同的底色是「不分层」，是个决定；差一点点才是「想分层但没生效」。
      if (d > 0.05 && d < minL) {
        const cls = typeof el.className === "string" ? el.className.split(/\s+/).filter(Boolean).join(".") : "";
        thin.push(`${el.tagName.toLowerCase()}${cls ? "." + cls : ""}  ΔL=${d.toFixed(1)}`);
      }
    }
    return [...new Set(thin)];
  }, MIN_L);

  if (found.length) {
    console.log(`\n${scheme}：只靠底色分层、但这一层看不出来的 ${found.length} 种：`);
    for (const f of found) console.log("  " + f);
  }
  check(`${scheme}：用底色分层的盒子，这一层真的看得见`, found.length === 0, found.length ? `${found.length} 种低于 ΔL ${MIN_L}` : `阈值 ΔL ${MIN_L}`);
  await page.close();
}

await browser.close();
console.log(fails.length ? `\n${fails.length} 项不过` : "\n全过");
process.exit(fails.length ? 1 : 0);
