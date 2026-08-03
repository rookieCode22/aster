import { useEffect, useState, useCallback } from 'react';
import { license, type LicenseStatus, type LicenseRecord } from '../api/client';

function statusBadge(st: LicenseStatus | null) {
  if (!st || st.active === false) return { text: '未激活', cls: 'bg-gray-800 text-gray-400' };
  if (st.days_remaining > 30) return { text: '有效', cls: 'bg-emerald-900/40 text-emerald-400' };
  if (st.days_remaining >= 0) return { text: `剩余 ${st.days_remaining} 天`, cls: 'bg-amber-900/40 text-amber-400' };
  return { text: '永久授权', cls: 'bg-sky-900/40 text-sky-400' };
}

export default function License() {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [history, setHistory] = useState<LicenseRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [licenseKey, setLicenseKey] = useState('');
  const [activating, setActivating] = useState(false);

  const fetch = useCallback(async () => {
    try {
      const [st, hist] = await Promise.all([license.status(), license.history()]);
      setStatus(st);
      setHistory(hist);
      setError('');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetch(); }, [fetch]);

  const handleActivate = async (e: React.FormEvent) => {
    e.preventDefault();
    const key = licenseKey.trim();
    if (!key) return;
    setActivating(true);
    setError('');
    try {
      // 校验是一个合法 JSON 后再提交
      JSON.parse(key);
      await license.activate(key);
      setLicenseKey('');
      await fetch();
      alert('授权激活成功');
    } catch (err: any) {
      // JSON.parse 失败时提示更友好
      const msg = err.message?.includes('Unexpected')
        ? '请输入完整的授权文件内容（JSON）'
        : err.message;
      setError(msg);
    } finally {
      setActivating(false);
    }
  };

  const badge = statusBadge(status);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">License 管理</h1>
        <p className="text-gray-400 mt-1">激活与管理安全评估授权</p>
      </div>

      {error && (
        <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded-lg px-3 py-2">{error}</div>
      )}

      {/* Current status */}
      <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
        <div className="flex items-center justify-between">
          <h2 className="text-gray-200 font-medium">当前授权</h2>
          <span className={`text-xs px-2 py-1 rounded-full ${badge.cls}`}>{badge.text}</span>
        </div>
        {loading ? (
          <p className="text-gray-500 text-sm py-4">加载中…</p>
        ) : status?.active ? (
          <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <p className="text-gray-500 text-xs">客户</p>
              <p className="text-gray-200 mt-1">{status.customer || '—'}</p>
            </div>
            <div>
              <p className="text-gray-500 text-xs">套餐</p>
              <p className="text-gray-200 mt-1 capitalize">{status.plan || '—'}</p>
            </div>
            <div>
              <p className="text-gray-500 text-xs">最大并发 Agent</p>
              <p className="text-gray-200 mt-1">{status.max_agents === 0 ? '不限' : status.max_agents}</p>
            </div>
            <div>
              <p className="text-gray-500 text-xs">到期时间</p>
              <p className="text-gray-200 mt-1">{status.expires_at ? new Date(status.expires_at).toLocaleString() : '永久'}</p>
            </div>
          </div>
        ) : status ? (
          <p className="text-gray-500 text-sm py-4">尚未激活任何授权，请粘贴授权文件内容激活。</p>
        ) : null}
      </div>

      {/* Activate form */}
      <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
        <h2 className="text-gray-200 font-medium">激活授权</h2>
        <form onSubmit={handleActivate} className="mt-4 space-y-3">
          <textarea
            value={licenseKey}
            onChange={(e) => setLicenseKey(e.target.value)}
            rows={6}
            placeholder='粘贴授权文件完整内容（JSON，包含 signature 字段）'
            className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition text-sm font-mono resize-y"
            required
          />
          <button
            type="submit"
            disabled={activating}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white font-medium rounded-lg transition text-sm"
          >
            {activating ? '激活中…' : '激活'}
          </button>
        </form>
      </div>

      {/* History */}
      <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
        <h2 className="text-gray-200 font-medium">激活历史</h2>
        {history.length === 0 ? (
          <p className="text-gray-500 text-sm py-4">暂无激活记录。</p>
        ) : (
          <div className="mt-3 overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-gray-500 text-xs border-b border-gray-800">
                  <th className="text-left py-2 pr-4">客户</th>
                  <th className="text-left py-2 pr-4">套餐</th>
                  <th className="text-left py-2 pr-4">激活时间</th>
                  <th className="text-left py-2">状态</th>
                </tr>
              </thead>
              <tbody>
                {history.map((h) => (
                  <tr key={h.id} className="border-b border-gray-800/60">
                    <td className="py-2 pr-4 text-gray-200">{h.customer || '—'}</td>
                    <td className="py-2 pr-4 text-gray-400 capitalize">{h.plan || '—'}</td>
                    <td className="py-2 pr-4 text-gray-400">
                      {h.activated_at ? new Date(h.activated_at).toLocaleString() : '—'}
                    </td>
                    <td className="py-2">
                      {h.is_active ? (
                        <span className="text-[10px] px-2 py-0.5 bg-emerald-900/40 rounded-full text-emerald-400">当前</span>
                      ) : (
                        <span className="text-[10px] px-2 py-0.5 bg-gray-800 rounded-full text-gray-400">历史</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
