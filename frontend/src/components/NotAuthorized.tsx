type NotAuthorizedProps = {
  message?: string;
};

export default function NotAuthorized({ message }: NotAuthorizedProps) {
  return (
    <div className="card bg-base-200/70 border border-slate-800">
      <div className="card-body">
        <h2 className="font-display text-2xl">Not authorized</h2>
        <p className="text-sm text-slate-400">
          {message || "You do not have permission to view this page."}
        </p>
      </div>
    </div>
  );
}
