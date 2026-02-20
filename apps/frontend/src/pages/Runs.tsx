import React from 'react';
import { Link } from 'react-router-dom';

const Runs: React.FC = () => {
  return (
    <div>
      <h1>Runs</h1>
      <p>Basic page skeleton listing replay runs.</p>
      <p>
        Example run link: <Link to="/runs/run-1234567890">run-1234567890</Link>
      </p>
    </div>
  );
};

export default Runs;
