import React from 'react';
import { BrowserRouter as Router, Routes, Route, NavLink } from 'react-router-dom';
import Scenarios from './pages/Scenarios';
import StartReplay from './pages/StartReplay';
import Runs from './pages/Runs';
import RunReport from './pages/RunReport';
import './index.css';

function BackendHealth() {
  const [status, setStatus] = React.useState('unknown');

  React.useEffect(() => {
    let mounted = true;
    async function checkHealth() {
      try {
        const res = await fetch('/healthz');
        if (!mounted) return;
        if (res.ok) {
          setStatus('ok');
        } else {
          setStatus('error');
        }
      } catch {
        if (mounted) setStatus('error');
      }
    }
    checkHealth();
    const interval = setInterval(checkHealth, 5000);
    return () => {
      mounted = false;
      clearInterval(interval);
    };
  }, []);

  let color = 'gray';
  let text = 'Unknown';
  if (status === 'ok') {
    color = 'green';
    text = 'Healthy';
  } else if (status === 'error') {
    color = 'red';
    text = 'Unhealthy';
  }

  return (<span style={{color, fontWeight: 'bold'}} title="Backend health status">{text}</span>);
}

const App: React.FC = () => {
  return (
    <Router>
      <div className="app-container">
        <header className="header">
          <div className="header-left">
            <h1 className="app-name">Incident Replay</h1>
            <div className="backend-health">
              Backend health: <BackendHealth />
            </div>
          </div>
          <nav className="nav">
            <NavLink className="nav-link" to="/scenarios" end>
              Scenarios
            </NavLink>
            <NavLink className="nav-link" to="/start-replay">
              Start Replay
            </NavLink>
            <NavLink className="nav-link" to="/runs">
              Runs
            </NavLink>
          </nav>
        </header>
        <main className="main">
          <Routes>
            <Route path="/" element={<Scenarios />} />
            <Route path="/scenarios" element={<Scenarios />} />
            <Route path="/start-replay" element={<StartReplay />} />
            <Route path="/runs" element={<Runs />} />
            <Route path="/replay/report" element={<RunReport />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
};

export default App;
