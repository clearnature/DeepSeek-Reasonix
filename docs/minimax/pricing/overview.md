# pricing/overview.md

> 来源: https://platform.minimaxi.com/docs/pricing/overview.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# 产品定价

> MiniMax 提供两类计价方案，按使用场景选择。

MiniMax 按 **「如何付费」** 将定价划分为两类:按调用实时结算的 **API 价格**,与月度固定额度的 **订阅套餐**。两类各自对应独立的 Key 体系,从下方入口进入具体定价文档。

<div className="grid grid-cols-1 md:grid-cols-2 gap-4 my-6 not-prose">
  <div className="rounded-xl border border-gray-200 dark:border-gray-700 p-6 bg-white dark:bg-zinc-900/40 flex flex-col">
    <h3 className="text-lg font-semibold mt-0 mb-1.5">
      <a href="https://platform.minimaxi.com/docs/guides/pricing-paygo" className="text-gray-900 dark:text-white no-underline hover:text-blue-600 dark:hover:text-blue-400">API 价格</a>
    </h3>

    <p className="text-sm text-gray-600 dark:text-gray-400 mt-0 mb-5">面向企业。</p>

    <div className="rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden mt-auto">
      <a href="https://platform.minimaxi.com/docs/guides/pricing-paygo" className="group flex items-center justify-between gap-3 px-4 py-3 border-b border-gray-200 dark:border-gray-700 no-underline hover:bg-gray-50 dark:hover:bg-gray-800/40">
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-gray-900 dark:text-gray-200">按量计费</div>
          <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">按 token / 调用次数实时结算</div>
        </div>

        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-400 dark:text-gray-500 shrink-0 transition-transform group-hover:translate-x-0.5">
          <path d="M5 12h14" />

          <path d="m12 5 7 7-7 7" />
        </svg>
      </a>

      <a href="https://platform.minimaxi.com/docs/guides/pricing-speech" className="group flex items-center justify-between gap-3 px-4 py-3 border-b border-gray-200 dark:border-gray-700 no-underline hover:bg-gray-50 dark:hover:bg-gray-800/40">
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-gray-900 dark:text-gray-200">语音资源包</div>
          <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">HD / Turbo 预付,单价更低</div>
        </div>

        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-400 dark:text-gray-500 shrink-0 transition-transform group-hover:translate-x-0.5">
          <path d="M5 12h14" />

          <path d="m12 5 7 7-7 7" />
        </svg>
      </a>

      <a href="https://platform.minimaxi.com/docs/guides/pricing-video" className="group flex items-center justify-between gap-3 px-4 py-3 no-underline hover:bg-gray-50 dark:hover:bg-gray-800/40">
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-gray-900 dark:text-gray-200">视频资源包</div>
          <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Hailuo 系列预付,单价更低</div>
        </div>

        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-400 dark:text-gray-500 shrink-0 transition-transform group-hover:translate-x-0.5">
          <path d="M5 12h14" />

          <path d="m12 5 7 7-7 7" />
        </svg>
      </a>
    </div>
  </div>

  <div className="rounded-xl border border-gray-200 dark:border-gray-700 p-6 bg-white dark:bg-zinc-900/40 flex flex-col">
    <h3 className="text-lg font-semibold mt-0 mb-1.5">
      <a href="https://platform.minimaxi.com/docs/guides/pricing-token-plan" className="text-gray-900 dark:text-white no-underline hover:text-blue-600 dark:hover:text-blue-400">订阅套餐</a>
    </h3>

    <p className="text-sm text-gray-600 dark:text-gray-400 mt-0 mb-5">面向个人与小团队。</p>

    <div className="flex-1 flex items-center">
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden w-full">
        <a href="https://platform.minimaxi.com/docs/guides/pricing-token-plan" className="group flex items-center justify-between gap-3 px-4 py-3 border-b border-gray-200 dark:border-gray-700 no-underline hover:bg-gray-50 dark:hover:bg-gray-800/40">
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-gray-900 dark:text-gray-200">Token Plan</div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">月度订阅,配额每月重置</div>
          </div>

          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-400 dark:text-gray-500 shrink-0 transition-transform group-hover:translate-x-0.5">
            <path d="M5 12h14" />

            <path d="m12 5 7 7-7 7" />
          </svg>
        </a>

        <a href="https://platform.minimaxi.com/docs/guides/pricing-token-plan-team" className="group flex items-center justify-between gap-3 px-4 py-3 no-underline hover:bg-gray-50 dark:hover:bg-gray-800/40">
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-gray-900 dark:text-gray-200">Token Plan 团队版</div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">席位分配和团队积分池规则</div>
          </div>

          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-400 dark:text-gray-500 shrink-0 transition-transform group-hover:translate-x-0.5">
            <path d="M5 12h14" />

            <path d="m12 5 7 7-7 7" />
          </svg>
        </a>
      </div>
    </div>
  </div>
</div>
