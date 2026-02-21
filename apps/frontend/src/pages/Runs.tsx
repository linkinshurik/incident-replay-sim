import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

interface Run {
  runId: string;
  state: string;
  startedAt: string;
  finishedAt?: string;
  stats: {
    requests: number;
    errors: number;
    p95ms: number;
  };
}

const Runs: React.FC = () => {
  const [runs, setRuns] = useState<Run[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  const fetchRuns = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/replay/runs?limit=20");
      if (!res.ok) throw new Error("Failed to fetch runs");
      const data = await res.json();
      if (Array.isArray(data)) {
        setRuns(data);
      } else {
        setRuns([]);
      }
    } catch (e: any) {
      setError(e.message || "Error fetching runs");
      setRuns([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRuns();
  }, []);

  const openReport = (runId: string) => {
    navigate(`/replay/report?runId=${encodeURIComponent(runId)}`);
  };

  return (
    <div>
      <h2>Replay Runs</h2>
      {loading && <div className="loading-spinner" aria-label="Loading"></div>}
      {error && <div className="error-message">{error}</div>}
      {!loading && runs.length === 0 && <p>No runs found.</p>}
      {!loading && runs.length > 0 && (
        <table className="table">
          <thead>
            <tr>
              <th>Run ID</th>
              <th>State</th>
              <th>Started At</th>
              <th>Finished At</th>
              <th>Requests</th>
              <th>Errors</th>
              <th>P95 Latency (ms)</th>
              <th>Report</th>
            </tr>
          </thead>
          <tbody>
            {runs.map((run) => (
              <tr key={run.runId}>
                <td>{run.runId}</td>
                <td>{run.state}</td>
                <td>{new Date(run.startedAt).toLocaleString()}</td>
                <td>
                  {run.finishedAt
                    ? new Date(run.finishedAt).toLocaleString()
                    : "-"}
                </td>
                <td>{run.stats.requests}</td>
                <td>{run.stats.errors}</td>
                <td>{run.stats.p95ms}</td>
                <td>
                  <button
                    className="link-button"
                    onClick={() => openReport(run.runId)}
                  >
                    View
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
};

export default Runs;
