type LoadingCardProps = {
  title?: string;
  subtitle?: string;
  full?: boolean;
};

export default function LoadingCard({
  title = "Loading",
  subtitle,
  full = false,
}: LoadingCardProps) {
  const wrapperClass = full
    ? "min-h-screen flex items-center justify-center"
    : "";

  return (
    <div className={wrapperClass}>
      <div className="card bg-base-200/70 border border-slate-800">
        <div className="card-body">
          <div className="flex items-center gap-3">
            <span className="loading loading-spinner loading-sm text-primary" />
            <h2 className="font-display text-2xl">{title}</h2>
          </div>
          {subtitle && <p className="text-sm text-slate-400">{subtitle}</p>}
        </div>
      </div>
    </div>
  );
}
