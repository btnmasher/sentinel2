import React from "react";

const DEFAULT_MESSAGE = "Something went wrong.";

type ErrorBoundaryProps = {
  name?: string;
  fallback?: React.ReactNode;
  children: React.ReactNode;
};

type ErrorBoundaryState = {
  hasError: boolean;
  error?: Error;
};

export default class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    const name = this.props.name ? `(${this.props.name})` : "";
    // eslint-disable-next-line no-console
    console.error(`ErrorBoundary ${name}`, error, info);
  }

  render() {
    const { hasError } = this.state;
    const { children, fallback, name } = this.props;

    if (!hasError) return children;

    if (fallback) return <>{fallback}</>;

    return (
      <div className="card bg-base-200/70 border border-slate-800">
        <div className="card-body">
          <h2 className="font-display text-lg">{name ?? "Error"}</h2>
          <p className="text-sm text-slate-300">{DEFAULT_MESSAGE}</p>
        </div>
      </div>
    );
  }
}
