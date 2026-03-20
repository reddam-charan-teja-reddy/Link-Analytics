import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import toast from 'react-hot-toast';
import { groupsApi } from '../lib/services';

export default function GroupsPage() {
  const [groups, setGroups] = useState([]);
  const [loading, setLoading] = useState(true);
  const [newGroupName, setNewGroupName] = useState('');
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const items = await groupsApi.list();
      setGroups(items || []);
    } catch (err) {
      toast.error(err.message || 'Failed to load groups');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function createGroup(e) {
    e.preventDefault();
    if (!newGroupName.trim()) return;

    setCreating(true);
    try {
      const created = await groupsApi.create({ name: newGroupName.trim() });
      setGroups((prev) => [created, ...prev]);
      setNewGroupName('');
      toast.success('Group created');
    } catch (err) {
      toast.error(err.message || 'Could not create group');
    } finally {
      setCreating(false);
    }
  }

  async function renameGroup(group) {
    const nextName = window.prompt('Rename group', group.name);
    if (!nextName || !nextName.trim() || nextName.trim() === group.name) return;

    try {
      const updated = await groupsApi.update(group.id, { name: nextName.trim() });
      setGroups((prev) => prev.map((item) => (String(item.id) === String(group.id) ? updated : item)));
      toast.success('Group renamed');
    } catch (err) {
      toast.error(err.message || 'Could not rename group');
    }
  }

  async function deleteGroup(group) {
    if (!window.confirm(`Delete group "${group.name}"?`)) return;

    try {
      await groupsApi.remove(group.id);
      setGroups((prev) => prev.filter((item) => String(item.id) !== String(group.id)));
      toast.success('Group deleted');
    } catch (err) {
      toast.error(err.message || 'Could not delete group');
    }
  }

  return (
    <div className="stack-xl">
      <section className="hero-panel hero-groups">
        <div>
          <p className="eyebrow">Groups hub</p>
          <h2 className="hero-title">Manage campaign groups and jump into group-level work</h2>
          <p className="muted">Open any group to manage links, sources, analytics, and link operations in one focused page.</p>
        </div>
      </section>

      <section className="panel">
        <h3>Create group</h3>
        <form className="inline-form compact" onSubmit={createGroup}>
          <label>
            Group name
            <input
              type="text"
              value={newGroupName}
              onChange={(e) => setNewGroupName(e.target.value)}
              placeholder="spring-campaign"
              required
            />
          </label>
          <button className="btn primary" type="submit" disabled={creating}>
            {creating ? 'Creating...' : 'Create group'}
          </button>
        </form>
      </section>

      <section className="panel">
        <h3>All groups</h3>
        {loading ? (
          <p className="muted">Loading groups...</p>
        ) : !groups.length ? (
          <p className="muted">No groups yet. Create your first one above.</p>
        ) : (
          <ul className="group-list">
            {groups.map((group) => (
              <li key={group.id} className="group-row">
                <div>
                  <p className="group-name">{group.name}</p>
                  <p className="muted small">{group.link_count ?? 0} links</p>
                </div>
                <div className="row-actions">
                  <Link to={`/groups/${group.id}`} className="btn secondary">Open</Link>
                  <button className="btn ghost" onClick={() => renameGroup(group)}>Rename</button>
                  <button className="btn danger" onClick={() => deleteGroup(group)}>Delete</button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
