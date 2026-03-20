import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import toast from 'react-hot-toast';
import { groupsApi, linksApi } from '../lib/services';

export default function GroupDetailPage() {
  const { groupId } = useParams();
  const navigate = useNavigate();

  const [group, setGroup] = useState(null);
  const [links, setLinks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [url, setUrl] = useState('');
  const [title, setTitle] = useState('');
  const [creating, setCreating] = useState(false);

  const [query, setQuery] = useState('');
  const [sourceFilter, setSourceFilter] = useState('');
  const [sourceNameForAll, setSourceNameForAll] = useState('');
  const [creatingSourcesForGroup, setCreatingSourcesForGroup] = useState(false);
  const [selectedLinkIds, setSelectedLinkIds] = useState([]);
  const [runningBatch, setRunningBatch] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [groups, groupedLinks] = await Promise.all([
        groupsApi.list(),
        linksApi.list({ groupId, source: sourceFilter.trim() || undefined }),
      ]);

      const current = (groups || []).find((item) => String(item.id) === String(groupId));
      if (!current) {
        toast.error('Group not found');
        navigate('/groups');
        return;
      }

      setGroup(current);
      setLinks(groupedLinks || []);
    } catch (err) {
      toast.error(err.message || 'Failed to load group page');
      navigate('/groups');
    } finally {
      setLoading(false);
    }
  }, [groupId, navigate, sourceFilter]);

  useEffect(() => {
    load();
  }, [load]);

  const filteredLinks = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const searched = !needle
      ? links
      : links.filter((item) =>
      [item.title, item.original_url, item.hash]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(needle))
    );

    return searched;
  }, [links, query]);

  const selectedVisibleCount = useMemo(() => {
    const visibleIDs = new Set(filteredLinks.map((item) => String(item.id)));
    return selectedLinkIds.filter((id) => visibleIDs.has(String(id))).length;
  }, [selectedLinkIds, filteredLinks]);
  const sourceModeEnabled = Boolean(sourceFilter.trim());

  function withSourceParam(shortUrl, sourceName) {
    if (!sourceName) return shortUrl;
    const separator = shortUrl.includes('?') ? '&' : '?';
    return `${shortUrl}${separator}src=${encodeURIComponent(sourceName)}`;
  }

  useEffect(() => {
    setSelectedLinkIds((prev) => prev.filter((id) => links.some((item) => String(item.id) === String(id))));
  }, [links]);

  async function renameGroup() {
    if (!group) return;

    const nextName = window.prompt('Rename group', group.name);
    if (!nextName || !nextName.trim() || nextName.trim() === group.name) return;

    try {
      const updated = await groupsApi.update(group.id, { name: nextName.trim() });
      setGroup(updated);
      toast.success('Group renamed');
    } catch (err) {
      toast.error(err.message || 'Could not rename group');
    }
  }

  async function removeLink(linkId) {
    if (!window.confirm('Remove this link from group?')) return;

    try {
      await groupsApi.removeLink(groupId, linkId);
      setLinks((prev) => prev.filter((item) => String(item.id) !== String(linkId)));
      toast.success('Link removed from group');
    } catch (err) {
      toast.error(err.message || 'Could not remove link');
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

  function toggleSelectLink(linkId) {
    setSelectedLinkIds((prev) => {
      const exists = prev.some((id) => String(id) === String(linkId));
      if (exists) return prev.filter((id) => String(id) !== String(linkId));
      return [...prev, linkId];
    });
  }

  function selectAllVisibleLinks() {
    setSelectedLinkIds((prev) => {
      const ids = new Set(prev.map((id) => String(id)));
      filteredLinks.forEach((item) => ids.add(String(item.id)));
      return Array.from(ids);
    });
  }

  function clearSelection() {
    setSelectedLinkIds([]);
  }

  async function batchRemoveFromGroup() {
    if (!selectedLinkIds.length) return;
    if (!window.confirm(`Remove ${selectedLinkIds.length} selected links from this group?`)) return;

    setRunningBatch(true);
    try {
      const results = await Promise.allSettled(
        selectedLinkIds.map((id) => groupsApi.removeLink(groupId, id))
      );
      const successCount = results.filter((r) => r.status === 'fulfilled').length;
      const failedCount = results.length - successCount;

      if (successCount > 0) {
        setLinks((prev) => prev.filter((item) => !selectedLinkIds.some((id) => String(id) === String(item.id))));
        setSelectedLinkIds([]);
      }

      if (successCount) toast.success(`${successCount} link(s) removed`);
      if (failedCount) toast.error(`${failedCount} link(s) failed to remove`);
    } catch (err) {
      toast.error(err.message || 'Batch remove failed');
    } finally {
      setRunningBatch(false);
    }
  }

  async function createSourceForEntireGroup(e) {
    e.preventDefault();
    if (!sourceNameForAll.trim()) return;

    setCreatingSourcesForGroup(true);
    try {
      const result = await linksApi.batchCreateSources({
        source_name: sourceNameForAll.trim(),
        scope_type: 'group',
        scope_id: groupId,
      });

      const created = result?.created_count || 0;
      const skipped = result?.skipped_count || 0;
      toast.success(`Created ${created} source links${skipped ? `, skipped ${skipped}` : ''}`);
      setSourceNameForAll('');
      await load();
    } catch (err) {
      toast.error(err.message || 'Could not create source links for group');
    } finally {
      setCreatingSourcesForGroup(false);
    }
  }

  async function handleCreateLink(e) {
    e.preventDefault();
    if (!url.trim()) return;

    setCreating(true);
    try {
      const created = await linksApi.create({
        original_url: url.trim(),
        title: title.trim() || undefined,
      });

      await groupsApi.addLink(groupId, created.id);
      setLinks((prev) => [created, ...prev]);
      setUrl('');
      setTitle('');
      setShowCreateModal(false);
      toast.success('Link created in this group');
    } catch (err) {
      toast.error(err.message || 'Could not create link in this group');
    } finally {
      setCreating(false);
    }
  }

  if (loading) return <p className="muted">Loading group...</p>;
  if (!group) return null;

  return (
    <div className="stack-xl">
      <div className="back-row">
        <Link to="/groups" className="btn ghost">← Back to groups</Link>
      </div>

      <section className="hero-panel hero-group-detail">
        <div>
          <p className="eyebrow">Group detail</p>
          <h2 className="hero-title">{group.name}</h2>
          <p className="muted">Manage all links in this group, then open each link for source-level and analytics operations.</p>
        </div>
        <div className="row-actions">
          <span className="pill neutral">{links.length} links</span>
          <button className="btn primary" onClick={() => setShowCreateModal(true)}>Create link in this group</button>
          <button className="btn secondary" onClick={renameGroup}>Rename group</button>
        </div>
      </section>

      <section className="panel">
        <h3>Group source actions</h3>
        <form className="inline-form compact" onSubmit={createSourceForEntireGroup}>
          <label>
            Add this source to all links in this group
            <input
              type="text"
              value={sourceNameForAll}
              onChange={(e) => setSourceNameForAll(e.target.value)}
              placeholder="linkedin"
              required
            />
          </label>
          <button className="btn primary" type="submit" disabled={creatingSourcesForGroup}>
            {creatingSourcesForGroup ? 'Applying...' : 'Apply to all links'}
          </button>
        </form>
        <p className="muted small">This uses batch source creation for the current group and skips already-existing source links.</p>
      </section>

      <section className="panel group-search-panel">
        <h3>Search and source filter</h3>
        <div className="filter-grid group-filter-grid">
          <label>
            Search links
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="title, destination URL, or hash"
            />
          </label>
          <label>
            Filter by source parameter
            <input
              type="text"
              value={sourceFilter}
              onChange={(e) => setSourceFilter(e.target.value)}
              placeholder="linkedin"
            />
          </label>
        </div>
        <p className="muted small">Default (empty source filter) shows normal group links. With a source filter, only matching source links are shown.</p>
      </section>

      <section className="panel">
        <div className="section-head">
          <h3>Links in group</h3>
          {sourceModeEnabled && (
            <div className="row-actions compact">
              <span className="mode-pill">
                Source mode: {sourceFilter.trim()} ({filteredLinks.length})
              </span>
              <button className="btn ghost" onClick={() => setSourceFilter('')}>Show normal links</button>
            </div>
          )}
        </div>
        <div className="batch-toolbar" role="group" aria-label="Batch actions">
          <div className="row-actions">
            <button className="btn ghost" onClick={selectAllVisibleLinks} disabled={!filteredLinks.length || runningBatch}>Select all visible</button>
            <button className="btn ghost" onClick={clearSelection} disabled={!selectedLinkIds.length || runningBatch}>Clear</button>
            <span className="muted small">Selected: {selectedVisibleCount}</span>
          </div>
          <div className="row-actions">
            <button className="btn danger" onClick={batchRemoveFromGroup} disabled={!selectedLinkIds.length || runningBatch}>Remove selected from group</button>
          </div>
        </div>
        {!filteredLinks.length ? (
          <p className="muted">No links in this group.</p>
        ) : (
          <ul className="link-list">
            {filteredLinks.map((item) => {
              const shortUrl = item.short_url || `${window.location.origin}/${item.hash}`;
              const displayUrl = sourceModeEnabled ? withSourceParam(shortUrl, sourceFilter.trim()) : shortUrl;
              return (
                <li key={item.id} className="link-row">
                  <label className="row-select">
                    <input
                      type="checkbox"
                      checked={selectedLinkIds.some((id) => String(id) === String(item.id))}
                      onChange={() => toggleSelectLink(item.id)}
                      aria-label={`Select ${item.title || item.hash}`}
                    />
                  </label>
                  <div className="link-main">
                    <p className="link-title-like">{item.title || 'Untitled link'}</p>
                    <p className="mono">{displayUrl}</p>
                    <p className="muted small truncate">{item.original_url}</p>
                    {sourceModeEnabled && (
                      <p className="muted small">Showing source-tag view for: <strong>{sourceFilter.trim()}</strong></p>
                    )}
                    <div className="row-actions compact">
                      <span className="pill neutral">Clicks: {item.total_clicks || 0}</span>
                      <span className="pill neutral">
                        Last visit: {item.last_clicked_at ? new Date(item.last_clicked_at).toLocaleString() : 'Never'}
                      </span>
                    </div>
                  </div>
                  <div className="row-actions">
                    <button className="btn ghost" onClick={() => copy(displayUrl)}>Copy</button>
                    <Link to={`/links/${item.id}`} className="btn secondary">Open analytics</Link>
                    <button className="btn danger" onClick={() => removeLink(item.id)}>Remove</button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      {showCreateModal && (
        <div className="modal-overlay" role="dialog" aria-modal="true" aria-label="Create link in group">
          <div className="modal-panel">
            <div className="modal-head">
              <h3>Create link in {group.name}</h3>
              <button className="btn ghost" onClick={() => setShowCreateModal(false)}>Close</button>
            </div>

            <form className="inline-form modal-form" onSubmit={handleCreateLink}>
              <label>
                Destination URL
                <input
                  type="url"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://example.com/landing-page"
                  required
                />
              </label>

              <label>
                Link title (optional)
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Group launch page"
                />
              </label>

              <div className="tip-box">
                <p className="small">This link is automatically added to <strong>{group.name}</strong>. Share it directly, or append <code>?src=channel</code> when you need source attribution.</p>
              </div>

              <button className="btn primary" type="submit" disabled={creating}>
                {creating ? 'Creating link...' : 'Create link'}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
