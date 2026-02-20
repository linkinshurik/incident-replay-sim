import React, { useState } from 'react';

const StartReplay: React.FC = () => {
  const [scenarioId, setScenarioId] = useState('');
  const [targetBaseUrl, setTargetBaseUrl] = useState('');
  const [rps, setRps] = useState(10);
  const [durationSec, setDurationSec] = useState(60);
  const [mode, setMode] = useState<'burst' | 'timestamp'>('burst');
  const [speed, setSpeed] = useState(1.0);
  const [maxDelayMs, setMaxDelayMs] = useState(0);
  const [startFromTs, setStartFromTs] = useState('');
  const [endAtTs, setEndAtTs] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [runId, setRunId] = useState<string | null>(null);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setRunId(null);

    if (!scenarioId.match(/^[\w\-]+$/)) {
      setError('Scenario ID must contain only letters, digits, underscores or dashes');
      return;
    }
    if (!targetBaseUrl) {
      setError('Target Base URL is required');
      return;
    }
    if (rps <= 0) {
      setError('RPS must be positive');
      return;
    }
    if (durationSec <= 0) {
      setError('Duration must be positive');
      return;
    }
    if (speed <= 0) {
      setError('Speed must be positive');
      return;
    }
    // Validate timestamps if mode=timestamp
    if (mode === 'timestamp') {
      if (startFromTs && isNaN(Date.parse(startFromTs))) {
        setError('Start From Timestamp is invalid');
        return;
      }
      if (endAtTs && isNaN(Date.parse(endAtTs))) {
        setError('End At Timestamp is invalid');
        return;
      }
    }

    const payload: any = {
      scenarioId,
      targetBaseUrl,
      rps,
      durationSec,
      mode,
      speed,
      maxDelayMs
    };
    if (mode === 'timestamp') {
      if (startFromTs) payload.startFromTs = startFromTs;
      if (endAtTs) payload.endAtTs = endAtTs;
    }

    setLoading(true);

    try {
      const res = await fetch('/replay/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!res.ok) {
        let errText = await res.text();
        setError(`Failed to start replay: ${res.statusText} ${errText}`);
        setLoading(false);
        return;
      }
      const data = await res.json();
      if (data.runId) {
        setRunId(data.runId);
      } else {
        setError('No runId returned');
      }
    } catch (e: any) {
      setError(e.message || 'Failed to start replay');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>Start Replay</h2>
      <form onSubmit={onSubmit}>
        <label htmlFor="scenarioId">Scenario ID:</label>
        <input
          id="scenarioId"
          type="text"
          value={scenarioId}
          onChange={e => setScenarioId(e.target.value)}
          required
          disabled={loading}
        />

        <label htmlFor="targetBaseUrl">Target Base URL:</label>
        <input
          id="targetBaseUrl"
          type="url"
          value={targetBaseUrl}
          onChange={e => setTargetBaseUrl(e.target.value)}
          placeholder="http://..."
          required
          disabled={loading}
        />

        <label htmlFor="rps">Requests Per Second (RPS):</label>
        <input
          id="rps"
          type="number"
          min={1}
          value={rps}
          onChange={e => setRps(Number(e.target.value))}
          required
          disabled={loading}
        />

        <label htmlFor="durationSec">Duration (seconds):</label>
        <input
          id="durationSec"
          type="number"
          min={1}
          value={durationSec}
          onChange={e => setDurationSec(Number(e.target.value))}
          required
          disabled={loading}
        />

        <label htmlFor="mode">Replay Mode:</label>
        <select
          id="mode"
          value={mode}
          onChange={e => setMode(e.target.value as 'burst' | 'timestamp')}
          disabled={loading}
        >
          <option value="burst">burst</option>
          <option value="timestamp">timestamp</option>
        </select>

        <label htmlFor="speed">Speed (multiplier, only timestamp mode):</label>
        <input
          id="speed"
          type="number"
          min={0.01}
          step={0.01}
          value={speed}
          onChange={e => setSpeed(Number(e.target.value))}
          disabled={loading || mode !== 'timestamp'}
        />

        <label htmlFor="maxDelayMs">Max Delay (ms, only timestamp mode):</label>
        <input
          id="maxDelayMs"
          type="number"
          min={0}
          value={maxDelayMs}
          onChange={e => setMaxDelayMs(Number(e.target.value))}
          disabled={loading || mode !== 'timestamp'}
        />

        <label htmlFor="startFromTs">Start From Timestamp (RFC3339, optional):</label>
        <input
          id="startFromTs"
          type="datetime-local"
          value={startFromTs}
          onChange={e => setStartFromTs(e.target.value)}
          disabled={loading || mode !== 'timestamp'}
        />

        <label htmlFor="endAtTs">End At Timestamp (RFC3339, optional):</label>
        <input
          id="endAtTs"
          type="datetime-local"
          value={endAtTs}
          onChange={e => setEndAtTs(e.target.value)}
          disabled={loading || mode !== 'timestamp'}
        />

        <button type="submit" disabled={loading}>Start Replay</button>
      </form>

      {loading && <div className="loading-spinner" aria-label="Loading"></div>}

      {error && <div className="error-message">{error}</div>}

      {runId && (
        <div>
          <p>Replay started! Run ID: <strong>{runId}</strong></p>
          <p>
            View report: <a href={`/replay/report?runId=${encodeURIComponent(runId)}`} target="_blank" rel="noreferrer">Report</a>
          </p>
        </div>
      )}
    </div>
  );
};

export default StartReplay;
