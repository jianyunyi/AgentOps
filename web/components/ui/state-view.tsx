import type { ReactNode } from "react";

export type StateKind = "loading" | "empty" | "error" | "forbidden";

type StateViewProps = {
  kind: StateKind;
  title: string;
  description?: ReactNode;
  action?: { label: string; onClick: () => void };
};

export function StateView({ kind, title, description, action }: StateViewProps) {
  const role = kind === "loading" || kind === "empty" ? "status" : "alert";

  return (
    <div className={`state state--${kind} ${kind}-state`} role={role}>
      <strong>{title}</strong>
      {description && <p>{description}</p>}
      {action && <button type="button" className="button button-quiet" onClick={action.onClick}>{action.label}</button>}
    </div>
  );
}
