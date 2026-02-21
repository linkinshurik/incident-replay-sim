import React, { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";

const RunReport: React.FC = () => {
  const location = useLocation();
  const [runId, setRunId] = useState<string | null>(null);
  const [report, setReport] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const id = params.get("runId");
    setRunId(id);
  }, [location.search]);

  useEffect(() => {
    if (!runId) return;
    setLoading(true);
    setError(null);
    setReport(null);

    fetch(`/replay/report?runId=${encodeURIComponent(runId)}`)
      .then((res) => {
        if (!res.ok)
          throw new Error(`Failed to load report: ${res.statusText}`);
        return res.json();
      })
      .then((data) => {
        setReport(data);
      })
      .catch((e: unknown) =>
        setError(e instanceof Error ? e.message : "Failed to fetch report")
      )
      .finally(() => setLoading(false));
  }, [runId]);

  if (!runId) {
    return (
      <div>
        <p>No runId specified in URL.</p>
      </div>
    );
  }

  return (
    <div>
      <h2>Run Report - {runId}</h2>
      {loading && <div className="loading-spinner" aria-label="Loading"></div>}
      {error && <div className="error-message">{error}</div>}
      {report && (
        <pre style={{ whiteSpace: "pre-wrap", wordBreak: "break-word" }}>
          {JSON.stringify(report, null, 2)}
        </pre>
      )}
    </div>
  );
};

export default RunReport;
