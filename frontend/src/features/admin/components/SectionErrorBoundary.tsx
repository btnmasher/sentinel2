import { Component, type ReactNode } from "react";

type SectionErrorBoundaryProps = {
  children: ReactNode;
  fallbackTitle?: string;
};

type SectionErrorBoundaryState = {
  hasError: boolean;
  errorMessage: string;
};

export default class SectionErrorBoundary extends Component<
  SectionErrorBoundaryProps,
  SectionErrorBoundaryState
> {
  state: SectionErrorBoundaryState = {
    hasError: false,
    errorMessage: "",
  };

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, errorMessage: error.message };
  }

  componentDidCatch() {
    // Errors are reported by the browser console; this boundary keeps the UI stable.
  }

  handleReset = () => {
    this.setState({ hasError: false, errorMessage: "" });
  };

  render() {
    if (this.state.hasError) {
      return (
        <section className="card bg-base-200/70 border border-red-800/60">
          <div className="card-body space-y-3">
            <h2 className="font-display text-2xl">
              {this.props.fallbackTitle || "Section Error"}
            </h2>
            <p className="text-xs text-slate-300">
              This section failed to render. Try reloading it.
            </p>
            {this.state.errorMessage && (
              <p className="text-xs text-red-300">{this.state.errorMessage}</p>
            )}
            <button className="btn btn-xs btn-outline" onClick={this.handleReset}>
              Reload section
            </button>
          </div>
        </section>
      );
    }

    return this.props.children;
  }
}
