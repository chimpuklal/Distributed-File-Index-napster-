// Distributed File Index - Dashboard Logic (Vanilla JS)

document.addEventListener('DOMContentLoaded', () => {
  // Elements
  const serverStatus = document.getElementById('server-status');
  const statusText = document.getElementById('status-text');

  const statPeers = document.getElementById('stat-peers');
  const statFiles = document.getElementById('stat-files');
  const statRegistrations = document.getElementById('stat-registrations');

  const registerForm = document.getElementById('register-form');
  const regPeerId = document.getElementById('reg-peer-id');
  const regFiles = document.getElementById('reg-files');
  const btnRegister = document.getElementById('btn-register');
  const registerAlert = document.getElementById('register-alert');

  const searchForm = document.getElementById('search-form');
  const searchInput = document.getElementById('search-input');
  const btnSearch = document.getElementById('btn-search');
  const btnClear = document.getElementById('btn-clear');
  const searchAlert = document.getElementById('search-alert');
  const searchResults = document.getElementById('search-results');
  const resultFilename = document.getElementById('result-filename');
  const resultCount = document.getElementById('result-count');
  const resultPeersList = document.getElementById('result-peers-list');

  const peersContainer = document.getElementById('peers-container');
  const peersCountBadge = document.getElementById('peers-count-badge');
  const btnRefresh = document.getElementById('btn-refresh');

  // Utility: Show Alert
  function showAlert(element, message, type = 'error') {
    element.className = `alert alert-${type}`;
    element.textContent = message;
    element.classList.remove('hidden');
  }

  function hideAlert(element) {
    element.className = 'alert hidden';
    element.textContent = '';
  }

  // 1. Health Check
  async function checkHealth() {
    try {
      const res = await fetch('/health', { cache: 'no-store' });
      if (res.ok) {
        const data = await res.json();
        if (data.status === 'ok') {
          serverStatus.className = 'status-badge status-online';
          statusText.textContent = 'Server Online';
          return true;
        }
      }
      throw new Error('Unhealthy status');
    } catch (err) {
      serverStatus.className = 'status-badge status-offline';
      statusText.textContent = 'Server Offline';
      return false;
    }
  }

  // 2. Fetch Overview Statistics
  async function fetchStats() {
    try {
      const res = await fetch('/api/v1/stats', { cache: 'no-store' });
      if (res.ok) {
        const data = await res.json();
        statPeers.textContent = data.total_peers ?? 0;
        statFiles.textContent = data.total_unique_files ?? 0;
        statRegistrations.textContent = data.total_registrations ?? 0;
      }
    } catch (err) {
      console.error('Failed to load stats:', err);
    }
  }

  // 3. Fetch Registered Peers List
  async function fetchPeers() {
    try {
      const res = await fetch('/api/v1/peers', { cache: 'no-store' });
      if (!res.ok) {
        throw new Error(`HTTP error ${res.status}`);
      }
      const data = await res.json();
      const peers = data.peers || [];

      // Update badge
      peersCountBadge.textContent = `${peers.length} ${peers.length === 1 ? 'Peer' : 'Peers'}`;

      if (peers.length === 0) {
        peersContainer.innerHTML = `
          <div class="empty-state">
            <div class="empty-state-title">No Peers Registered</div>
            <div class="empty-state-desc">Use the form above or the CLI client to register peers and their files.</div>
          </div>
        `;
        return;
      }

      // Render peer cards
      let html = '';
      peers.forEach(peer => {
        const fileChips = (peer.files || [])
          .map(file => `<span class="file-chip">${escapeHtml(file)}</span>`)
          .join('');

        html += `
          <div class="peer-card">
            <div class="peer-card-header">
              <div class="peer-title">
                <span>⚡</span>
                <span>${escapeHtml(peer.peer_id)}</span>
              </div>
              <span class="badge">${(peer.files || []).length} ${(peer.files || []).length === 1 ? 'file' : 'files'}</span>
            </div>
            <div class="files-chip-container">
              ${fileChips}
            </div>
          </div>
        `;
      });
      peersContainer.innerHTML = html;
    } catch (err) {
      peersContainer.innerHTML = `
        <div class="empty-state">
          <div class="empty-state-title" style="color: var(--error-text);">Failed to load registered peers</div>
          <div class="empty-state-desc">Check if the central server is running.</div>
        </div>
      `;
    }
  }

  // Helper: Refresh all dashboard data
  async function refreshDashboard() {
    await checkHealth();
    await Promise.all([fetchStats(), fetchPeers()]);
  }

  // 4. Register a Peer
  registerForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    hideAlert(registerAlert);

    const peerId = regPeerId.value.trim();
    const rawFiles = regFiles.value.trim();

    if (!peerId) {
      showAlert(registerAlert, 'Peer ID is required and cannot be empty.', 'error');
      regPeerId.focus();
      return;
    }

    if (!rawFiles) {
      showAlert(registerAlert, 'Please enter at least one filename.', 'error');
      regFiles.focus();
      return;
    }

    // Split and clean filenames
    const files = rawFiles
      .split(',')
      .map(f => f.trim())
      .filter(f => f.length > 0);

    if (files.length === 0) {
      showAlert(registerAlert, 'Please enter valid, non-empty filenames.', 'error');
      regFiles.focus();
      return;
    }

    btnRegister.disabled = true;
    btnRegister.textContent = 'Registering...';

    try {
      const res = await fetch('/api/v1/peers/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          peer_id: peerId,
          files: files
        })
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || `Server responded with status ${res.status}`);
      }

      showAlert(
        registerAlert,
        `Peer "${data.peer_id}" successfully registered with ${data.files.length} file(s).`,
        'success'
      );

      // Reset input fields
      regPeerId.value = '';
      regFiles.value = '';

      // Refresh dashboard data
      await refreshDashboard();
    } catch (err) {
      showAlert(registerAlert, err.message, 'error');
    } finally {
      btnRegister.disabled = false;
      btnRegister.textContent = 'Register Peer';
    }
  });

  // 5. Search Files
  searchForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    hideAlert(searchAlert);
    searchResults.classList.add('hidden');

    const fileName = searchInput.value.trim();
    if (!fileName) {
      showAlert(searchAlert, 'Please enter a filename to search for.', 'error');
      searchInput.focus();
      return;
    }

    btnSearch.disabled = true;
    btnSearch.textContent = 'Searching...';

    try {
      const res = await fetch(`/api/v1/search?file=${encodeURIComponent(fileName)}`);
      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || `Server responded with status ${res.status}`);
      }

      resultFilename.textContent = data.file;
      const peers = data.peers || [];

      resultCount.textContent = `${peers.length} ${peers.length === 1 ? 'peer' : 'peers'} found`;

      if (peers.length === 0) {
        resultPeersList.innerHTML = `
          <div class="search-empty-state">
            No peers currently have "<strong>${escapeHtml(data.file)}</strong>" registered in the index.
          </div>
        `;
      } else {
        resultPeersList.innerHTML = peers.map(p => `
          <div class="peer-result-row">
            <span class="peer-id-badge">${escapeHtml(p.peer_id)}</span>
            <span class="peer-status-tag">Ready for transfer</span>
          </div>
        `).join('');
      }

      searchResults.classList.remove('hidden');
    } catch (err) {
      showAlert(searchAlert, err.message, 'error');
    } finally {
      btnSearch.disabled = false;
      btnSearch.textContent = 'Search';
    }
  });

  // Clear Search
  btnClear.addEventListener('click', () => {
    searchInput.value = '';
    hideAlert(searchAlert);
    searchResults.classList.add('hidden');
    searchInput.focus();
  });

  // Refresh Button
  btnRefresh.addEventListener('click', async () => {
    btnRefresh.disabled = true;
    await refreshDashboard();
    btnRefresh.disabled = false;
  });

  // Helper: Escape HTML
  function escapeHtml(str) {
    if (!str) return '';
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // Initial dashboard load
  refreshDashboard();

  // Periodic health check every 10 seconds
  setInterval(checkHealth, 10000);
});
