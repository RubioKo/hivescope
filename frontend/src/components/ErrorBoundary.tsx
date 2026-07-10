import { Component, type ReactNode } from "react";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <div className="w-full h-full flex items-center justify-center bg-hive-bg">
          <div className="text-center max-w-md px-6">
            <p className="text-hive-red text-lg font-semibold mb-2">Something went wrong</p>
            <p className="text-hive-muted text-sm mb-4 font-mono">
              {this.state.error?.message || "Unknown error"}
            </p>
            <button
              onClick={this.handleReset}
              className="px-4 py-2 text-sm rounded border border-hive-border bg-hive-surface text-hive-text hover:bg-hive-border transition-colors"
            >
              Try again
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
