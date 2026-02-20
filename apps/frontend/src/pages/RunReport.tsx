import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';

interface Report {
  [key: string]: any;
}

const RunReport: React.FC = () => {
  const { runId } = useParams<{ runId: string }>();
  const [report, setReport] = useState<Report | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!runId) return;

    setLoading(true);
    fetch(`/replay/report?runId=${encodeURIComponent(runId)}`)
      .then(res => {
        if (!res.ok) {
          throw new Error(`Failed to fetch report, status: ${res.status}`);
        }
        return res.json();
      })
      .then(data => {
        setReport(data);
        setError(null);
      })
      .catch(err => {
        setError(err.message);
        setReport(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [runId]);

  return (
    <div style={{ padding: '1rem' }}>
      <h1>Run Report</h1>
      <p><Link to="/runs">Back to Runs List</Link></p>

      {loading && <p>Loading report...</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}

      {report && (
        <pre style={{ whiteSpace: 'pre-wrap', wordWrap: 'break-word', backgroundColor: '#f8f8f8', padding: '1rem', borderRadius: '4px' }}>
          {JSON.stringify(report, null, 2)}
        </pre>
      )}

      {!loading && !error && !report && <p>No report data available.</p>}
    </div>
  );
};

export default RunReport;
