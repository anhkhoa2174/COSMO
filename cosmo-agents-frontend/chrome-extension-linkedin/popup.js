// === Environment Config ===
const ENVIRONMENTS = {
  staging: {
    apiBase: 'https://api-cosmoagents.rockship.xyz',
    webApp: 'https://cosmoagents.rockship.xyz',
    cookieDomain: 'rockship.xyz'
  },
  production: {
    apiBase: 'https://api.cosmo.rockship.co',
    webApp: 'https://cosmo.rockship.co',
    cookieDomain: 'cosmo.rockship.co'
  },
  local: {
    apiBase: 'http://localhost:8081',
    webApp: 'http://localhost:3000',
    cookieDomain: 'localhost'
  }
};

// === State ===
let currentEnv = 'staging'; // Default to staging for testing
let authToken = null;
let scrapedConnections = [];

// === DOM Elements ===
const envSelect = document.getElementById('envSelect');
const loginStatusEl = document.getElementById('loginStatus');
const loginStatusText = document.getElementById('loginStatusText');
const loginBtn = document.getElementById('loginBtn');
const apiBaseInput = document.getElementById('apiBase');
const tokenInput = document.getElementById('token');
const statusEl = document.getElementById('status');
const successResultEl = document.getElementById('successResult');
const contactLinkEl = document.getElementById('contactLink');

// New Contact elements
const createContactBtn = document.getElementById('createContactBtn');
const useAiMapNewInput = document.getElementById('useAiMapNew');
const useAiEnrichNewInput = document.getElementById('useAiEnrichNew');
const newContactSection = document.getElementById('newContactSection');

// Update Contact elements
const contactIdInput = document.getElementById('contactId');
const useAiInput = document.getElementById('useAi');
const sendBtn = document.getElementById('sendBtn');
const updateContactSection = document.getElementById('updateContactSection');

// Connections elements
const maxScrollsInput = document.getElementById('maxScrolls');
const scrapeConnectionsBtn = document.getElementById('scrapeConnectionsBtn');
const sendConnectionsBtn = document.getElementById('sendConnectionsBtn');
const connectionsListEl = document.getElementById('connectionsList');
const connectionsSection = document.getElementById('connectionsSection');

// Gen Message elements
const genMessageSection = document.getElementById('genMessageSection');
const toneSelect = document.getElementById('toneSelect');
const purposeSelect = document.getElementById('purposeSelect');
const customNoteInput = document.getElementById('customNote');
const generateMsgBtn = document.getElementById('generateMsgBtn');
const generatedMessagesEl = document.getElementById('generatedMessages');

// Mode elements
const modeNewContactBtn = document.getElementById('modeNewContact');
const modeUpdateContactBtn = document.getElementById('modeUpdateContact');
const modeConnectionsBtn = document.getElementById('modeConnections');
const modeGenMessageBtn = document.getElementById('modeGenMessage');

// === Utility Functions ===
function setStatus(text, type = 'info') {
  statusEl.textContent = text;
  statusEl.className = `status ${type}`;
  // Hide success result when showing new status
  if (successResultEl) {
    successResultEl.style.display = 'none';
  }
}

function showContactLink(contactId) {
  const env = getEnvConfig();
  if (successResultEl && contactLinkEl) {
    contactLinkEl.href = `${env.webApp}/contacts/${contactId}`;
    successResultEl.style.display = 'block';
  }
}

function getEnvConfig() {
  return ENVIRONMENTS[currentEnv];
}

function getApiBase() {
  const customApi = apiBaseInput.value.trim();
  return customApi || getEnvConfig().apiBase;
}

function getToken() {
  const manualToken = tokenInput.value.trim();
  return manualToken || authToken;
}

// === Auth Functions ===
async function checkAuth() {
  const env = getEnvConfig();
  loginStatusText.textContent = 'Checking login...';
  loginBtn.style.display = 'none';

  try {
    // Try to get cookies from the specific domain
    const cookies = await chrome.cookies.getAll({ domain: env.cookieDomain });
    console.log('[Auth] Cookies found:', cookies.map(c => c.name));

    // COSMO uses these cookie names for auth
    const tokenCookie = cookies.find(c =>
      c.name === 'agent_access_token' ||  // COSMO primary token
      c.name === 'access_token' ||         // COSMO secondary token
      c.name === 'next-auth.session-token' ||
      c.name === '__Secure-next-auth.session-token' ||
      c.name === 'token' ||
      c.name === 'accessToken' ||
      c.name === 'auth_token'
    );

    if (tokenCookie) {
      console.log('[Auth] Found token cookie:', tokenCookie.name);
      const isValid = await validateToken(tokenCookie.value);
      if (isValid) {
        authToken = tokenCookie.value;
        setLoginStatus(true);
        return true;
      } else {
        console.log('[Auth] Token validation failed');
      }
    }

    // Also try getting cookies from the full URL domain
    const fullDomainCookies = await chrome.cookies.getAll({ url: env.webApp });
    console.log('[Auth] Full domain cookies:', fullDomainCookies.map(c => c.name));

    const fullDomainToken = fullDomainCookies.find(c =>
      c.name === 'agent_access_token' || c.name === 'access_token'
    );

    if (fullDomainToken) {
      console.log('[Auth] Found full domain token:', fullDomainToken.name);
      const isValid = await validateToken(fullDomainToken.value);
      if (isValid) {
        authToken = fullDomainToken.value;
        setLoginStatus(true);
        return true;
      }
    }

    const token = await getTokenFromLocalStorage();
    if (token) {
      const isValid = await validateToken(token);
      if (isValid) {
        authToken = token;
        setLoginStatus(true);
        return true;
      }
    }

    setLoginStatus(false);
    return false;
  } catch (err) {
    console.error('[Auth] Error checking auth:', err);
    setLoginStatus(false);
    return false;
  }
}

async function getTokenFromLocalStorage() {
  try {
    const env = getEnvConfig();
    const stored = await chrome.storage.local.get(['authToken', 'authTokenExpiry']);
    if (stored.authToken && stored.authTokenExpiry > Date.now()) {
      console.log('[Auth] Using cached token');
      return stored.authToken;
    }

    // Try to find a Cosmo tab and get token from cookies via js-cookie
    const tabs = await chrome.tabs.query({ url: `${env.webApp}/*` });
    console.log('[Auth] Found Cosmo tabs:', tabs.length);

    if (tabs.length > 0) {
      const result = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func: () => {
          // Try to get from document.cookie (js-cookie stores here)
          const cookies = document.cookie.split(';').reduce((acc, cookie) => {
            const [key, value] = cookie.trim().split('=');
            acc[key] = value;
            return acc;
          }, {});

          console.log('[Cosmo Tab] Cookies:', Object.keys(cookies));

          // COSMO token names
          const token = cookies['agent_access_token'] ||
                       cookies['access_token'] ||
                       localStorage.getItem('token') ||
                       localStorage.getItem('accessToken') ||
                       localStorage.getItem('agent_access_token') ||
                       sessionStorage.getItem('token') ||
                       sessionStorage.getItem('accessToken');
          return token;
        }
      });

      if (result && result[0] && result[0].result) {
        console.log('[Auth] Got token from Cosmo tab');
        await chrome.storage.local.set({
          authToken: result[0].result,
          authTokenExpiry: Date.now() + 3600000
        });
        return result[0].result;
      }
    }

    return null;
  } catch (err) {
    console.error('[Auth] Error getting token from localStorage:', err);
    return null;
  }
}

async function validateToken(token) {
  try {
    const apiBase = getApiBase();
    const res = await fetch(`${apiBase}/v1/users/me`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    });
    return res.ok;
  } catch (err) {
    console.error('[Auth] Token validation error:', err);
    return false;
  }
}

function setLoginStatus(isLoggedIn) {
  if (isLoggedIn) {
    loginStatusEl.className = 'login-status logged-in';
    loginStatusText.textContent = '✓ Connected to Cosmo';
    loginBtn.style.display = 'none';
  } else {
    loginStatusEl.className = 'login-status logged-out';
    loginStatusText.textContent = '✗ Not logged in';
    loginBtn.style.display = 'inline-block';
  }
}

// === Settings ===
function saveSettings() {
  chrome.storage.local.set({
    env: currentEnv,
    apiBase: apiBaseInput.value.trim(),
    contactId: contactIdInput?.value?.trim() || '',
    token: tokenInput.value.trim(),
    useAi: useAiInput?.checked ?? true,
    useAiMapNew: useAiMapNewInput?.checked ?? true,
    useAiEnrichNew: useAiEnrichNewInput?.checked ?? true,
    maxScrolls: maxScrollsInput?.value || '10',
  });
}

function loadSettings() {
  chrome.storage.local.get(['env', 'apiBase', 'contactId', 'token', 'useAi', 'useAiMapNew', 'useAiEnrichNew', 'maxScrolls'], (data) => {
    currentEnv = data.env || 'staging';
    envSelect.value = currentEnv;

    // Clear custom API URL field - use env default instead
    let apiBase = data.apiBase || '';
    // Only clear truly invalid URLs (old format with wrong domain)
    if (apiBase.includes('apicosmoagents') && !apiBase.includes('api-cosmoagents')) {
      apiBase = ''; // Clear invalid URL, will use env default
      chrome.storage.local.set({ apiBase: '' });
      console.log('[Settings] Cleared invalid API URL');
    }
    apiBaseInput.value = apiBase;

    if (contactIdInput) contactIdInput.value = data.contactId || '';
    tokenInput.value = data.token || '';
    if (useAiInput) useAiInput.checked = data.useAi ?? true;
    if (useAiMapNewInput) useAiMapNewInput.checked = data.useAiMapNew ?? true;
    if (useAiEnrichNewInput) useAiEnrichNewInput.checked = data.useAiEnrichNew ?? true;
    if (maxScrollsInput) maxScrollsInput.value = data.maxScrolls || '10';
  });
}

async function getActiveTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  return tab;
}

// === Event Listeners ===
envSelect.addEventListener('change', async () => {
  currentEnv = envSelect.value;
  saveSettings();
  await checkAuth();
});

loginBtn.addEventListener('click', () => {
  const env = getEnvConfig();
  chrome.tabs.create({ url: `${env.webApp}/auth/login` });
});

// === Mode Switching ===
function switchMode(mode) {
  // Reset all buttons
  modeNewContactBtn.classList.remove('active');
  modeUpdateContactBtn.classList.remove('active');
  modeConnectionsBtn.classList.remove('active');
  modeGenMessageBtn.classList.remove('active');

  // Hide all sections
  newContactSection.style.display = 'none';
  updateContactSection.style.display = 'none';
  connectionsSection.style.display = 'none';
  genMessageSection.style.display = 'none';

  // Show selected
  if (mode === 'new') {
    modeNewContactBtn.classList.add('active');
    newContactSection.style.display = 'block';
  } else if (mode === 'update') {
    modeUpdateContactBtn.classList.add('active');
    updateContactSection.style.display = 'block';
  } else if (mode === 'connections') {
    modeConnectionsBtn.classList.add('active');
    connectionsSection.style.display = 'block';
  } else if (mode === 'genmessage') {
    modeGenMessageBtn.classList.add('active');
    genMessageSection.style.display = 'block';
  }

  setStatus('');
}

modeNewContactBtn.addEventListener('click', () => switchMode('new'));
modeUpdateContactBtn.addEventListener('click', () => switchMode('update'));
modeConnectionsBtn.addEventListener('click', () => switchMode('connections'));
modeGenMessageBtn.addEventListener('click', () => switchMode('genmessage'));

// === Normalize LinkedIn URL ===
function normalizeLinkedInUrl(url) {
  if (!url) return '';
  // Remove query params and hash
  let normalized = url.split('?')[0].split('#')[0];
  // Remove trailing slash
  normalized = normalized.replace(/\/+$/, '');
  // Convert to lowercase for consistent matching
  return normalized.toLowerCase();
}

// === Check if Contact Already Exists ===
async function checkContactByLinkedInUrl(linkedinUrl) {
  const apiBase = getApiBase();
  const token = getToken();

  // Normalize the URL for consistent matching
  const normalizedUrl = normalizeLinkedInUrl(linkedinUrl);

  if (!token || !normalizedUrl) {
    return null;
  }

  try {
    console.log('[Search] Searching for LinkedIn URL:', normalizedUrl);

    // Search for contact with this LinkedIn URL
    const res = await fetch(`${apiBase}/v1/contacts/search`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        filter: {
          contact_information: normalizedUrl
        }
      }),
    });

    if (!res.ok) {
      console.log('[Search] First search failed:', res.status);
      return null;
    }

    const payload = await res.json();
    const contacts = payload?.data?.list || [];
    console.log('[Search] First search (contact_information) found:', contacts.length, 'contacts');

    // Check each contact to verify URL actually matches
    for (const item of contacts) {
      const contact = item.entity;
      const storedUrl = normalizeLinkedInUrl(contact?.contact_information || '');
      console.log('[Search] Checking contact:', contact?.name, 'stored URL:', storedUrl);

      // Only return if URLs actually match
      if (storedUrl === normalizedUrl) {
        console.log('[Search] URL match confirmed, returning contact');
        return contact;
      }
    }

    console.log('[Search] No matching contact found');
    return null;
  } catch (err) {
    console.error('[Popup] Error checking contact:', err);
    return null;
  }
}

function showAlreadyImported(contact) {
  const env = getEnvConfig();

  // Update status
  setStatus(`✓ Already imported: ${contact.name || 'Unknown'}`, 'success');

  // Show link to contact
  if (successResultEl && contactLinkEl) {
    contactLinkEl.href = `${env.webApp}/contacts`;
    contactLinkEl.textContent = 'View in Cosmo';
    successResultEl.style.display = 'block';
  }

  // Disable create button and update text
  if (createContactBtn) {
    createContactBtn.disabled = true;
    createContactBtn.innerHTML = '<span>✓ Already in Cosmo</span>';
    createContactBtn.classList.add('already-imported');
  }
}

function resetCreateButton() {
  if (createContactBtn) {
    createContactBtn.disabled = false;
    createContactBtn.innerHTML = '<span>Create New Contact</span>';
    createContactBtn.classList.remove('already-imported');
  }
  if (successResultEl) {
    successResultEl.style.display = 'none';
  }
}

// === Profile Scraping ===
async function scrapeCurrentTab() {
  const tab = await getActiveTab();
  console.log('[Popup] Active tab:', tab?.url);

  if (!tab?.id) {
    throw new Error('No active tab');
  }

  // Check if on LinkedIn profile page
  if (!tab.url?.includes('linkedin.com/in/')) {
    console.log('[Popup] Not on LinkedIn profile page. Current URL:', tab.url);
    throw new Error('Please navigate to a LinkedIn profile page first');
  }

  let response = null;

  // First try: send message to existing content script
  try {
    console.log('[Popup] Trying to send message to content script...');
    response = await chrome.tabs.sendMessage(tab.id, { type: 'SCRAPE_PROFILE' });
    console.log('[Popup] Content script response:', response);
  } catch (err) {
    console.log('[Popup] Content script not ready, will inject:', err.message);
  }

  // Second try: inject content script and send message again
  if (!response) {
    try {
      console.log('[Popup] Injecting content.js...');
      await chrome.scripting.executeScript({
        target: { tabId: tab.id },
        files: ['content.js'],
      });
      console.log('[Popup] Content script injected, sending message...');

      // Wait a bit for content script to initialize
      await new Promise(resolve => setTimeout(resolve, 100));

      response = await chrome.tabs.sendMessage(tab.id, { type: 'SCRAPE_PROFILE' });
      console.log('[Popup] Response after injection:', response);
    } catch (err) {
      console.error('[Popup] Failed to inject or communicate:', err);
      throw new Error(`Failed to scrape: ${err.message}`);
    }
  }

  if (!response?.ok) {
    console.log('[Popup] Response not OK:', response);
    throw new Error('Failed to scrape profile - no valid response');
  }
  return response;
}

// === NEW: Create Contact from LinkedIn Profile ===
async function createNewContact(scrapedData) {
  const apiBase = getApiBase();
  const token = getToken();
  const useAiMap = useAiMapNewInput?.checked ?? true;
  const useAiEnrich = useAiEnrichNewInput?.checked ?? true;

  if (!token) {
    throw new Error('Not logged in. Please login to Cosmo first.');
  }

  const profile = scrapedData.data || {};

  // Get name directly from profile (now uses 'name' field instead of first_name/last_name)
  const contactName = profile.name || '';

  // Get LinkedIn URL - normalize for consistent matching
  const rawLinkedinUrl = profile.source_url || profile.linkedin_url || '';
  const linkedinUrl = normalizeLinkedInUrl(rawLinkedinUrl);

  // Build raw_linkedin_data only with non-empty fields
  // Limit data size to avoid database index size limits
  const rawLinkedInData = {};
  if (profile.about) {
    rawLinkedInData.about = profile.about.slice(0, 500); // Limit about to 500 chars
  }
  if (profile.experience && profile.experience.length > 0) {
    // Only keep first 3 experiences, and limit description length
    rawLinkedInData.experience = profile.experience.slice(0, 3).map(exp => ({
      title: exp.title?.slice(0, 100) || '',
      company: exp.company?.slice(0, 100) || '',
      date_range: exp.date_range?.slice(0, 50) || '',
      location: exp.location?.slice(0, 50) || '',
      // Skip description to save space
    }));
  }
  if (profile.education) rawLinkedInData.education = profile.education;
  if (profile.skills) rawLinkedInData.skills = profile.skills;

  // Build contact data with 'name' field only (no first_name/last_name)
  // Email is left empty - no placeholder email generated
  const contactData = {
    name: contactName,
    linkedin_url: linkedinUrl,
    job_title: profile.headline || profile.title,
    company: profile.company,
    location: profile.location,
    source: 'LinkedIn',
  };

  // Only add raw_linkedin_data if it has content
  if (Object.keys(rawLinkedInData).length > 0) {
    contactData.raw_linkedin_data = rawLinkedInData;
  }

  // Remove empty fields
  Object.keys(contactData).forEach(key => {
    if (contactData[key] === undefined || contactData[key] === '' || contactData[key] === null) {
      delete contactData[key];
    }
  });

  console.log('[Popup] Creating contact with data:', JSON.stringify(contactData, null, 2));

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 45000);

  // Create contact via API
  const res = await fetch(`${apiBase}/v1/contacts`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(contactData),
    signal: controller.signal,
  });
  clearTimeout(timeoutId);

  const payload = await res.json().catch(() => ({}));
  console.log('[Popup] API response:', res.status, JSON.stringify(payload, null, 2));

  if (!res.ok) {
    if (res.status === 401) {
      authToken = null;
      await chrome.storage.local.remove(['authToken', 'authTokenExpiry']);
      setLoginStatus(false);
      throw new Error('Session expired. Please login again.');
    }
    if (res.status === 409) {
      // Contact already exists - try to find by LinkedIn URL
      throw new Error('Contact with this LinkedIn URL already exists');
    }
    const message = payload?.error?.message || payload?.message || res.statusText;
    throw new Error(message);
  }

  const contactId = payload?.data?.id;
  if (contactId && (useAiMap || useAiEnrich)) {
    try {
      if (useAiMap) {
        await applyAiMapping(contactId, scrapedData);
      }
    } catch (err) {
      console.warn('[Popup] AI mapping skipped:', err?.message || err);
    }
    try {
      if (useAiEnrich) {
        await enrichContact(contactId);
      }
    } catch (err) {
      console.warn('[Popup] AI enrichment skipped:', err?.message || err);
    }
  }

  return payload;
}

createContactBtn.addEventListener('click', async () => {
  setStatus('');
  createContactBtn.disabled = true;

  try {
    saveSettings();
    setStatus('Scraping LinkedIn profile...');

    const scrapedData = await scrapeCurrentTab();
    console.log('[Popup] Scraped data:', scrapedData);

    const profile = scrapedData.data || {};
    console.log('[Popup] Profile extracted:', profile);

    if (!profile.name && !profile.headline) {
      console.log('[Popup] Empty profile data - extraction failed');
      setStatus('Warning: Could not extract profile data. Try refreshing the LinkedIn page.', 'warning');
      return;
    }

    setStatus(`Found: ${profile.name || 'Unknown'}. Creating contact...`);

    const result = await createNewContact(scrapedData);
    const contact = result?.data;

    if (contact) {
      // Show as already imported since we just created it
      showAlreadyImported(contact);
      setStatus(`Contact created: ${profile.name}`, 'success');
    }
  } catch (err) {
    console.error('[Popup] Error:', err);
    setStatus(`Error: ${err.message || err}`, 'error');
  } finally {
    createContactBtn.disabled = false;
  }
});

// === Update Existing Contact ===
async function updateExistingContact(scrapedData) {
  const apiBase = getApiBase();
  const contactId = contactIdInput.value.trim();
  const token = getToken();
  const useAi = useAiInput?.checked ?? false;

  if (!contactId) {
    throw new Error('Contact ID is required');
  }

  if (!token) {
    throw new Error('Not logged in. Please login to Cosmo first.');
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 45000);

  const res = await fetch(`${apiBase}/v1/contacts/${contactId}/extract-from-extension`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      source_url: scrapedData.data?.source_url,
      data: scrapedData.data || {},
      raw_text: scrapedData.raw_text || '',
      use_ai: useAi,
    }),
    signal: controller.signal,
  });
  clearTimeout(timeoutId);

  const payload = await res.json().catch(() => ({}));

  if (!res.ok) {
    if (res.status === 401) {
      authToken = null;
      await chrome.storage.local.remove(['authToken', 'authTokenExpiry']);
      setLoginStatus(false);
      throw new Error('Session expired. Please login again.');
    }
    const message = payload?.error?.message || payload?.message || res.statusText;
    throw new Error(message);
  }

  return payload;
}

async function applyAiMapping(contactId, scrapedData) {
  const apiBase = getApiBase();
  const token = getToken();

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 45000);

  const res = await fetch(`${apiBase}/v1/contacts/${contactId}/extract-from-extension`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      source_url: scrapedData.data?.source_url,
      data: scrapedData.data || {},
      raw_text: scrapedData.raw_text || '',
      use_ai: true,
    }),
    signal: controller.signal,
  });
  clearTimeout(timeoutId);

  if (!res.ok) {
    const payload = await res.json().catch(() => ({}));
    const message = payload?.error?.message || payload?.message || res.statusText;
    throw new Error(message);
  }
}

async function enrichContact(contactId) {
  const apiBase = getApiBase();
  const token = getToken();

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 45000);

  const res = await fetch(`${apiBase}/v1/contacts/${contactId}/enrich`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ force_refresh: false }),
    signal: controller.signal,
  });
  clearTimeout(timeoutId);

  if (!res.ok) {
    const payload = await res.json().catch(() => ({}));
    const message = payload?.error?.message || payload?.message || res.statusText;
    throw new Error(message);
  }
}

sendBtn.addEventListener('click', async () => {
  setStatus('');
  sendBtn.disabled = true;

  try {
    saveSettings();
    setStatus('Scraping profile...');
    const scrapedData = await scrapeCurrentTab();

    console.log('[Popup] Scraped data:', scrapedData);
    const profile = scrapedData.data || {};
    const scrapedFields = Object.keys(profile).filter(k => profile[k]).join(', ');

    if (!profile.name && !profile.headline && !profile.company) {
      setStatus('Warning: No data extracted. Check console and try AI mode.', 'warning');
      return;
    }

    setStatus(`Scraped: ${scrapedFields || 'raw text'}. Updating...`);
    const result = await updateExistingContact(scrapedData);

    const fieldsAdded = result?.data?.fields_added || [];
    const contactId = contactIdInput.value.trim();

    if (fieldsAdded.length > 0) {
      setStatus(`Success! Updated: ${fieldsAdded.join(', ')}`, 'success');
    } else {
      setStatus(`Sent, but no new fields added (may already exist)`, 'success');
    }

    showContactLink(contactId);
  } catch (err) {
    console.error('[Popup] Error:', err);
    setStatus(`Error: ${err.message || err}`, 'error');
  } finally {
    sendBtn.disabled = false;
  }
});

// === Connections Scraping ===
async function scrapeConnections() {
  const tab = await getActiveTab();
  if (!tab?.id) {
    throw new Error('No active tab');
  }

  if (!tab.url?.includes('/mynetwork/invite-connect/connections')) {
    throw new Error('Please navigate to LinkedIn Connections page first');
  }

  const maxScrolls = parseInt(maxScrollsInput.value) || 10;

  let response = null;
  try {
    response = await chrome.tabs.sendMessage(tab.id, {
      type: 'SCRAPE_CONNECTIONS',
      maxScrolls: maxScrolls
    });
  } catch (err) {
    // content script not injected yet
  }

  if (!response) {
    await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      files: ['content-connections.js'],
    });
    response = await chrome.tabs.sendMessage(tab.id, {
      type: 'SCRAPE_CONNECTIONS',
      maxScrolls: maxScrolls
    });
  }

  if (!response?.ok) {
    throw new Error('Failed to scrape connections');
  }
  return response;
}

function displayConnections(connections) {
  connectionsListEl.innerHTML = '';

  if (!connections || connections.length === 0) {
    connectionsListEl.innerHTML = '<p class="no-data">No connections found</p>';
    return;
  }

  // Add select all checkbox
  const header = document.createElement('div');
  header.className = 'conn-header';
  header.innerHTML = `
    <label class="select-all">
      <input type="checkbox" id="selectAll" checked />
      <span>Select All (${connections.length})</span>
    </label>
  `;
  connectionsListEl.appendChild(header);

  const list = document.createElement('ul');
  list.className = 'conn-list';

  connections.forEach((conn, index) => {
    const li = document.createElement('li');
    li.className = 'conn-item';
    li.innerHTML = `
      <input type="checkbox" id="conn_${index}" class="conn-checkbox" checked />
      <label for="conn_${index}">
        <strong>${conn.name || 'Unknown'}</strong>
        ${conn.headline ? `<br><small>${conn.headline}</small>` : ''}
      </label>
    `;
    list.appendChild(li);
  });

  connectionsListEl.appendChild(list);

  // Handle select all
  document.getElementById('selectAll').addEventListener('change', (e) => {
    const checkboxes = document.querySelectorAll('.conn-checkbox');
    checkboxes.forEach(cb => cb.checked = e.target.checked);
  });
}

async function sendConnectionsToApi(connections) {
  const apiBase = getApiBase();
  const token = getToken();

  if (!token) {
    throw new Error('Not logged in. Please login to Cosmo first.');
  }

  const selectedConnections = connections.filter((_, index) => {
    const checkbox = document.getElementById(`conn_${index}`);
    return checkbox?.checked;
  });

  if (selectedConnections.length === 0) {
    throw new Error('No connections selected');
  }

  // Send to bulk create endpoint
  const res = await fetch(`${apiBase}/v1/contacts/bulk`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      contacts: selectedConnections.map(conn => ({
        name: conn.name,
        job_title: conn.headline,
        linkedin_url: conn.profile_url,
        source: 'LinkedIn',
      })),
    }),
  });

  const payload = await res.json().catch(() => ({}));

  if (!res.ok) {
    if (res.status === 401) {
      authToken = null;
      await chrome.storage.local.remove(['authToken', 'authTokenExpiry']);
      setLoginStatus(false);
      throw new Error('Session expired. Please login again.');
    }
    const message = payload?.error?.message || payload?.message || res.statusText;
    throw new Error(message);
  }

  return payload;
}

scrapeConnectionsBtn.addEventListener('click', async () => {
  setStatus('');
  scrapeConnectionsBtn.disabled = true;
  scrapedConnections = [];
  sendConnectionsBtn.disabled = true;

  try {
    saveSettings();
    setStatus('Scraping connections (scrolling page)...');

    const result = await scrapeConnections();
    scrapedConnections = result.data || [];

    setStatus(`Found ${scrapedConnections.length} connections`, 'success');
    displayConnections(scrapedConnections);

    if (scrapedConnections.length > 0) {
      sendConnectionsBtn.disabled = false;
    }
  } catch (err) {
    console.error('[Popup] Error:', err);
    setStatus(`Error: ${err.message || err}`, 'error');
  } finally {
    scrapeConnectionsBtn.disabled = false;
  }
});

sendConnectionsBtn.addEventListener('click', async () => {
  setStatus('');
  sendConnectionsBtn.disabled = true;

  try {
    saveSettings();

    const selectedCount = scrapedConnections.filter((_, index) => {
      const checkbox = document.getElementById(`conn_${index}`);
      return checkbox?.checked;
    }).length;

    setStatus(`Creating ${selectedCount} contacts...`);

    const result = await sendConnectionsToApi(scrapedConnections);
    const created = result?.data?.created || 0;
    const skipped = result?.data?.skipped || 0;

    setStatus(`Created: ${created}, Skipped: ${skipped}`, 'success');
  } catch (err) {
    console.error('[Popup] Error:', err);
    setStatus(`Error: ${err.message || err}`, 'error');
  } finally {
    sendConnectionsBtn.disabled = false;
  }
});

// === Generate LinkedIn Message ===
async function generateLinkedInMessage(scrapedData, tone, purpose, customNote) {
  const apiBase = getApiBase();
  const token = getToken();

  if (!token) {
    throw new Error('Not logged in. Please login to Cosmo first.');
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 60000);

  const res = await fetch(`${apiBase}/v1/linkedin/generate-message`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      profile: scrapedData.data || {},
      tone,
      purpose,
      custom_note: customNote,
    }),
    signal: controller.signal,
  });
  clearTimeout(timeoutId);

  const payload = await res.json().catch(() => ({}));

  if (!res.ok) {
    if (res.status === 401) {
      authToken = null;
      await chrome.storage.local.remove(['authToken', 'authTokenExpiry']);
      setLoginStatus(false);
      throw new Error('Session expired. Please login again.');
    }
    const message = payload?.error?.message || payload?.message || res.statusText;
    throw new Error(message);
  }

  return payload;
}

function displayGeneratedMessages(messages) {
  generatedMessagesEl.innerHTML = '';

  if (!messages || messages.length === 0) {
    generatedMessagesEl.innerHTML = '<p class="no-data">No messages generated</p>';
    return;
  }

  messages.forEach((msg, index) => {
    const card = document.createElement('div');
    card.className = 'gen-msg-card';
    card.innerHTML = `
      <div class="msg-label">Version ${index + 1}</div>
      <div class="msg-text"></div>
      <button class="copy-btn" data-index="${index}">Copy</button>
    `;
    // Set text content safely to avoid XSS
    card.querySelector('.msg-text').textContent = msg;
    generatedMessagesEl.appendChild(card);
  });

  // Handle copy buttons
  generatedMessagesEl.querySelectorAll('.copy-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const index = parseInt(btn.dataset.index);
      try {
        await navigator.clipboard.writeText(messages[index]);
        btn.textContent = 'Copied!';
        btn.classList.add('copied');
        setTimeout(() => {
          btn.textContent = 'Copy';
          btn.classList.remove('copied');
        }, 2000);
      } catch (err) {
        console.error('[Popup] Copy failed:', err);
        btn.textContent = 'Failed';
        setTimeout(() => { btn.textContent = 'Copy'; }, 2000);
      }
    });
  });
}

generateMsgBtn.addEventListener('click', async () => {
  setStatus('');
  generateMsgBtn.disabled = true;
  generatedMessagesEl.innerHTML = '';

  try {
    setStatus('Scraping LinkedIn profile...');
    const scrapedData = await scrapeCurrentTab();

    const profile = scrapedData.data || {};
    if (!profile.name && !profile.headline) {
      setStatus('Warning: Could not extract profile data. Try refreshing the LinkedIn page.', 'warning');
      return;
    }

    setStatus(`Found: ${profile.name || 'Unknown'}. Generating messages...`);

    const tone = toneSelect.value;
    const purpose = purposeSelect.value;
    const customNote = customNoteInput.value.trim();

    const result = await generateLinkedInMessage(scrapedData, tone, purpose, customNote);
    const messages = result?.data?.messages || result?.data || [];

    if (Array.isArray(messages) && messages.length > 0) {
      displayGeneratedMessages(messages);
      setStatus(`Generated ${messages.length} message versions`, 'success');
    } else {
      setStatus('No messages returned from API', 'warning');
    }
  } catch (err) {
    console.error('[Popup] Error:', err);
    setStatus(`Error: ${err.message || err}`, 'error');
  } finally {
    generateMsgBtn.disabled = false;
  }
});

// === Init ===
async function init() {
  loadSettings();
  const isLoggedIn = await checkAuth();

  // Auto-detect page type and switch mode
  const tab = await getActiveTab();
  if (tab?.url?.includes('/mynetwork/invite-connect/connections')) {
    switchMode('connections');
  } else if (tab?.url?.includes('linkedin.com/in/')) {
    switchMode('new');

    // Check if this profile is already imported (only if logged in)
    if (isLoggedIn && tab.url) {
      setStatus('Checking if already imported...');
      resetCreateButton();

      // Extract and normalize LinkedIn URL from tab URL
      const linkedinUrl = normalizeLinkedInUrl(tab.url);

      const existingContact = await checkContactByLinkedInUrl(linkedinUrl);
      if (existingContact) {
        showAlreadyImported(existingContact);
      } else {
        setStatus('Ready to import');
      }
    }
  }
}

init();
