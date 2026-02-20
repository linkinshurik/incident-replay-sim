import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

export interface RunStats {
  requests: number;
  errors: number;
  p95ms: number;
}

export interface Run {
  runId: string;
  state: string;
  startedAt: string;
  finishedAt?: string;
  stats: RunStats;
}

const Runs: React.FC = () => {
  const [runs, setRuns] = useState<Run[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch('/replay/runs?limit=20')
      .then(res => {
        if (!res.ok) {
          throw new Error(`Failed to fetch runs, status: ${res.status}`);
        }
        return res.json();
      })
      .then((data: Run[]) => {
        setRuns(data);
        setError(null);
      })
      .catch(err => {
        setError(err.message);
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  return (
    <div style={{ padding: '1rem' }}>
      <h1>Runs List</h1>
      {loading && <p>Loading runs...</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}
      {!loading && !error && runs.length === 0 && <p>No runs found.</p>}
      {!loading && !error && runs.length > 0 && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>Run ID</th>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>State</th>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>Started At</th>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>Finished At</th>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>Requests</th>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>Errors</th>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>P95 ms</th>
              <th style={{ border: '1px solid #ccc', padding: '8px' }}>Report</th>
            </tr>
          </thead>
          <tbody>
            {runs.map(run => (
              <tr key={run.runId}>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>{run.runId}</td>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>{run.state}</td>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>{new Date(run.startedAt).toLocaleString()}</td>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>{run.finishedAt ? new Date(run.finishedAt).toLocaleString() : '-'}</td>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>{run.stats.requests}</td>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>{run.stats.errors}</td>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>{run.stats.p95ms}</td>
                <td style={{ border: '1px solid #ccc', padding: '8px' }}>
                  <Link to={`/runs/${encodeURIComponent(run.runId)}`}>View Report</Link>
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
