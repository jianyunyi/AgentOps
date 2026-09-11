import Link from "next/link";
import type { PolicyPosture as PolicyPostureModel } from "./policy-posture-model";

export type PolicyPostureState =
  | { kind: "loading" }
  | { kind: "error"; onRetry: () => void }
  | { kind: "ready"; posture: PolicyPostureModel };

type PolicyPostureProps = {
  state: PolicyPostureState;
};

export const PolicyPosture = ({ state }: PolicyPostureProps) => {
  return (
    <section className="tower-policy-state" aria-labelledby="policy-posture-heading">
      <p className="eyebrow">Governance configuration</p>
      <div className="tower-panel-heading">
        <h2 id="policy-posture-heading">Policy posture</h2>
        <Link href="/settings/policies">Manage policies</Link>
      </div>

      {state.kind === "loading" ? <p role="status">Reading policy posture</p> : null}

      {state.kind === "error" ? (
        <div className="tower-policy-error" role="alert">
          <p>Policy posture is temporarily unavailable.</p>
          <button type="button" onClick={state.onRetry}>Retry policy posture</button>
        </div>
      ) : null}

      {state.kind === "ready" && !state.posture.active ? <p>No policies configured.</p> : null}

      {state.kind === "ready" && state.posture.active ? (
        <dl>
          <div>
            <dt>Enabled policies</dt>
            <dd>{state.posture.enabledCount} enabled in loaded results</dd>
          </div>
          <div>
            <dt>Active policy</dt>
            <dd className="policy-posture-value" style={{ overflowWrap: "anywhere" }}>
              {state.posture.active.name} · v{state.posture.active.version}
            </dd>
          </div>
          <div>
            <dt>Rules</dt>
            <dd>Rules {state.posture.active.rules_enabled ? "enabled" : "disabled"}</dd>
          </div>
          <div>
            <dt>LLM analysis</dt>
            <dd>LLM analysis {state.posture.active.llm_enabled ? "enabled" : "disabled"}</dd>
          </div>
        </dl>
      ) : null}
    </section>
  );
};
