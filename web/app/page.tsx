import Link from "next/link";
import { AgentNetworkScene } from "../components/landing/agent-network-scene";
import { OverviewSignals } from "../components/landing/overview-signals";

const capabilities = [
  { href: "/dashboard/traces", eyebrow: "Observe", title: "Trace Explorer", description: "Follow every execution chain, token cost, latency, and span." },
  { href: "/dashboard/risk", eyebrow: "Detect", title: "Risk Review", description: "Turn deterministic rules and structured analysis into decisions." },
  { href: "/settings/policies", eyebrow: "Govern", title: "Policy Guardrails", description: "Version the controls that shape safe Agent behavior." },
  { href: "/settings/audit", eyebrow: "Prove", title: "Audit Evidence", description: "Keep immutable records for every sensitive operation." },
];

export default function OverviewPage() {
  return (
    <div className="overview-page">
      <section className="hero-grid overview-hero">
        <div className="hero-copy">
          <span className="eyebrow">AI operations control</span>
          <h1>让每个 Agent 的行为都能被看见</h1>
          <p>从一次调用到完整审计证据，AgentScope 把 AI 运行时变成可观测、可治理、可追责的生产系统。</p>
          <div className="hero-actions"><Link className="button-primary" href="/dashboard/traces">进入 Trace Explorer</Link><Link className="button-quiet" href="/settings/agents">管理 Agents</Link></div>
          <div className="hero-proof"><span>Observe</span><i /> <span>Detect</span><i /> <span>Govern</span><i /> <span>Audit</span></div>
        </div>
        <AgentNetworkScene />
      </section>
      <section className="capability-section" aria-labelledby="capability-title"><div className="section-heading"><div><span className="eyebrow">One operating picture</span><h2 id="capability-title">从信号到证据</h2></div><p>把生产中的 Agent 活动汇聚成团队可以行动的上下文。</p></div><div className="capability-grid">{capabilities.map((item) => <Link className="capability-card" href={item.href} key={item.href}><span className="eyebrow">{item.eyebrow}</span><h3>{item.title}</h3><p>{item.description}</p><span className="capability-arrow" aria-hidden="true">↗</span></Link>)}</div></section>
      <OverviewSignals />
    </div>
  );
}
