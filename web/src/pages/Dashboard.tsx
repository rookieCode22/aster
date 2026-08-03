import { useEffect, useState } from 'react';
import { sessions, skills, modules, health } from '../api/client';
import { useAuth } from '../contexts/AuthContext';

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
      <p className="text-gray-400 text-sm">{label}</p>
      <p className={`text-3xl font-bold mt-1 ${color}`}>{value}</p>
    </div>
  );
}

export default function Dashboard() {
  const { user } = useAuth();
  const [sessionCount, setSessionCount] = useState(0);
  const [skillCount, setSkillCount] = useState(0);
  const [moduleCount, setModuleCount] = useState(0);
  const [serverStatus, setServerStatus] = useState<'ok' | 'error'>('ok');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      sessions.list().then((s) => setSessionCount(s.length)).catch(() => {}),
      skills.list().then((s) => setSkillCount(s.length)).catch(() => {}),
      modules.list().then((m) => setModuleCount(m.length)).catch(() => {}),
      health()
        .then((d) => setServerStatus(d.ok ? 'ok' : 'error'))
        .catch(() => setServerStatus('error')),
    ]).finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Dashboard</h1>
        <p className="text-gray-400 mt-1">Welcome back, {user?.username}</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Sessions" value={sessionCount} color="text-blue-400" />
        <StatCard label="Security Skills" value={skillCount} color="text-emerald-400" />
        <StatCard label="Modules" value={moduleCount} color="text-purple-400" />
        <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
          <p className="text-gray-400 text-sm">Server Status</p>
          <div className="flex items-center gap-2 mt-2">
            <span className={`w-2.5 h-2.5 rounded-full ${serverStatus === 'ok' ? 'bg-emerald-400' : 'bg-red-400'}`} />
            <p className="text-lg font-bold text-gray-200">{serverStatus === 'ok' ? 'Online' : 'Offline'}</p>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
        <h2 className="text-lg font-semibold text-gray-200 mb-4">Quick Actions</h2>
        <div className="flex flex-wrap gap-3">
          <a href="/sessions" className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium transition">
            + New Session
          </a>
          <a href="/skills" className="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg text-sm font-medium transition">
            Manage Skills
          </a>
        </div>
      </div>

      {loading && (
        <div className="text-center text-gray-500 text-sm py-8">Loading...</div>
      )}
    </div>
  );
}
