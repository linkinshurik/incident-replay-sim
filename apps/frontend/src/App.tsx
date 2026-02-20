import React from 'react';
import { Routes, Route, Link, Navigate } from 'react-router-dom';
import Scenarios from './pages/Scenarios';
import StartReplay from './pages/StartReplay';
import Runs from './pages/Runs';
import RunReport from './pages/RunReport';

const App: React.FC = () => {
  return (
    <div>
      <nav style={{ padding: '1rem', borderBottom: '1px solid #ccc' }}>
        <Link to="/scenarios" style={{ marginRight: 10 }}>
          Scenarios
        </Link>
        <Link to="/start-replay" style={{ marginRight: 10 }}>
          Start Replay
        </Link>
        <Link to="/runs" style={{ marginRight: 10 }}>
          Runs
        </Link>
      </nav>
      <div style={{ padding: '1rem' }}>
        <Routes>
          <Route path="/" element={<Navigate to="/scenarios" replace />} />
          <Route path="/scenarios" element={<Scenarios />} />
          <Route path="/start-replay" element={<StartReplay />} />
          <Route path="/runs" element={<Runs />} />
          <Route path="/runs/:runId" element={<RunReport />} />
          <Route path="*" element={<div>Page Not Found</div>} />
        </Routes>
      </div>
    </div>
  );
};

export default App;
