import { Routes, Route, Link } from 'react-router-dom';
import FlowList from './pages/FlowList';
import FlowEdit from './pages/FlowEdit';
import FlowLogs from './pages/FlowLogs';
import Logo from './Logo';

function App() {
  return (
    <div className="app">
      <header className="header">
        <Link to="/" className="header-brand">
          <Logo size={28} />
          <span className="header-name">NianHe</span>
        </Link>
        <span className="header-divider" />
        <span className="header-tagline">API 胶水平台</span>
      </header>
      <Routes>
        <Route path="/" element={<FlowList />} />
        <Route path="/flows/:id" element={<FlowEdit />} />
        <Route path="/flows/:id/logs" element={<FlowLogs />} />
      </Routes>
    </div>
  );
}

export default App;
