import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { sessions as sessionsApi, type Session } from '../api/client';
import { exportReport, type ReportFormat } from '../api/report';

export default function Sessions() {
  const [list, setList] = useState<Session[]>([]);
  const [title, setTitle] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [exporting, setExporting] = useState<string>(''); // session id being exported
  const navigate = useNavigate();

  const fetchSessions = useCallback(async () => {
    try {
      setList(await sessionsApi.list());
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;
    setError('');
    try {
      const session = await sessionsApi.create(title.trim());
      setTitle('');
      navigate(`/chat/${session.id}`);
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await sessionsApi.delete(id);
      setList((prev) => prev.filter((s) => s.id !== id));
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleExport = async (id: string, format: ReportFormat) => {
    if (exporting) return;
    setExporting(id);
    setError('');
    try {
      await exportReport(id, format);
    } catch (err: any) {
      setError(err.message || '导出失败');
    } finally {
      setExporting('');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Sessions</h1>
        <p className="text-gray-400 mt-1">Create and manage security analysis sessions</p>
      </div>

      {/* Create Form */}
      <form onSubmit={handleCreate} className="flex gap-3">
        <input
          type="text"
          placeholder="Session title (e.g., 'Penetration Test — client-api')"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="flex-1 px-4 py-2.5 bg-gray-900/50 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition"
        />
        <button
          type="submit"
          className="px-6 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-lg transition"
        >
          Create
        </button>
      </form>

      {error && (
        <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded-lg px-3 py-2">{error}</div>
      )}

      {/* Session List */}
      <div className="space-y-3">
        {loading ? (
          <p className="text-gray-500 text-center py-8">Loading sessions...</p>
        ) : list.length === 0 ? (
          <div className="text-center py-12 text-gray-500">
            <p className="text-4xl mb-3">📋</p>
            <p>No sessions yet. Create one to start.</p>
          </div>
        ) : (
          list.map((s) => (
            <div
              key={s.id}
              className="flex items-center justify-between bg-gray-900/50 border border-gray-800 rounded-xl px-5 py-4 hover:border-gray-700 transition group"
            >
              <div
                className="flex-1 cursor-pointer min-w-0"
                onClick={() => navigate(`/chat/${s.id}`)}
              >
                <h3 className="text-gray-200 font-medium truncate">{s.title}</h3>
                <p className="text-gray-500 text-xs mt-1">
                  {new Date(s.created_at).toLocaleString()} · {s.status}
                </p>
              </div>
              <div className="flex items-center gap-2 ml-4 shrink-0">
                <select
                  onChange={(e) => {
                    const fmt = e.target.value as ReportFormat;
                    if (fmt) handleExport(s.id, fmt);
                    e.target.value = '';
                  }}
                  disabled={exporting === s.id}
                  className="opacity-0 group-hover:opacity-100 px-2 py-1 bg-gray-800 border border-gray-700 rounded text-gray-300 text-xs focus:outline-none focus:border-emerald-500 transition"
                  defaultValue=""
                >
                  <option value="" disabled>
                    {exporting === s.id ? '导出中…' : '导出'}
                  </option>
                  <option value="html">HTML</option>
                  <option value="pdf">PDF</option>
                  <option value="docx">DOCX</option>
                  <option value="xlsx">XLSX</option>
                </select>
                <button
                  onClick={(e) => { e.stopPropagation(); handleDelete(s.id); }}
                  className="opacity-0 group-hover:opacity-100 px-3 py-1 text-red-400 hover:bg-red-900/30 rounded text-sm transition"
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
