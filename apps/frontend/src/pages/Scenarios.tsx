import React, { useEffect, useState } from "react";

const Scenarios: React.FC = () => {
  const [scenarioId, setScenarioId] = useState("");
  const [fileContent, setFileContent] = useState("");
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [scenarioList, setScenarioList] = useState<string[]>([]);

  const [harScenarioId, setHarScenarioId] = useState("");
  const [harFile, setHarFile] = useState<File | null>(null);
  const [harUploading, setHarUploading] = useState(false);
  const [harUploadError, setHarUploadError] = useState<string | null>(null);

  const fetchScenarios = async () => {
    try {
      const res = await fetch("/scenarios/list");
      if (!res.ok) throw new Error("Failed to fetch scenarios");
      const data = await res.json();
      setScenarioList(Array.isArray(data) ? data : []);
    } catch (e) {
      setScenarioList([]);
    }
  };

  useEffect(() => {
    fetchScenarios();
  }, []);

  const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setUploadError(null);
    const file = e.target.files && e.target.files[0];
    if (!file) {
      setFileContent("");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === "string") {
        setFileContent(reader.result);
      } else {
        setFileContent("");
      }
    };
    reader.readAsText(file);
  };

  const onHarFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setHarUploadError(null);
    const file = e.target.files?.[0];
    setHarFile(file ?? null);
  };

  const onHarUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    setHarUploadError(null);
    if (!harScenarioId.match(/^[\w\-]+$/)) {
      setHarUploadError(
        "Scenario ID must contain only letters, digits, underscores or dashes"
      );
      return;
    }
    if (!harFile) {
      setHarUploadError("Select a HAR file");
      return;
    }
    setHarUploading(true);
    try {
      const formData = new FormData();
      formData.append("scenarioId", harScenarioId);
      formData.append("file", harFile);
      const res = await fetch("/scenarios/upload-har", {
        method: "POST",
        body: formData,
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || res.statusText);
      }
      await fetchScenarios();
      setHarScenarioId("");
      setHarFile(null);
    } catch (err: any) {
      setHarUploadError(err.message || "Upload failed");
    } finally {
      setHarUploading(false);
    }
  };

  const onUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    setUploadError(null);
    if (!scenarioId.match(/^[\w\-]+$/)) {
      setUploadError(
        "Scenario ID must contain only letters, digits, underscores or dashes"
      );
      return;
    }
    if (!fileContent) {
      setUploadError("No file content to upload");
      return;
    }
    setUploading(true);
    try {
      const res = await fetch("/scenarios/upload", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ scenarioId, jsonl: fileContent }),
      });
      if (!res.ok) {
        throw new Error(`Upload failed: ${res.statusText}`);
      }
      await fetchScenarios();
      setScenarioId("");
      setFileContent("");
    } catch (err: any) {
      setUploadError(err.message || "Upload failed");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <h2>Scenarios</h2>

      <section className="upload-section">
        <h3>Upload HAR (Chrome DevTools)</h3>
        <p className="hint">
          Export network activity from Chrome DevTools (Network tab → right-click → Save all as HAR with content), then upload here to use as a load-test scenario. Requests will keep their order and timestamps for replay.
        </p>
        <form onSubmit={onHarUpload}>
          <label htmlFor="harScenarioId">Scenario ID:</label>
          <input
            id="harScenarioId"
            type="text"
            value={harScenarioId}
            onChange={(e) => setHarScenarioId(e.target.value)}
            placeholder="e.g. my-api-session"
            disabled={harUploading}
          />
          <label htmlFor="harFile">HAR file:</label>
          <input
            id="harFile"
            type="file"
            accept=".har,application/json"
            onChange={onHarFileChange}
            disabled={harUploading}
          />
          <button type="submit" disabled={harUploading || !harFile}>
            Upload HAR
          </button>
          {harUploadError && <div className="error-message">{harUploadError}</div>}
        </form>
      </section>

      <section className="upload-section">
        <h3>Upload JSONL</h3>
        <form onSubmit={onUpload}>
          <label htmlFor="scenarioId">Scenario ID:</label>
          <input
            id="scenarioId"
            type="text"
            value={scenarioId}
            onChange={(e) => setScenarioId(e.target.value)}
            placeholder="letters, digits, underscore, dash"
            required
            disabled={uploading}
          />

          <label htmlFor="scenarioFile">Scenario File (JSONL):</label>
          <input
            id="scenarioFile"
            type="file"
            accept="text/plain,application/jsonl"
            onChange={onFileChange}
            disabled={uploading}
            required
          />

          <button type="submit" disabled={uploading}>
            Upload
          </button>

          {uploadError && <div className="error-message">{uploadError}</div>}
        </form>
      </section>

      <h3>Available Scenarios</h3>
      {scenarioList.length === 0 ? (
        <p>No scenarios found.</p>
      ) : (
        <ul>
          {scenarioList.map((id) => (
            <li key={id}>{id}</li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default Scenarios;
