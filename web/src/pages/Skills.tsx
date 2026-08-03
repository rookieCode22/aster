import { useEffect, useState, useCallback, useMemo } from 'react';
import { skills as skillsApi, type Skill } from '../api/client';

export default function Skills() {
  const [list, setList] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [filter, setFilter] = useState('');

  // Create form
  const [cName, setCName] = useState('');
  const [cDesc, setCDesc] = useState('');
  const [cTags, setCTags] = useState('');
  const [cContent, setCContent] = useState('');

  const fetchSkills = useCallback(async () => {
    try {
      setList(await skillsApi.list());
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSkills();
  }, [fetchSkills]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!cName.trim() || !cContent.trim()) return;
    setError('');
    try {
      await skillsApi.create({
        name: cName.trim(),
        description: cDesc.trim(),
        instructions: cContent.trim(),
        tags: cTags.split(',').map((t) => t.trim()).filter(Boolean),
      });
      setShowCreate(false);
      setCName('');
      setCDesc('');
      setCTags('');
      setCContent('');
      fetchSkills();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleDelete = async (name: string) => {
    try {
      await skillsApi.delete(name);
      setList((prev) => prev.filter((s) => s.name !== name));
    } catch (err: any) {
      setError(err.message);
    }
  };

  // 从现有技能的 tags 动态汇总出可筛选的标签集
  const allTags = useMemo(() => {
    const set = new Set<string>();
    list.forEach((s) => (s.tags ?? []).forEach((t) => set.add(t)));
    return Array.from(set).sort();
  }, [list]);

  const filtered = filter ? list.filter((s) => (s.tags ?? []).includes(filter)) : list;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Security Skills</h1>
          <p className="text-gray-400 mt-1">{filtered.length} skills · {list.length} total</p>
        </div>
        <button
          onClick={() => setShowCreate(!showCreate)}
          className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-lg transition text-sm"
        >
          {showCreate ? 'Cancel' : '+ New Skill'}
        </button>
      </div>

      {error && (
        <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded-lg px-3 py-2">{error}</div>
      )}

      {/* Create Form */}
      {showCreate && (
        <form onSubmit={handleCreate} className="space-y-3 bg-gray-900/50 border border-gray-800 rounded-xl p-5">
          <h3 className="text-gray-200 font-medium">Create Custom Skill</h3>
          <input
            type="text"
            placeholder="Skill name"
            value={cName}
            onChange={(e) => setCName(e.target.value)}
            className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition text-sm"
            required
          />
          <input
            type="text"
            placeholder="Short description"
            value={cDesc}
            onChange={(e) => setCDesc(e.target.value)}
            className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition text-sm"
          />
          <input
            type="text"
            placeholder="Tags (逗号分隔，如 recon, web)"
            value={cTags}
            onChange={(e) => setCTags(e.target.value)}
            className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition text-sm"
          />
          <textarea
            placeholder="Skill content (SKILL.md format)"
            value={cContent}
            onChange={(e) => setCContent(e.target.value)}
            rows={8}
            className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition text-sm font-mono resize-y"
            required
          />
          <button
            type="submit"
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-lg transition text-sm"
          >
            Save Skill
          </button>
        </form>
      )}

      {/* Category Filter */}
      <div className="flex flex-wrap gap-2">
        <button
          onClick={() => setFilter('')}
          className={`px-3 py-1 rounded-full text-xs font-medium transition ${
            !filter ? 'bg-emerald-600 text-white' : 'bg-gray-800 text-gray-400 hover:text-gray-200'
          }`}
        >
          All
        </button>
        {allTags.map((tag) => (
          <button
            key={tag}
            onClick={() => setFilter(tag)}
            className={`px-3 py-1 rounded-full text-xs font-medium transition ${
              filter === tag ? 'bg-emerald-600 text-white' : 'bg-gray-800 text-gray-400 hover:text-gray-200'
            }`}
          >
            {tag}
          </button>
        ))}
      </div>

      {/* Skills Grid */}
      {loading ? (
        <p className="text-gray-500 text-center py-8">Loading skills...</p>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <p className="text-4xl mb-3">🔧</p>
          <p>No skills found{filter ? ` in "${filter}"` : ''}.</p>
        </div>
      ) : (
        <div className="grid gap-3">
          {filtered.map((skill) => (
            <div
              key={skill.name}
              className="flex items-start justify-between bg-gray-900/50 border border-gray-800 rounded-xl px-5 py-4 hover:border-gray-700 transition group"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <h3 className="text-gray-200 font-medium truncate">{skill.name}</h3>
                  {(skill.tags ?? []).map((tag) => (
                    <span key={tag} className="text-[10px] px-2 py-0.5 bg-gray-800 rounded-full text-gray-400">
                      {tag}
                    </span>
                  ))}
                  {skill.source === 'custom' ? (
                    <span className="text-[10px] px-2 py-0.5 bg-emerald-900/40 rounded-full text-emerald-400 uppercase">
                      custom
                    </span>
                  ) : (
                    <span className="text-[10px] px-2 py-0.5 bg-sky-900/40 rounded-full text-sky-400 uppercase">
                      built-in
                    </span>
                  )}
                </div>
                <p className="text-gray-500 text-xs mt-1 line-clamp-2">{skill.description}</p>
              </div>
              {skill.deletable && (
                <button
                  onClick={() => handleDelete(skill.name)}
                  className="opacity-0 group-hover:opacity-100 px-3 py-1 text-red-400 hover:bg-red-900/30 rounded text-sm transition ml-4 shrink-0"
                >
                  Delete
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
