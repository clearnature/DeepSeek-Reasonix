import { lazy, Suspense } from "react";

const Markdown = lazy(async () => ({ default: (await import("./Markdown")).Markdown }));

export function LazyMarkdown({ text, streaming }: { text: string; streaming?: boolean }) {
  return (
    <Suspense fallback={<div className="md" style={{ whiteSpace: "pre-wrap" }}>{text}</div>}>
      <Markdown text={text} streaming={streaming} />
    </Suspense>
  );
}
