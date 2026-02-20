import React, { useState } from 'react';

function StartReplay() {
  const [scenarioId, setScenarioId] = useState('');
  const [targetBaseUrl, setTargetBaseUrl] = useState('');
  const [rps, setRps] = useState(10);
  const [durationSec, setDurationSec] = useState(60);
  const [mode, setMode] = useState('burst');
  const [speed, setSpeed] = useState(1.0);
  const [maxDelayMs, setMaxDelayMs] = useState(0);
  const [startFromTs, setStartFromTs] = useState('');
  const [endAtTs, setEndAtTs] = useState('');

  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  const [runId, setRunId] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg('');
    setRunId(null);

    if (!scenarioId.trim() || !targetBaseUrl.trim()) {
      setErrorMsg('Scenario ID and Target Base URL are required');
      return;
    }

    if (rps <= 0) {
      setErrorMsg('RPS must be greater than 0');
      return;
    }

    if (durationSec <= 0) {
      setErrorMsg('Duration must be greater than 0');
      return;
    }

    if (mode !== 'burst' && mode !== 'timestamp') {
      setErrorMsg('Mode must be burst or timestamp');
      return;
    }

    if (speed !== 0 && speed <= 0) {
      setErrorMsg('Speed must be greater than 0');
      return;
    }

    if (maxDelayMs < 0) {
      setErrorMsg('Max Delay must be >= 0');
      return;
    }

    setLoading(true);
    try {
      const body = {
        scenarioId: scenarioId.trim(),
        targetBaseUrl: targetBaseUrl.trim(),
        rps,
        durationSec,
        mode,
        speed,
        maxDelayMs,
        startFromTs: startFromTs.trim() || undefined,
        endAtTs: endAtTs.trim() || undefined,
      };

      const res = await fetch('/replay/start', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const errText = await res.text();
        setErrorMsg(`Failed to start replay: ${errText}`);
        setLoading(false);
        return;
      }

      const data = await res.json();
      setRunId(data.runId);
    } catch (err) {
      setErrorMsg(`Request error: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: 600, margin: 'auto' }}>
      <h2>Start Replay</h2>
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: 12 }}>
          <label>
            Scenario ID:<br />
            <input
              type="text"
              value={scenarioId}
              onChange={(e) => setScenarioId(e.target.value)}
              required
              style={{ width: '100%' }}
            />
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            Target Base URL:<br />
            <input
              type="text"
              value={targetBaseUrl}
              onChange={(e) => setTargetBaseUrl(e.target.value)}
              required
              placeholder="http://example.com"
              style={{ width: '100%' }}
            />
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            RPS (requests per second):<br />
            <input
              type="number"
              value={rps}
              onChange={(e) => setRps(Number(e.target.value))}
              min={1}
              required
              style={{ width: '100%' }}
            />
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            Duration (seconds):<br />
            <input
              type="number"
              value={durationSec}
              onChange={(e) => setDurationSec(Number(e.target.value))}
              min={1}
              required
              style={{ width: '100%' }}
            />
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            Mode:<br />
            <select value={mode} onChange={(e) => setMode(e.target.value)} style={{ width: '100%' }}>
              <option value="burst">burst</option>
              <option value="timestamp">timestamp</option>
            </select>
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            Speed (optional, leave 1 for normal):<br />
            <input
              type="number"
              value={speed}
              onChange={(e) => setSpeed(Number(e.target.value))}
              min={0}
              step={0.1}
              style={{ width: '100%' }}
            />
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            Max Delay (ms, optional, default 0):<br />
            <input
              type="number"
              value={maxDelayMs}
              onChange={(e) => setMaxDelayMs(Number(e.target.value))}
              min={0}
              style={{ width: '100%' }}
            />
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            Start From Timestamp (RFC3339, optional):<br />
            <input
              type="text"
              value={startFromTs}
              onChange={(e) => setStartFromTs(e.target.value)}
              placeholder="2026-02-20T09:20:21Z"
              style={{ width: '100%' }}
            />
          </label>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label>
            End At Timestamp (RFC3339, optional):<br />
            <input
              type="text"
              value={endAtTs}
              onChange={(e) => setEndAtTs(e.target.value)}
              placeholder="2026-02-20T09:50:21Z"
              style={{ width: '100%' }}
            />
          </label>
        </div>

        {errorMsg && <div style={{ color: 'red', marginBottom: 12 }}>{errorMsg}</div>}

        <button type="submit" disabled={loading} style={{ padding: '8px 16px' }}>
          {loading ? 'Starting...' : 'Start Replay'}
        </button>
      </form>

      {runId && (
        <div style={{ marginTop: 20 }}>
          <h3>Replay started</h3>
          <p>
            Run ID: <strong>{runId}</strong>
          </p>
          <p>
            <a href={`/runs/${runId}`} target="_blank" rel="noopener noreferrer">
              View Run Report
            </a>
          </p>
        </div>
      )}
    </div>
  );
}

export default StartReplay;
