import type { Guardian } from "../../port/wire";
import { Sym } from "../Sym";
import { t } from "../../i18n";

export function GuardianCard({ g }: { g: Guardian }) {
  const risk = g.risk_level ?? "unknown";
  const riskLabel = risk === "low" ? t("低风险") : risk === "medium" ? t("中风险") : risk === "high" ? t("高风险") : t("风险未知");
  return (
    <div className="call">
      <div className="g">
        <Sym glyph="⊛" />
        <span className="line" />
      </div>
      <div className="c">
        <div className="hl">
          <span className="nm">Guardian</span>
          <span className="tag">guardian_assessment</span>
          <span className="arg">{g.subject}</span>
        </div>
        <div className="out">
          <div className="guard" data-risk={risk}>
            <div className="guard-hd">
              <span className="verdict">{g.outcome}</span>
              <span className="risk">{riskLabel}</span>
              <span className="gauge" aria-label={riskLabel}>
                <i />
                <i />
                <i />
              </span>
            </div>
            {g.rationale && <div className="guard-why">{g.rationale}</div>}
          </div>
        </div>
      </div>
    </div>
  );
}
