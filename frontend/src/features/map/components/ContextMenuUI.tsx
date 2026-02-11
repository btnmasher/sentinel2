import type { ReactNode } from "react";

export function ContextMenuList({ children }: { children: ReactNode }) {
  return <div className="context-menu-list">{children}</div>;
}

export function ContextMenuTitle({ children }: { children: ReactNode }) {
  return <div className="context-menu-title">{children}</div>;
}

export function ContextMenuSeparator() {
  return <div className="context-menu-separator" />;
}

type ItemProps = {
  children: ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  href?: string;
  target?: string;
  rel?: string;
  muted?: boolean;
  sub?: boolean;
};

export function ContextMenuItem({
  children,
  onClick,
  disabled,
  href,
  target,
  rel,
  muted,
  sub,
}: ItemProps) {
  const className = [
    "context-menu-item",
    muted ? "is-muted" : "",
    sub ? "context-menu-sub" : "",
  ]
    .filter(Boolean)
    .join(" ");

  if (href) {
    return (
      <a className={className} href={href} target={target} rel={rel}>
        {children}
      </a>
    );
  }

  return (
    <button className={className} onClick={onClick} disabled={disabled}>
      {children}
    </button>
  );
}

export function ContextMenuChevron() {
  return <span className="context-menu-chevron">›</span>;
}
