import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import toast from 'react-hot-toast';
import { analyticsApi, groupsApi, linksApi } from '../lib/services';

function TopSources({ items }) {
  return (
    <article className="panel">
      <h3>Top sources</h3>
      {!items?.length ? (
        <p className="muted small">No source activity yet.</p>
      ) : (
        <ul className="breakdown-list">
          {items.slice(0, 7).map((item) => (
            <li key={item.label}>
              <span>{item.label}</span>
              <strong>{item.clicks}</strong>
            </li>
          ))}
        </ul>
      )}
    </article>
  );
}

export default function LinkDetailPage() {
  const { linkId } = useParams();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [link, setLink] = useState(null);
  const [sources, setSources] = useState([]);
  const [allGroups, setAllGroups] = useState([]);
  const [linkGroups, setLinkGroups] = useState([]);
  const [selectedGroup, setSelectedGroup] = useState('');

  const [sourceName, setSourceName] = useState('');
  const [creatingSource, setCreatingSource] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const [summary, setSummary] = useState(null);
  const [sourceBreakdown, setSourceBreakdown] = useState([]);
  const [browsers, setBrowsers] = useState([]);
  const [locations, setLocations] = useState([]);
  const [recent, setRecent] = useState([]);
  const [clickTrend, setClickTrend] = useState([]);
  const [granularity, setGranularity] = useState('day');

  const availableGroups = useMemo(() => {
    const assigned = new Set((linkGroups || []).map((g) => String(g.id)));
    return allGroups.filter((group) => !assigned.has(String(group.id)));
  }, [allGroups, linkGroups]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const from = new Date();
      from.setDate(from.getDate() - 6);

      const [linkData, sourceData, groups, groupsForLink, analyticsSummary, sourceDataBreakdown, browserData, locData, recentData, trendData] = await Promise.all([
        linksApi.getOne(linkId),
        linksApi.listSources(linkId),
        groupsApi.list(),
        groupsApi.forLink(linkId),
        analyticsApi.summary(linkId),
        analyticsApi.sources(linkId),
        analyticsApi.browsers(linkId),
        analyticsApi.locations(linkId),
        analyticsApi.recent(linkId, 12),
        analyticsApi.clicks(linkId, { from: from.toISOString().slice(0, 10), granularity }),
      ]);

      setLink(linkData);
      setSources(sourceData || []);
      setAllGroups(groups || []);
      setLinkGroups(groupsForLink || []);
      setSummary(analyticsSummary || null);
      setSourceBreakdown(sourceDataBreakdown || []);
      setBrowsers(browserData || []);
      setLocations(locData || []);
      setRecent(recentData || []);
      setClickTrend((trendData || []).map((p) => ({
        date: new Date(p.timestamp).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
        clicks: p.clicks,
      })));
    } catch (err) {
      toast.error(err.message || 'Failed to load link details');
      navigate('/links');
    } finally {
      setLoading(false);
    }
  }, [granularity, linkId, navigate]);

  useEffect(() => {
    load();
  }, [load]);

  async function toggleActive() {
    try {
      const updated = await linksApi.update(linkId, { is_active: !link.is_active });
      setLink(updated);
      toast.success(updated.is_active ? 'Link activated' : 'Link paused');
    } catch (err) {
      toast.error(err.message || 'Failed to update status');
    }
  }

  async function renameLink() {
    const currentTitle = link?.title || '';
    const nextTitle = window.prompt('Rename link', currentTitle);

    if (nextTitle === null) return;
    if (!nextTitle.trim()) {
      toast.error('Title cannot be empty');
      return;
    }
    if (nextTitle.trim() === currentTitle) return;

    try {
      const updated = await linksApi.update(linkId, { title: nextTitle.trim() });
      setLink(updated);
      toast.success('Link renamed');
    } catch (err) {
      toast.error(err.message || 'Failed to rename link');
    }
  }

  async function deleteLink() {
    if (!window.confirm('Delete this link?')) return;
    setDeleting(true);
    try {
      await linksApi.remove(linkId);
      toast.success('Link deleted');
      navigate('/links');
    } catch (err) {
      toast.error(err.message || 'Delete failed');
      setDeleting(false);
    }
  }

  async function createSource(e) {
    e.preventDefault();
    if (!sourceName.trim()) return;

    setCreatingSource(true);
    try {
      const created = await linksApi.createSource(linkId, { source_name: sourceName.trim() });
      setSources((prev) => [created, ...prev]);
      setSourceName('');
      toast.success('Source added');
    } catch (err) {
      toast.error(err.message || 'Failed to add source');
    } finally {
      setCreatingSource(false);
    }
  }

  async function removeSource(sourceId) {
    if (!window.confirm('Delete source link?')) return;
    try {
      await linksApi.removeSource(linkId, sourceId);
      setSources((prev) => prev.filter((item) => item.id !== sourceId));
      toast.success('Source removed');
    } catch (err) {
      toast.error(err.message || 'Failed to delete source');
    }
  }

  async function addToGroup() {
    if (!selectedGroup) return;
    try {
      await groupsApi.addLink(selectedGroup, linkId);
      const next = await groupsApi.forLink(linkId);
      setLinkGroups(next || []);
      setSelectedGroup('');
      toast.success('Added to group');
    } catch (err) {
      toast.error(err.message || 'Could not add to group');
    }
  }

  async function removeFromGroup(groupId) {
    try {
      await groupsApi.removeLink(groupId, linkId);
      setLinkGroups((prev) => prev.filter((group) => String(group.id) !== String(groupId)));
      toast.success('Removed from group');
    } catch (err) {
      toast.error(err.message || 'Could not remove from group');
    }
  }

  async function copy(text) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success('Copied');
    } catch (err) {
      if (window.isSecureContext !== true) {
        toast.error('Clipboard requires HTTPS or localhost.');
        return;
      }
      if (err?.name === 'NotAllowedError') {
        toast.error('Clipboard permission denied by browser.');
        return;
      }
      toast.error(err?.message || 'Copy failed');
    }
  }

  if (loading) {
    return <p className="muted">Loading details...</p>;
  }

  if (!link) return null;

  const shortUrl = link.short_url || `${window.location.origin}/${link.hash}`;

  return (
    <div className="stack-xl">
      <div className="back-row">
        <Link to="/links" className="btn ghost">← Back to home</Link>
      </div>

      <section className="hero-panel">
        <div>
          <p className="eyebrow">Link details</p>
          <h2 className="hero-title">{link.title || 'Untitled link'}</h2>
          <p className="mono clickable" onClick={() => copy(shortUrl)}>{shortUrl}</p>
          <p className="muted small truncate">{link.original_url}</p>
          <p className="muted small">Manual source format: <code>{shortUrl}?src=channel-name</code></p>
        </div>
        <div className="row-actions">
          <span className={link.is_active ? 'pill success' : 'pill warn'}>{link.is_active ? 'Active' : 'Paused'}</span>
          <button className="btn secondary" onClick={renameLink}>Rename link</button>
          <button className="btn secondary" onClick={toggleActive}>Toggle status</button>
          <button className="btn danger" onClick={deleteLink} disabled={deleting}>{deleting ? 'Deleting...' : 'Delete link'}</button>
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h3>Performance overview (last 7 days)</h3>
          <label>
            Granularity
            <select value={granularity} onChange={(e) => setGranularity(e.target.value)}>
              <option value="day">Day</option>
              <option value="hour">Hour</option>
            </select>
          </label>
        </div>
        <div className="summary-grid">
          <div><span>Total clicks</span><strong>{summary?.total_clicks ?? 0}</strong></div>
          <div><span>Unique visitors</span><strong>{summary?.unique_visitors ?? 0}</strong></div>
          <div><span>Bot clicks</span><strong>{summary?.bot_clicks ?? 0}</strong></div>
        </div>
        <div className="chart-wrap">
          <ResponsiveContainer width="100%" height={230}>
            <LineChart data={clickTrend}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis allowDecimals={false} />
              <Tooltip />
              <Line type="monotone" dataKey="clicks" stroke="var(--brand)" strokeWidth={2.5} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className="board two-col">
        <article className="panel">
          <h3>Source links</h3>
          <form className="inline-form compact" onSubmit={createSource}>
            <label>
              Source name
              <input
                type="text"
                value={sourceName}
                onChange={(e) => setSourceName(e.target.value)}
                placeholder="twitter"
                required
              />
            </label>
            <button className="btn primary" type="submit" disabled={creatingSource}>
              {creatingSource ? 'Adding...' : 'Add source'}
            </button>
          </form>

          <ul className="source-list">
            {sources.map((source) => {
              const sourceUrl = source.short_url || `${window.location.origin}/${source.hash}`;
              return (
                <li key={source.id}>
                  <div>
                    <p className="source-name">{source.source_name}</p>
                    <p className="mono small">{sourceUrl}</p>
                  </div>
                  <div className="row-actions">
                    <button className="btn ghost" onClick={() => copy(sourceUrl)}>Copy</button>
                    <button className="btn danger" onClick={() => removeSource(source.id)}>Delete</button>
                  </div>
                </li>
              );
            })}
            {!sources.length && <li className="muted">No sources yet.</li>}
          </ul>
        </article>

        <article className="panel">
          <h3>Groups</h3>
          <div className="inline-form compact">
            <label>
              Add to group
              <select value={selectedGroup} onChange={(e) => setSelectedGroup(e.target.value)}>
                <option value="">Select group</option>
                {availableGroups.map((group) => (
                  <option key={group.id} value={group.id}>{group.name}</option>
                ))}
              </select>
            </label>
            <button className="btn secondary" onClick={addToGroup}>Add</button>
          </div>

          <div className="chip-wrap">
            {linkGroups.map((group) => (
              <div key={group.id} className="chip">
                <span>{group.name}</span>
                <button className="chip-danger" onClick={() => removeFromGroup(group.id)} aria-label={`Remove ${group.name}`}>
                  ×
                </button>
              </div>
            ))}
            {!linkGroups.length && <p className="muted small">No groups assigned.</p>}
          </div>

          <TopSources items={sourceBreakdown} />
        </article>
      </section>

      <section className="panel">
        <h3>Recent click activity</h3>
        <p className="muted small">Browser and country data may not be fully accurate (derived from headers/IP).</p>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Source</th>
                <th>Referrer</th>
                <th>Browser</th>
                <th>Country</th>
              </tr>
            </thead>
            <tbody>
              {recent.length ? recent.map((item, idx) => (
                <tr key={`${item.clicked_at}-${idx}`}>
                  <td>{new Date(item.clicked_at).toLocaleString()}</td>
                  <td>{item.source}</td>
                  <td className="truncate-cell">{item.referer}</td>
                  <td>{item.browser}</td>
                  <td>{item.country}</td>
                </tr>
              )) : (
                <tr>
                  <td colSpan={5} className="muted">No recent clicks yet.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="board two-col detail-snapshots">
          <article className="panel inset">
            <h4>Browser snapshot</h4>
            <ul className="breakdown-list">
              {(browsers || []).slice(0, 5).map((item) => (
                <li key={item.label}><span>{item.label}</span><strong>{item.clicks}</strong></li>
              ))}
              {!browsers.length && <li className="muted">No browser data.</li>}
            </ul>
          </article>

          <article className="panel inset">
            <h4>Country snapshot</h4>
            <ul className="breakdown-list">
              {(locations || []).slice(0, 5).map((item) => (
                <li key={item.label}><span>{item.label}</span><strong>{item.clicks}</strong></li>
              ))}
              {!locations.length && <li className="muted">No country data.</li>}
            </ul>
          </article>
        </div>
      </section>
    </div>
  );
}
