import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import toast from 'react-hot-toast';
import StatCard from '../components/StatCard';
import { groupsApi, linksApi } from '../lib/services';

const NEW_GROUP_VALUE = '__new_group__';

export default function LinksPage() {
  const [loading, setLoading] = useState(true);
  const [links, setLinks] = useState([]);
  const [groups, setGroups] = useState([]);
  const [selectedGroup, setSelectedGroup] = useState('');
  const [sourceFilter, setSourceFilter] = useState('');
  const [sortPreset, setSortPreset] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showGuide, setShowGuide] = useState(() => localStorage.getItem('guideDismissed') !== 'true');
  const [selectedLinkIds, setSelectedLinkIds] = useState([]);
  const [batchGroupId, setBatchGroupId] = useState('');
  const [runningBatch, setRunningBatch] = useState(false);

  const [url, setUrl] = useState('');
  const [title, setTitle] = useState('');
  const [groupAtCreate, setGroupAtCreate] = useState('');
  const [newGroupAtCreate, setNewGroupAtCreate] = useState('');
  const [creating, setCreating] = useState(false);

  const [newGroupName, setNewGroupName] = useState('');
  const [creatingGroup, setCreatingGroup] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [nextLinks, nextGroups] = await Promise.all([
        linksApi.list({
          groupId: selectedGroup || undefined,
          source: sourceFilter.trim() || undefined,
        }),
        groupsApi.list(),
      ]);
      setLinks(nextLinks || []);
      setGroups(nextGroups || []);
    } catch (err) {
      toast.error(err.message || 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, [selectedGroup, sourceFilter]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const total = links.length;
  const active = useMemo(() => links.filter((item) => item.is_active).length, [links]);
  const totalClicks = useMemo(
    () => links.reduce((sum, item) => sum + (item.total_clicks || 0), 0),
    [links]
  );

  const visibleLinks = useMemo(() => {
    const next = [...links];
    if (sortPreset === 'top-clicked') {
      next.sort((a, b) => (b.total_clicks || 0) - (a.total_clicks || 0));
      return next.slice(0, 5);
    }

    if (sortPreset === 'recent-visited') {
      next.sort((a, b) => {
        const aTs = a.last_clicked_at ? new Date(a.last_clicked_at).getTime() : 0;
        const bTs = b.last_clicked_at ? new Date(b.last_clicked_at).getTime() : 0;
        return bTs - aTs;
      });
      return next.slice(0, 5);
    }

    return next;
  }, [links, sortPreset]);

  const selectedVisibleCount = useMemo(() => {
    const visibleIDs = new Set(visibleLinks.map((item) => String(item.id)));
    return selectedLinkIds.filter((id) => visibleIDs.has(String(id))).length;
  }, [selectedLinkIds, visibleLinks]);

  useEffect(() => {
    setSelectedLinkIds((prev) => prev.filter((id) => links.some((item) => String(item.id) === String(id))));
  }, [links]);

  const shouldCreateGroupAtCreate = groupAtCreate === NEW_GROUP_VALUE;

  async function handleCreateLink(e) {
    e.preventDefault();
    setCreating(true);

    try {
      let chosenGroupID = groupAtCreate;
      if (shouldCreateGroupAtCreate) {
        if (!newGroupAtCreate.trim()) {
          toast.error('Enter a new group name or choose an existing group.');
          setCreating(false);
          return;
        }

        const createdGroup = await groupsApi.create({ name: newGroupAtCreate.trim() });
        setGroups((prev) => [createdGroup, ...prev]);
        chosenGroupID = createdGroup.id;
      }

      const created = await linksApi.create({
        original_url: url.trim(),
        title: title.trim() || undefined,
      });

      if (chosenGroupID) {
        await groupsApi.addLink(chosenGroupID, created.id);
      }

      setLinks((prev) => [created, ...prev]);
      setUrl('');
      setTitle('');
      setGroupAtCreate('');
      setNewGroupAtCreate('');
      setShowCreateModal(false);
      toast.success('Link created');
    } catch (err) {
      toast.error(err.message || 'Failed to create link');
    } finally {
      setCreating(false);
    }
  }

  async function handleCreateGroup(e) {
    e.preventDefault();
    if (!newGroupName.trim()) return;
    setCreatingGroup(true);
    try {
      const created = await groupsApi.create({ name: newGroupName.trim() });
      setGroups((prev) => [created, ...prev]);
      setNewGroupName('');
      toast.success('Group created');
    } catch (err) {
      toast.error(err.message || 'Failed to create group');
    } finally {
      setCreatingGroup(false);
    }
  }

  async function handleDeleteGroup(group) {
    if (!window.confirm(`Delete group \"${group.name}\"?`)) return;
    try {
      await groupsApi.remove(group.id);
      setGroups((prev) => prev.filter((item) => item.id !== group.id));
      if (String(selectedGroup) === String(group.id)) setSelectedGroup('');
      toast.success('Group deleted');
    } catch (err) {
      toast.error(err.message || 'Failed to delete group');
    }
  }

  async function handleRenameGroup(group) {
    const nextName = window.prompt('Rename group', group.name);
    if (!nextName || !nextName.trim() || nextName.trim() === group.name) return;

    try {
      const updated = await groupsApi.update(group.id, { name: nextName.trim() });
      setGroups((prev) => prev.map((item) => (String(item.id) === String(group.id) ? updated : item)));
      toast.success('Group renamed');
    } catch (err) {
      toast.error(err.message || 'Failed to rename group');
    }
  }

  async function copy(urlToCopy) {
    try {
      await navigator.clipboard.writeText(urlToCopy);
      toast.success('Copied link');
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
      visibleLinks.forEach((item) => ids.add(String(item.id)));
      return Array.from(ids);
    });
  }

  function clearSelection() {
    setSelectedLinkIds([]);
  }

  async function batchDeleteSelected() {
    if (!selectedLinkIds.length) return;
    if (!window.confirm(`Delete ${selectedLinkIds.length} selected links?`)) return;

    setRunningBatch(true);
    try {
      const idsToDelete = [...selectedLinkIds];
      const results = await Promise.allSettled(idsToDelete.map((id) => linksApi.remove(id)));
      const successCount = results.filter((r) => r.status === 'fulfilled').length;
      const failedCount = results.length - successCount;
      const failedIDs = results
        .map((result, index) => (result.status === 'rejected' ? idsToDelete[index] : null))
        .filter(Boolean);

      if (successCount > 0) {
        setLinks((prev) => prev.filter((item) => !idsToDelete.some((id) => String(id) === String(item.id))));
      }

      setSelectedLinkIds(failedIDs);

      if (successCount) toast.success(`${successCount} link(s) deleted`);
      if (failedCount) toast.error(`${failedCount} link(s) failed to delete`);
    } catch (err) {
      toast.error(err.message || 'Batch delete failed');
    } finally {
      setRunningBatch(false);
    }
  }

  async function batchAddToGroup() {
    if (!selectedLinkIds.length || !batchGroupId) return;

    setRunningBatch(true);
    try {
      const results = await Promise.allSettled(
        selectedLinkIds.map((id) => groupsApi.addLink(batchGroupId, id))
      );
      const successCount = results.filter((r) => r.status === 'fulfilled').length;
      const failedCount = results.length - successCount;

      if (successCount) toast.success(`${successCount} link(s) added to group`);
      if (failedCount) toast.error(`${failedCount} link(s) failed to add`);
    } catch (err) {
      toast.error(err.message || 'Batch add to group failed');
    } finally {
      setRunningBatch(false);
    }
  }

  function dismissGuide() {
    setShowGuide(false);
    localStorage.setItem('guideDismissed', 'true');
  }

  return (
    <div className="stack-xl">
      <section className="hero-panel">
        <div>
          <p className="eyebrow">Control room</p>
          <h2 className="hero-title">Launch links that look simple and perform like campaigns</h2>
          <p className="muted">Create fast, group clearly, and monitor real traffic quality in one place.</p>
          <div className="row-actions">
            <button className="btn primary" onClick={() => setShowCreateModal(true)}>Create new link</button>
            <Link to="/groups" className="btn secondary">Open groups workspace</Link>
          </div>
        </div>
        <div className="stats-grid">
          <StatCard label="Total Links" value={total} tone="teal" />
          <StatCard label="Active" value={active} tone="green" />
          <StatCard label="Total Clicks" value={totalClicks} tone="sand" />
        </div>
      </section>

      {showGuide && (
        <section className="panel guide-panel guide-panel-strong" role="status" aria-live="polite">
          <div>
            <p className="eyebrow">Quick start</p>
            <h3>How to run campaigns with clean tracking</h3>
            <ol className="guide-list">
              <li>Create link from Home. Add a title to keep dashboard readable.</li>
              <li>Assign an existing group or create a brand-new group in the same modal.</li>
              <li>Every link includes a unique hash and works immediately without any source parameter.</li>
              <li>Share short links with source tags like <code>?src=instagram</code>, <code>?src=email</code>, and <code>?src=partner-a</code>.</li>
              <li>Use filters + presets to focus on best or most recently visited links.</li>
              <li>Open a link for source operations and 7-day analytics. Open a group page for grouped link operations.</li>
            </ol>
          </div>
          <div className="guide-actions">
            <Link to="/groups" className="btn secondary">Go to groups</Link>
            <button className="btn ghost" onClick={dismissGuide}>Dismiss guide</button>
          </div>
        </section>
      )}

      <section className="board two-col">
        <article className="panel links-filter-panel">
          <h3>Link filters and ranking</h3>

          <div className="filter-row">
            <div className="filter-grid">
              <label>
                Filter by group
                <select value={selectedGroup} onChange={(e) => setSelectedGroup(e.target.value)}>
                  <option value="">All groups</option>
                  {groups.map((group) => (
                    <option key={group.id} value={group.id}>{group.name}</option>
                  ))}
                </select>
              </label>

              <label>
                Filter by source
                <input
                  type="text"
                  value={sourceFilter}
                  onChange={(e) => setSourceFilter(e.target.value)}
                  placeholder="email, twitter, reddit"
                />
              </label>

              <label>
                Quick sort presets
                <select value={sortPreset} onChange={(e) => setSortPreset(e.target.value)}>
                  <option value="all">All links (default)</option>
                  <option value="top-clicked">Top 5 most clicked</option>
                  <option value="recent-visited">5 most recently visited</option>
                </select>
              </label>
            </div>
          </div>
        </article>

        <article className="panel">
          <h3>Manage groups</h3>
          <form className="inline-form compact" onSubmit={handleCreateGroup}>
            <label>
              New group
              <input
                type="text"
                value={newGroupName}
                onChange={(e) => setNewGroupName(e.target.value)}
                placeholder="newsletter"
                required
              />
            </label>
            <button className="btn primary" type="submit" disabled={creatingGroup}>
              {creatingGroup ? 'Adding...' : 'Add'}
            </button>
          </form>

          <div className="chip-wrap">
            {groups.map((group) => (
              <div key={group.id} className="chip chip-actions">
                <Link to={`/groups/${group.id}`} className="chip-link">{group.name}</Link>
                <button className="chip-mini" onClick={() => handleRenameGroup(group)}>Rename</button>
                <button className="chip-danger" onClick={() => handleDeleteGroup(group)} aria-label={`Delete ${group.name}`}>Delete</button>
              </div>
            ))}
            {!groups.length && <p className="muted small">No groups yet. Create one now, or add it while creating a link.</p>}
          </div>
        </article>
      </section>

      <section className="panel">
        <h3>Your links</h3>
        <div className="batch-toolbar" role="group" aria-label="Batch actions">
          <div className="row-actions">
            <button className="btn ghost" onClick={selectAllVisibleLinks} disabled={!visibleLinks.length || runningBatch}>Select all visible</button>
            <button className="btn ghost" onClick={clearSelection} disabled={!selectedLinkIds.length || runningBatch}>Clear</button>
            <span className="muted small">Selected: {selectedVisibleCount}</span>
          </div>
          <div className="row-actions">
            <select
              value={batchGroupId}
              onChange={(e) => setBatchGroupId(e.target.value)}
              aria-label="Batch add to group"
              disabled={!groups.length || runningBatch}
            >
              <option value="">Add selected to group...</option>
              {groups.map((group) => (
                <option key={group.id} value={group.id}>{group.name}</option>
              ))}
            </select>
            <button className="btn secondary" onClick={batchAddToGroup} disabled={!selectedLinkIds.length || !batchGroupId || runningBatch}>Apply</button>
            <button className="btn danger" onClick={batchDeleteSelected} disabled={!selectedLinkIds.length || runningBatch}>Delete selected</button>
          </div>
        </div>
        {loading ? (
          <p className="muted">Loading links...</p>
        ) : visibleLinks.length === 0 ? (
          <p className="muted">No links match current filters.</p>
        ) : (
          <ul className="link-list" aria-label="Links">
            {visibleLinks.map((item) => {
              const shortUrl = item.short_url || `${window.location.origin}/${item.hash}`;
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
                    <Link to={`/links/${item.id}`} className="link-title">{item.title || 'Untitled link'}</Link>
                    <p className="mono">{shortUrl}</p>
                    <p className="muted small truncate">{item.original_url}</p>
                    <div className="row-actions compact">
                      <span className="pill neutral">Clicks: {item.total_clicks || 0}</span>
                      <span className="pill neutral">
                        Last visit: {item.last_clicked_at ? new Date(item.last_clicked_at).toLocaleString() : 'Never'}
                      </span>
                    </div>
                    <p className="muted small">
                      Unique hash works directly. Optional source shortcut: <code>{shortUrl}?src=channel-name</code>
                    </p>
                  </div>
                  <div className="row-actions">
                    <span className={item.is_active ? 'pill success' : 'pill warn'}>
                      {item.is_active ? 'Active' : 'Inactive'}
                    </span>
                    <button className="btn ghost" onClick={() => copy(shortUrl)}>Copy</button>
                    <Link to={`/links/${item.id}`} className="btn secondary">Open</Link>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      {showCreateModal && (
        <div className="modal-overlay" role="dialog" aria-modal="true" aria-label="Create link modal">
          <div className="modal-panel">
            <div className="modal-head">
              <h3>Create new link</h3>
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
                  placeholder="Product launch page"
                />
              </label>

              <label>
                Assign group (optional)
                <select
                  value={groupAtCreate}
                  onChange={(e) => {
                    setGroupAtCreate(e.target.value);
                    if (e.target.value !== NEW_GROUP_VALUE) setNewGroupAtCreate('');
                  }}
                >
                  <option value="">No group</option>
                  <option value={NEW_GROUP_VALUE}>+ Create new group</option>
                  {groups.map((group) => (
                    <option key={group.id} value={group.id}>{group.name}</option>
                  ))}
                </select>
              </label>

              {!groups.length && (
                <p className="muted small">No groups yet. Choose <strong>+ Create new group</strong> if you want to attach this link immediately.</p>
              )}

              {shouldCreateGroupAtCreate && (
                <label>
                  New group name
                  <input
                    type="text"
                    value={newGroupAtCreate}
                    onChange={(e) => setNewGroupAtCreate(e.target.value)}
                    placeholder="product-launch-q2"
                    required
                  />
                </label>
              )}

              <div className="tip-box">
                <p className="small">
                  Your unique hash is ready to use immediately. Add <code>?src=your-source</code> only when you want source-level attribution.
                </p>
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
