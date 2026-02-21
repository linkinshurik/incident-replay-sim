import React, { useEffect, useState } from "react";

const Scenarios: React.FC = () => {
  const [scenarioId, setScenarioId] = useState("");
  const [fileContent, setFileContent] = useState("");
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [scenarioList, setScenarioList] = useState<string[]>([]);

  const fetchScenarios = async () => {
    try {
      const res = await fetch("/scenarios/list");
      if (!res.ok) throw new Error("Failed to fetch scenarios");
      const data = await res.json();
      setScenarioList(Array.isArray(data) ? data : []);
    } catch {
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
    } catch (err: unknown) {
      setUploadError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <h2>Scenarios</h2>
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
