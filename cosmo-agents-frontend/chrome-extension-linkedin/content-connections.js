/**
 * Content script for LinkedIn Connections page
 * Scrapes connection list from /mynetwork/invite-connect/connections/
 */

function normalizeText(value) {
  if (!value) return '';
  return String(value).replace(/\s+/g, ' ').trim();
}

function extractConnections() {
  const connections = [];

  console.log('[Cosmo Scraper] Starting connections extraction...');

  // LinkedIn connections are in a list with various possible selectors (2024-2025)
  const connectionSelectors = [
    // Main connection cards (current layout)
    '.mn-connection-card',
    '.ember-view.mn-connection-card',
    // List items in connections
    'li.mn-connection-card',
    // Alternative: connection list items
    '.scaffold-finite-scroll__content > ul > li',
    // New 2024-2025 layouts
    'li.reusable-search__result-container',
    'li[class*="search-result"]',
    '.search-results-container li',
    'ul.reusable-search__entity-result-list > li',
    // Connections page specific
    'div[data-view-name="profile-component-entity"]',
    '.artdeco-list__item',
    // Fallback: any list item with connection link
    'li[class*="connection"]',
    'li[class*="result"]',
  ];

  let connectionElements = [];

  for (const selector of connectionSelectors) {
    const elements = document.querySelectorAll(selector);
    if (elements.length > 0) {
      connectionElements = elements;
      console.log(`[Cosmo Scraper] Found ${elements.length} connections with selector: ${selector}`);
      break;
    }
  }

  // If still no results, try broader search
  if (connectionElements.length === 0) {
    // Look for connection cards by structure
    const allCards = document.querySelectorAll('a[href*="/in/"]');
    console.log(`[Cosmo Scraper] Found ${allCards.length} profile links`);

    const processedUrls = new Set();

    allCards.forEach(link => {
      const href = link.getAttribute('href');
      if (!href || processedUrls.has(href)) return;

      // Extract LinkedIn username from URL
      const match = href.match(/\/in\/([^\/\?]+)/);
      if (!match) return;

      processedUrls.add(href);

      // Try to find parent card element
      const card = link.closest('li') || link.closest('[class*="card"]') || link.parentElement?.parentElement;
      if (!card) return;

      const connection = extractConnectionFromCard(card, href);
      if (connection && connection.name) {
        connections.push(connection);
      }
    });
  } else {
    // Process found connection cards
    connectionElements.forEach((card, index) => {
      const connection = extractConnectionFromCard(card);
      if (connection && connection.name) {
        connections.push(connection);
      }
    });
  }

  console.log(`[Cosmo Scraper] ========== CONNECTIONS SUMMARY ==========`);
  console.log(`[Cosmo Scraper] Total extracted: ${connections.length} connections`);
  if (connections.length > 0) {
    console.log(`[Cosmo Scraper] Sample connection:`, connections[0]);
  }
  console.log(`[Cosmo Scraper] ==========================================`);
  return connections;
}

function extractConnectionFromCard(card, profileUrl = null) {
  const connection = {
    name: '',
    headline: '',
    profile_url: profileUrl ? profileUrl.split('?')[0] : '', // Normalize: remove query params
    connected_date: '',
  };

  // Extract profile URL
  if (!connection.profile_url) {
    const profileLink = card.querySelector('a[href*="/in/"]');
    if (profileLink) {
      let url = profileLink.getAttribute('href');
      // Make absolute URL
      if (url.startsWith('/')) {
        url = 'https://www.linkedin.com' + url;
      }
      // Normalize: remove query params for consistent matching
      connection.profile_url = url.split('?')[0];
    }
  }

  // Extract name - multiple selectors (2024-2025)
  const nameSelectors = [
    '.mn-connection-card__name',
    '.entity-result__title-text a',
    '.entity-result__title-text span[aria-hidden="true"]',
    // New layouts
    '.app-aware-link span[aria-hidden="true"]',
    '.reusable-search__result-container span[aria-hidden="true"]',
    'a[href*="/in/"] span[dir="ltr"]',
    'a[href*="/in/"] span[aria-hidden="true"]',
    // Fallback selectors
    'span[aria-hidden="true"]',
    '.t-16.t-black.t-bold',
    '.t-roman.t-sans',
  ];

  for (const selector of nameSelectors) {
    const nameEl = card.querySelector(selector);
    if (nameEl?.textContent?.trim()) {
      connection.name = normalizeText(nameEl.textContent);
      break;
    }
  }

  // Fallback: get name from profile link text
  if (!connection.name) {
    const profileLink = card.querySelector('a[href*="/in/"]');
    if (profileLink?.textContent?.trim()) {
      // Filter out "View profile" type texts
      const text = normalizeText(profileLink.textContent);
      if (!text.toLowerCase().includes('view') && !text.toLowerCase().includes('profile')) {
        connection.name = text;
      }
    }
  }

  // Extract headline/occupation (2024-2025)
  const headlineSelectors = [
    '.mn-connection-card__occupation',
    '.entity-result__primary-subtitle',
    '.entity-result__summary',
    // New layouts
    '.reusable-search-simple-insight__text-container',
    '.linked-area .t-14',
    '.t-14.t-black--light.t-normal',
    'span.t-14.t-black--light',
    '.t-14.t-normal',
    // Fallback
    'p.t-14',
    'div.t-14',
  ];

  for (const selector of headlineSelectors) {
    const headlineEl = card.querySelector(selector);
    if (headlineEl?.textContent?.trim()) {
      connection.headline = normalizeText(headlineEl.textContent);
      break;
    }
  }

  // Extract connected date if available
  const dateSelectors = [
    '.mn-connection-card__connected-date',
    'time',
    '.time-badge',
    'span[class*="time"]',
  ];

  for (const selector of dateSelectors) {
    const dateEl = card.querySelector(selector);
    if (dateEl?.textContent?.trim()) {
      connection.connected_date = normalizeText(dateEl.textContent);
      break;
    }
  }

  return connection;
}

// Auto-scroll to load more connections
async function loadAllConnections(maxScrolls = 10) {
  console.log('[Cosmo Scraper] Starting auto-scroll to load connections...');

  let previousCount = 0;
  let scrollCount = 0;

  while (scrollCount < maxScrolls) {
    // Scroll to bottom
    window.scrollTo(0, document.body.scrollHeight);

    // Wait for content to load
    await new Promise(resolve => setTimeout(resolve, 1500));

    // Count current connections
    const currentCount = document.querySelectorAll('a[href*="/in/"]').length;

    console.log(`[Cosmo Scraper] Scroll ${scrollCount + 1}: ${currentCount} profile links found`);

    // If no new content loaded, stop
    if (currentCount === previousCount) {
      console.log('[Cosmo Scraper] No new content loaded, stopping scroll');
      break;
    }

    previousCount = currentCount;
    scrollCount++;
  }

  // Scroll back to top
  window.scrollTo(0, 0);

  return extractConnections();
}

// Listen for messages from popup
chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === 'SCRAPE_CONNECTIONS') {
    const maxScrolls = message.maxScrolls || 10;

    // Use async loading
    loadAllConnections(maxScrolls).then(connections => {
      sendResponse({
        ok: true,
        data: connections,
        count: connections.length,
        page_url: window.location.href
      });
    });

    // Return true to indicate async response
    return true;
  }

  if (message?.type === 'SCRAPE_CONNECTIONS_QUICK') {
    // Quick scrape without scrolling
    const connections = extractConnections();
    sendResponse({
      ok: true,
      data: connections,
      count: connections.length,
      page_url: window.location.href
    });
    return true;
  }

  return false;
});

// Log when script is loaded
console.log('[Cosmo Scraper] Connections content script loaded on:', window.location.href);
