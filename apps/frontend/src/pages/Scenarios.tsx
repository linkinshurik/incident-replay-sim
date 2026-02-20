import React, { useState, useEffect, ChangeEvent, FormEvent } from 'react';

const API_SCENARIOS_LIST = '/scenarios/list';
const API_SCENARIOS_UPLOAD = '/scenarios/upload';

export default function Scenarios() {
  const [scenarios, setScenarios] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);
  const [scenarioId, setScenarioId] = useState('');
  const [jsonl, setJsonl] = useState('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const fetchScenarios = async () => {
    try {
      const res = await fetch(API_SCENARIOS_LIST);
      if (!res.ok) throw new Error(`HTTP error ${res.status}`);
      const data = await res.json();
      if (Array.isArray(data)) {
        setScenarios(data);
      } else {
        setScenarios([]);
      }
    } catch (e) {
      setErrorMsg('Failed to fetch scenarios');
      setScenarios([]);
    }
  };

  useEffect(() => {
    fetchScenarios();
  }, []);

  const handleScenarioIdChange = (e: ChangeEvent<HTMLInputElement>) => {
    setScenarioId(e.target.value);
  };

  const handleJsonlChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    setJsonl(e.target.value);
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    setSuccessMsg(null);
    if (!scenarioId.match(/^[a-zA-Z0-9_-]+$/)) {
      setErrorMsg('Scenario ID must contain only letters, digits, underscore, or dash');
      return;
    }
    if (jsonl.trim() === '') {
      setErrorMsg('JSONL content cannot be empty');
      return;
    }

    setUploading(true);
    try {
      const res = await fetch(API_SCENARIOS_UPLOAD, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scenarioId, jsonl }),
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(errorText || `HTTP error ${res.status}`);
      }

      const data = await res.json();
      if (data.status === 'ok') {
        setSuccessMsg(`Scenario '${data.scenarioId}' uploaded successfully.`);
        setScenarioId('');
        setJsonl('');
        fetchScenarios();
      } else {
        setErrorMsg('Upload failed');
      }
    } catch (error) {
      setErrorMsg(error instanceof Error ? error.message : String(error));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div style={{ maxWidth: 800, margin: '1rem auto', padding: '0 1rem' }}>
      <h2>Scenarios</h2>

      <section style={{ marginBottom: '2rem' }}>
        <h3>Upload Scenario</h3>
        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom: '0.5rem' }}>
            <label htmlFor="scenarioId" style={{ display: 'block', marginBottom: 4 }}>
              Scenario ID (letters, digits, underscore, dash):
            </label>
            <input
              id="scenarioId"
              type="text"
              value={scenarioId}
              onChange={handleScenarioIdChange}
              disabled={uploading}
              required
              style={{ width: '100%', padding: 6, fontSize: 14 }}
              maxLength={64}
              pattern="[a-zA-Z0-9_-]+"
              title="Only letters, digits, underscore, or dash allowed"
            />
          </div>
          <div style={{ marginBottom: '0.5rem' }}>
            <label htmlFor="jsonl" style={{ display: 'block', marginBottom: 4 }}>
              JSON Lines Content:
            </label>
            <textarea
              id="jsonl"
              value={jsonl}
              onChange={handleJsonlChange}
              rows={10}
              disabled={uploading}
              required
              style={{ width: '100%', fontSize: 14, fontFamily: 'monospace' }}
              placeholder={`Example:\n{"type":"http","method":"GET","path":"/api/foo","weight":1}\n{"type":"http","method":"POST","path":"/api/bar","weight":2}`}
            />
          </div>
          <button type="submit" disabled={uploading} style={{ padding: '0.5rem 1rem', fontSize: 16 }}>
            {uploading ? 'Uploading...' : 'Upload'}
          </button>
        </form>
        {errorMsg && <p style={{ color: 'red', marginTop: '0.5rem' }}>{errorMsg}</p>}
        {successMsg && <p style={{ color: 'green', marginTop: '0.5rem' }}>{successMsg}</p>}
      </section>

      <section>
        <h3>Available Scenarios</h3>
        {scenarios.length === 0 ? (
          <p>No scenarios available.</p>
        ) : (
          <ul style={{ listStyleType: 'none', paddingLeft: 0 }}>
            {scenarios.map((id) => (
              <li key={id} style={{ padding: '0.25rem 0', borderBottom: '1px solid #ddd' }}>
                {id}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
