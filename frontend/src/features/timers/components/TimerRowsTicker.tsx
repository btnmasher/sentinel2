import { useEffect, useState } from "react";
import type { ReactNode } from "react";

type TimerRowsTickerProps = {
  children: (nowMs: number) => ReactNode;
};

export default function TimerRowsTicker({ children }: TimerRowsTickerProps) {
  const [nowMs, setNowMs] = useState(() => Date.now());

  useEffect(() => {
    const tick = window.setInterval(() => setNowMs(Date.now()), 1000);
    return () => window.clearInterval(tick);
  }, []);

  return <>{children(nowMs)}</>;
}
