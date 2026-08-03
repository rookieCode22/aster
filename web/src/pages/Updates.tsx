import { useEffect, useState, useCallback } from 'react';
import { updates, type UpdateCheckResult, type UpdateItem, type InstalledModule } from '../api/client';

export default function Updates() {
  const [data, setData] = useState<UpdateCheckResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [applying, setApplying] = useState<string>(''); // filename being applied

  const fetch = useCallback(async () => {
    try {
      setData(await updates.check());
      setError('');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetch(); }, [fetch]);

  const handleApply = async (item: UpdateItem) => {
    setApplying(item.filename);
    setError('');
    try {
      const res = await updates.apply(item.filename);
      setError(`✅ 已应用 ${res.module_id} → v${res.version}`);
      await fetch();
    } catch (err: any) {
      setError(err.message || '应用失败');
    } finally {
      setApplying('');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">内容更新</h1>
          <p className="text-gray-400 mt-1">检查并应用技能 / 规则内容包 (.astm)</p>
        </div>
        <button
          onClick={() => { setLoading(true); fetch(); }}
          disabled={loading}
          className="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white font-medium rounded-lg transition text-sm disabled:opacity-50"
        >
          {loading ? '检查中…' : '重新检查'}
        </button>
      </div>

      {error && (
        <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded-lg px-3 py-2">{error}</div>
      )}

      {/* Available updates */}
      <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
        <h2 className="text-gray-200 font-medium">可用更新</h2>
        {loading ? (
          <p className="text-gray-500 text-sm py-4">加载中…</p>
        ) : !data || data.updates.length === 0 ? (
          <p className="text-gray-500 text-sm py-4">当前没有可用更新，已是最新内容。</p>
        ) : (
          <div className="mt-3 space-y-2">
            {data.updates.map((item) => (
              <div
                key={item.filename}
                className="flex items-center justify-between bg-gray-800/40 border border-gray-700/60 rounded-lg px-4 py-3"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-gray-200 font-medium">{item.module_id}</span>
                    <span className="text-[10px] px-2 py-0.5 bg-emerald-900/40 rounded-full text-emerald-400">
                      v{item.version}
                    </span>
                    <span className="text-[10px] px-2 py-0.5 bg-gray-800 rounded-full text-gray-400 uppercase">
                      {item.asset_type}
                    </span>
                    <span className="text-[10px] text-gray-500">{item.count} 项内容</span>
                  </div>
                  {item.description && (
                    <p className="text-gray-500 text-xs mt-1 truncate">{item.description}</p>
                  )}
                </div>
                <button
                  onClick={() => handleApply(item)}
                  disabled={applying === item.filename}
                  className="ml-4 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm rounded-lg transition shrink-0"
                >
                  {applying === item.filename ? '应用中…' : '应用'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Installed modules */}
      <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-5">
        <h2 className="text-gray-200 font-medium">已安装内容</h2>
        {loading ? (
          <p className="text-gray-500 text-sm py-4">加载中…</p>
        ) : !data || data.installed.length === 0 ? (
          <p className="text-gray-500 text-sm py-4">暂无已安装内容包。</p>
        ) : (
          <div className="mt-3 overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-gray-500 text-xs border-b border-gray-800">
                  <th className="text-left py-2 pr-4">模块</th>
                  <th className="text-left py-2 pr-4">类型</th>
                  <th className="text-left py-2 pr-4">版本</th>
                  <th className="text-left py-2">内容数</th>
                </tr>
              </thead>
              <tbody>
                {data.installed.map((m: InstalledModule) => (
                  <tr key={m.module_id} className="border-b border-gray-800/60">
                    <td className="py-2 pr-4 text-gray-200">{m.module_id}</td>
                    <td className="py-2 pr-4 text-gray-400 capitalize">{m.asset_type}</td>
                    <td className="py-2 pr-4 text-gray-400">v{m.version}</td>
                    <td className="py-2 text-gray-400">{m.count}</td>
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
