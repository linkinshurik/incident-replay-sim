import React from 'react';
import { useParams } from 'react-router-dom';

const RunReport: React.FC = () => {
  const { runId } = useParams<{ runId: string }>();

  return (
    <div>
      <h1>Run Report</h1>
      <p>Basic page skeleton for Run Report details.</p>
      <p>Run ID: {runId}</p>
    </div>
  );
};

export default RunReport;
