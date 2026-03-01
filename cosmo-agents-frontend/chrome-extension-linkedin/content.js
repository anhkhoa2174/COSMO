function normalizeText(value) {
  if (!value) return '';
  return String(value).replace(/\s+/g, ' ').trim();
}

function getTextBySelectors(selectors) {
  for (const selector of selectors) {
    const el = document.querySelector(selector);
    if (el && el.textContent) {
      const text = normalizeText(el.textContent);
      if (text) return text;
    }
  }
  return '';
}

function getMetaContent(names) {
  for (const name of names) {
    const el =
      document.querySelector(`meta[name="${name}"]`) ||
      document.querySelector(`meta[property="${name}"]`);
    if (el) {
      const content = normalizeText(el.getAttribute('content'));
      if (content) return content;
    }
  }
  return '';
}

function parseJsonLd() {
  const scripts = document.querySelectorAll('script[type="application/ld+json"]');
  for (const script of scripts) {
    try {
      const data = JSON.parse(script.textContent || '{}');
      const candidates = Array.isArray(data) ? data : [data];
      for (const item of candidates) {
        if (item?.['@type'] === 'Person') {
          return item;
        }
      }
    } catch {
      continue;
    }
  }
  return null;
}

// Try to extract profile data from LinkedIn's embedded data in script tags
function parseLinkedInEmbeddedData() {
  const profile = {
    name: '',
    headline: '',
    location: '',
    company: '',
    about: '',
  };

  console.log('[Cosmo Scraper] Trying embedded data extraction...');

  // LinkedIn often embeds data in various script tags
  const scripts = document.querySelectorAll('script');

  for (const script of scripts) {
    const content = script.textContent || '';

    // Look for profile data patterns
    if (content.includes('publicIdentifier') || content.includes('firstName') || content.includes('"profile"')) {
      console.log('[Cosmo Scraper] Found potential profile data script');

      try {
        // Try to find JSON objects containing profile data
        const jsonMatches = content.match(/\{[^{}]*"firstName"[^{}]*\}/g) ||
                           content.match(/\{[^{}]*"publicIdentifier"[^{}]*\}/g);

        if (jsonMatches) {
          for (const jsonStr of jsonMatches) {
            try {
              const data = JSON.parse(jsonStr);
              if (data.firstName && data.lastName) {
                profile.name = `${data.firstName} ${data.lastName}`.trim();
                console.log('[Cosmo Scraper] Embedded - Found name:', profile.name);
              }
              if (data.headline) {
                profile.headline = data.headline;
                console.log('[Cosmo Scraper] Embedded - Found headline:', profile.headline);
              }
              if (data.locationName || data.location) {
                profile.location = data.locationName || data.location;
                console.log('[Cosmo Scraper] Embedded - Found location:', profile.location);
              }
            } catch {
              continue;
            }
          }
        }

        // Also try to find in larger JSON structures
        const bigJsonMatch = content.match(/"included"\s*:\s*\[[\s\S]*?\]/);
        if (bigJsonMatch) {
          try {
            const arr = JSON.parse(`[${bigJsonMatch[0].split(':').slice(1).join(':')}]`);
            // ... process included array
          } catch {
            // ignore
          }
        }
      } catch (e) {
        console.log('[Cosmo Scraper] Error parsing embedded data:', e.message);
      }
    }

    // Look for __INITIAL_STATE__ or similar
    if (content.includes('__INITIAL_STATE__') || content.includes('window.__PRELOADED_STATE__')) {
      console.log('[Cosmo Scraper] Found preloaded state');
      try {
        const stateMatch = content.match(/__INITIAL_STATE__\s*=\s*({[\s\S]*?});/) ||
                          content.match(/__PRELOADED_STATE__\s*=\s*({[\s\S]*?});/);
        if (stateMatch) {
          const state = JSON.parse(stateMatch[1]);
          console.log('[Cosmo Scraper] Parsed preloaded state');
          // Extract profile data from state
        }
      } catch {
        // ignore
      }
    }
  }

  return profile;
}

function extractFromInnerText(text) {
  const profile = {
    name: '',
    title: '',
    headline: '',
    company: '',
    location: '',
    about: '',
  };

  // Split by newlines and filter empty
  const lines = text.split('\n').map(l => l.trim()).filter(l => l.length > 0);

  console.log('[Cosmo Scraper] Parsing innerText, total lines:', lines.length);

  // Find key sections by markers
  let nameIndex = -1;
  let aboutIndex = -1;
  let activityIndex = -1;
  let experienceIndex = -1;

  // Find section markers
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line === 'About' && aboutIndex === -1) {
      aboutIndex = i;
    }
    if (line === 'Activity' && activityIndex === -1) {
      activityIndex = i;
    }
    if (line === 'Experience' && experienceIndex === -1) {
      experienceIndex = i;
    }
  }

  console.log('[Cosmo Scraper] Section indices - About:', aboutIndex, 'Activity:', activityIndex, 'Experience:', experienceIndex);

  // Parse profile header (before About section)
  const headerEndIndex = aboutIndex > 0 ? aboutIndex : (activityIndex > 0 ? activityIndex : 50);

  for (let i = 0; i < Math.min(headerEndIndex, lines.length); i++) {
    const line = lines[i];

    // Skip navigation and common UI elements
    if (line.match(/^(Home|My Network|Jobs|Messaging|Notifications|Me|For Business|Try Premium|Skip to|notifications|\d+\s*notifications)/i)) {
      continue;
    }

    // Skip buttons and actions
    if (line.match(/^(Message|Connect|Follow|More|Open to|Visit my website|Contact info|Show details|Profile enhanced)/i)) {
      continue;
    }

    // Skip LinkedIn UI/accessibility text
    if (line.match(/^(Keyboard shortcuts|Accessibility|Settings|Help Center|Privacy|Terms|Advertising|Business Services|Get the LinkedIn app|More options|See all|Show more|Show less|Loading|Search|Close jump menu|Jump to|Skip to)/i)) {
      continue;
    }

    // Skip common LinkedIn menu items and labels
    if (line.match(/(shortcuts|accessibility|copyright|linkedin corporation|jump menu|close menu|skip navigation)/i)) {
      continue;
    }

    // Skip connection/follower info
    if (line.match(/^\d+[\+,]?\s*(followers|connections|mutual)/i)) {
      continue;
    }

    // Skip degree indicators
    if (line.match(/^·?\s*(1st|2nd|3rd)$/i)) {
      continue;
    }

    // Skip mutual connection text
    if (line.match(/are mutual connections/i)) {
      continue;
    }

    // First non-skip line that looks like a name
    // Support various name formats: English, Vietnamese with diacritics, mixed case
    if (!profile.name && line.length < 50 && line.length > 2) {
      // Match names: starts with capital letter, may contain Unicode chars, spaces
      // Examples: "Jane Ng", "Trần Anh Khoa", "José García"
      const namePattern = /^[\p{Lu}][\p{L}\s\-'.]+$/u;

      // Common UI words that should NEVER appear in a person's name
      const uiWords = ['about', 'activity', 'experience', 'education', 'skills', 'home', 'jobs', 'network', 'keyboard', 'accessibility', 'settings', 'help', 'privacy', 'terms', 'advertising', 'business', 'search', 'loading', 'premium', 'profile', 'contact', 'message', 'connect', 'follow', 'notifications', 'show', 'more', 'less', 'view', 'open', 'close', 'save', 'edit', 'delete', 'cancel', 'submit', 'send', 'share', 'copy', 'download', 'upload', 'print', 'export', 'import', 'add', 'remove', 'create', 'update', 'recent', 'popular', 'trending', 'suggested', 'recommended', 'featured', 'jump', 'menu', 'skip', 'navigation', 'shortcuts', 'all', 'see'];

      // Check if ANY word in the line matches UI words
      const lineWords = line.toLowerCase().split(/\s+/);
      const hasUIWord = lineWords.some(word => uiWords.includes(word));

      if (hasUIWord) {
        continue;
      }

      if (namePattern.test(line)) {
        // Additional check: name should have at least 2 words or be short single word
        const words = line.split(' ').filter(w => w.length > 0);
        if (words.length >= 2 || (words.length === 1 && line.length <= 15)) {
          profile.name = line;
          nameIndex = i;
          console.log('[Cosmo Scraper] Found name:', line);
          continue;
        }
      }
    }

    // After finding name, look for headline (usually contains • or descriptive text)
    if (profile.name && !profile.headline && i > nameIndex) {
      // Skip if it's just the name repeated
      if (line === profile.name) continue;

      // Headline often contains bullet points or job descriptions
      if (line.includes('•') || line.includes('|') || line.includes('@') ||
          line.match(/(CEO|CTO|Founder|Engineer|Manager|Director|VP|President|Developer|Designer|Consultant|at\s)/i)) {
        if (line.length > 5 && line.length < 200) {
          profile.headline = line;
          console.log('[Cosmo Scraper] Found headline:', line);
          continue;
        }
      }
    }

    // Look for company (usually after headline, contains Inc., LLC, or company-like names)
    if (profile.headline && !profile.company && i > nameIndex) {
      if (line === profile.name || line === profile.headline) continue;
      if (line.match(/^·?\s*(1st|2nd|3rd)$/i)) continue;

      // Company patterns
      if (line.match(/(Inc\.|LLC|Ltd|Corp|Company|Technologies|Solutions|Labs|Studio|Agency|Group|Partners)$/i) ||
          (line.length > 3 && line.length < 60 && !line.match(/^\d/) && !line.match(/^(Contact|Message|Follow)/i))) {
        // Check it's not location
        if (!line.match(/(Metropolitan|Metro|Area|Metroplex|City,|, [A-Z]{2}$)/i)) {
          profile.company = line;
          console.log('[Cosmo Scraper] Found company:', line);
          continue;
        }
      }
    }

    // Look for location (city patterns)
    if (profile.name && !profile.location) {
      if (line.match(/(Metropolitan|Metro|Area|Metroplex)/i) ||
          line.match(/^[A-Z][a-z]+,\s*[A-Z]{2}$/) ||  // City, ST format
          line.match(/(Vietnam|Singapore|Indonesia|Malaysia|Thailand|India|USA|UK|Australia|Canada|Germany|France)/i)) {
        profile.location = line
          .replace(/\s+Metropolitan\s+Area$/i, '')
          .replace(/\s+Metro\s+Area$/i, '')
          .replace(/\s+Metroplex$/i, '')
          .trim();
        console.log('[Cosmo Scraper] Found location:', profile.location);
      }
    }
  }

  // Extract About section content
  if (aboutIndex > 0) {
    // About content is usually right after "About" heading
    const aboutEndIndex = activityIndex > aboutIndex ? activityIndex : aboutIndex + 10;
    let aboutContent = [];

    for (let i = aboutIndex + 1; i < Math.min(aboutEndIndex, lines.length); i++) {
      const line = lines[i];
      // Stop at next section or UI elements
      if (line === 'Activity' || line === 'Experience' || line === 'Education' ||
          line.match(/^\d+[\+,]?\s*followers/i) || line === 'Posts' || line === 'Comments') {
        break;
      }
      // Skip UI elements
      if (line.match(/^(Message|Show more|See all)/i)) {
        continue;
      }
      // Collect about content
      if (line.length > 10) {
        aboutContent.push(line);
      }
    }

    if (aboutContent.length > 0) {
      // Clean up "About me --" prefix
      let about = aboutContent.join(' ');
      about = about.replace(/^About me\s*[-–—:]+\s*/i, '').trim();
      profile.about = about.slice(0, 2000);
      console.log('[Cosmo Scraper] Found about:', profile.about.slice(0, 100) + '...');
    }
  }

  // Extract title from headline if present
  if (profile.headline && !profile.title) {
    // If headline contains separators, first part is usually the title
    const separators = [' • ', ' | ', ' @ ', ' at ', ' - '];
    for (const sep of separators) {
      if (profile.headline.includes(sep)) {
        profile.title = profile.headline.split(sep)[0].trim();
        break;
      }
    }
    // If no separator, use full headline as title
    if (!profile.title) {
      profile.title = profile.headline;
    }
  }

  // Extract Experience section
  if (experienceIndex > 0) {
    const experiences = [];
    let currentExp = null;

    // Find end of experience section (next major section)
    let expEndIndex = lines.length;
    for (let i = experienceIndex + 1; i < lines.length; i++) {
      if (lines[i].match(/^(Education|Licenses & certifications|Skills|Recommendations|Courses|Projects|Honors|Languages|Organizations)$/i)) {
        expEndIndex = i;
        break;
      }
    }

    console.log('[Cosmo Scraper] Experience section:', experienceIndex, 'to', expEndIndex);

    for (let i = experienceIndex + 1; i < expEndIndex; i++) {
      const line = lines[i];

      // Skip UI elements
      if (line.match(/^(Show all|See more|more|less|\d+ more|Show \d+)/i)) continue;
      if (line.length < 3) continue;

      // Detect job title patterns (usually followed by company)
      // Titles often contain: CEO, Director, Manager, Engineer, Developer, etc.
      const isTitleLine = line.match(/(CEO|CTO|CFO|COO|President|Vice President|VP|Director|Manager|Head of|Lead|Senior|Junior|Engineer|Developer|Designer|Analyst|Consultant|Specialist|Coordinator|Associate|Assistant|Intern|Founder|Co-Founder|Owner|Partner)/i);

      // Detect company line (contains · Full-time, Part-time, Contract, etc.)
      const isCompanyLine = line.match(/·\s*(Full-time|Part-time|Contract|Freelance|Internship|Self-employed|Seasonal|Apprenticeship)/i);

      // Detect date range (contains months/years pattern)
      const isDateLine = line.match(/(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{4}\s*[-–]\s*(Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)/i) ||
                         line.match(/\d+\s*(yr|yrs|mo|mos|year|years|month|months)/i);

      // Detect location line
      const isLocationLine = line.match(/(On-site|Remote|Hybrid)/i) ||
                             line.match(/(Vietnam|Singapore|USA|UK|Australia|Germany|France|Japan|Korea|China|India|Thailand|Malaysia|Indonesia)/i);

      if (isTitleLine && !isCompanyLine && !isDateLine) {
        // Start new experience entry
        if (currentExp && currentExp.title) {
          experiences.push(currentExp);
        }
        currentExp = {
          title: line,
          company: '',
          date_range: '',
          location: '',
          description: ''
        };
      } else if (currentExp) {
        if (isCompanyLine) {
          // Extract company name (before the ·)
          const companyMatch = line.match(/^([^·]+)/);
          if (companyMatch) {
            currentExp.company = companyMatch[1].trim();
          }
        } else if (isDateLine && !currentExp.date_range) {
          currentExp.date_range = line;
        } else if (isLocationLine && !currentExp.location) {
          currentExp.location = line.replace(/\s*·\s*(On-site|Remote|Hybrid)/i, '').trim();
        } else if (line.length > 20 && !line.match(/^(Skills:|Training:)/i)) {
          // Likely description text
          if (currentExp.description) {
            currentExp.description += ' ' + line;
          } else {
            currentExp.description = line;
          }
        }
      }
    }

    // Push last experience
    if (currentExp && currentExp.title) {
      experiences.push(currentExp);
    }

    if (experiences.length > 0) {
      profile.experience = experiences;
      console.log('[Cosmo Scraper] Found experiences:', experiences.length);
    }
  }

  return profile;
}

// Note: We only use 'name' field now (not first_name/last_name)

// Helper to get all text from an element, including shadow DOM
function getDeepText(element) {
  if (!element) return '';

  let text = '';

  // If element has shadow root, get text from it
  if (element.shadowRoot) {
    text += getDeepText(element.shadowRoot);
  }

  // Get text from child nodes
  for (const child of element.childNodes) {
    if (child.nodeType === Node.TEXT_NODE) {
      text += child.textContent + '\n';
    } else if (child.nodeType === Node.ELEMENT_NODE) {
      text += getDeepText(child);
    }
  }

  return text;
}

// Helper to find elements in shadow DOM
function querySelectorDeep(selector, root = document) {
  // First try normal querySelector
  let result = root.querySelector(selector);
  if (result) return result;

  // Search in shadow roots
  const allElements = root.querySelectorAll('*');
  for (const el of allElements) {
    if (el.shadowRoot) {
      result = querySelectorDeep(selector, el.shadowRoot);
      if (result) return result;
    }
  }

  return null;
}

// Helper to find all elements matching selector in shadow DOM
function querySelectorAllDeep(selector, root = document) {
  const results = [];

  // Add normal matches
  results.push(...root.querySelectorAll(selector));

  // Search in shadow roots
  const allElements = root.querySelectorAll('*');
  for (const el of allElements) {
    if (el.shadowRoot) {
      results.push(...querySelectorAllDeep(selector, el.shadowRoot));
    }
  }

  return results;
}

// Method to extract from profile card section (2024-2025 LinkedIn layout)
function extractFromProfileCard() {
  const profile = {
    name: '',
    headline: '',
    location: '',
    company: '',
    about: '',
  };

  console.log('[Cosmo Scraper] Trying profile card extraction...');

  // Try to get full text including shadow DOM
  const fullText = getDeepText(document.body);
  const lineCount = fullText.split('\n').filter(l => l.trim()).length;
  console.log('[Cosmo Scraper] Deep text lines:', lineCount);

  // Find the main profile section - LinkedIn uses various containers
  // Also try shadow DOM selectors
  const profileContainers = [
    document.querySelector('.pv-top-card'),
    document.querySelector('.scaffold-layout__main'),
    document.querySelector('main section'),
    document.querySelector('#main'),
    document.querySelector('[role="main"]'),
    querySelectorDeep('.pv-top-card'),
    querySelectorDeep('.profile-card'),
    querySelectorDeep('[data-view-name="profile-card"]'),
  ].filter(Boolean);

  for (const container of profileContainers) {
    // Find H1 for name
    if (!profile.name) {
      const h1 = container.querySelector('h1');
      if (h1) {
        const text = normalizeText(h1.textContent);
        if (text && text.length > 2 && text.length < 50) {
          // Skip if it looks like a section title
          if (!text.match(/^(About|Experience|Education|Skills|Activity)/i)) {
            profile.name = text;
            console.log('[Cosmo Scraper] ProfileCard - Found name:', text);
          }
        }
      }
    }

    // Find headline - usually a div with text-body-medium class after the name
    if (!profile.headline) {
      const headlineEls = container.querySelectorAll('.text-body-medium');
      for (const el of headlineEls) {
        const text = normalizeText(el.textContent);
        if (text && text.length > 5 && text.length < 300) {
          // Skip if it's navigation or common UI
          if (!text.match(/^(Home|Jobs|Messaging|Notifications|My Network|Premium)/i)) {
            profile.headline = text;
            console.log('[Cosmo Scraper] ProfileCard - Found headline:', text);
            break;
          }
        }
      }
    }

    // Find location - usually text-body-small with location keywords
    if (!profile.location) {
      const locationEls = container.querySelectorAll('.text-body-small');
      for (const el of locationEls) {
        const text = normalizeText(el.textContent);
        if (text && text.length > 3 && text.length < 100) {
          // Check if it looks like a location
          if (text.match(/(Vietnam|Singapore|Indonesia|Malaysia|Thailand|India|USA|UK|Australia|Canada|Germany|France|Japan|Korea|China|Hong Kong|Taiwan|City|Metro|Area)/i) ||
              text.match(/^[A-Z][a-z]+,\s*[A-Z]/)) {
            profile.location = text
              .replace(/\s+Metropolitan\s+Area$/i, '')
              .replace(/\s+Metro\s+Area$/i, '')
              .trim();
            console.log('[Cosmo Scraper] ProfileCard - Found location:', profile.location);
            break;
          }
        }
      }
    }
  }

  // Find About section
  const aboutSection = document.querySelector('#about') ||
                       document.querySelector('[data-view-name="profile-component-entity"]') ||
                       document.querySelector('.pv-about-section');
  if (aboutSection) {
    const aboutText = aboutSection.querySelector('.inline-show-more-text') ||
                      aboutSection.querySelector('.pv-about__summary-text') ||
                      aboutSection.querySelector('span[aria-hidden="true"]');
    if (aboutText) {
      profile.about = normalizeText(aboutText.textContent).slice(0, 2000);
      console.log('[Cosmo Scraper] ProfileCard - Found about:', profile.about.slice(0, 100) + '...');
    }
  }

  return profile;
}

function extractProfileData() {
  // Normalize LinkedIn URL - remove query params for consistent matching
  const normalizedUrl = window.location.href.split('?')[0];

  const profile = {
    name: '',
    title: '',
    headline: '',
    company: '',
    location: '',
    about: '',
    source_url: normalizedUrl,
  };

  console.log('[Cosmo Scraper] Starting extraction...');
  console.log('[Cosmo Scraper] URL:', window.location.href);

  // Method 0: Try embedded LinkedIn data first (most reliable on new layouts)
  const embeddedData = parseLinkedInEmbeddedData();
  if (embeddedData.name) {
    profile.name = embeddedData.name;
  }
  if (embeddedData.headline) {
    profile.headline = embeddedData.headline;
  }
  if (embeddedData.location) {
    profile.location = embeddedData.location;
  }
  if (embeddedData.company) {
    profile.company = embeddedData.company;
  }
  if (embeddedData.about) {
    profile.about = embeddedData.about;
  }

  // Method 1: Try JSON-LD first (most reliable if available)
  const jsonLd = parseJsonLd();
  if (jsonLd) {
    console.log('[Cosmo Scraper] Found JSON-LD data:', jsonLd);
    profile.name = profile.name || normalizeText(jsonLd.name);
    profile.title = normalizeText(jsonLd.jobTitle);

    if (jsonLd.worksFor && !profile.company) {
      if (typeof jsonLd.worksFor === 'string') {
        profile.company = normalizeText(jsonLd.worksFor);
      } else if (jsonLd.worksFor.name) {
        profile.company = normalizeText(jsonLd.worksFor.name);
      }
    }

    if (jsonLd.address && !profile.location) {
      const city = normalizeText(jsonLd.address.addressLocality);
      const state = normalizeText(jsonLd.address.addressRegion);
      const country = normalizeText(jsonLd.address.addressCountry);
      const parts = [city, state, country].filter(Boolean);
      profile.location = parts.join(', ');
    }
  }

  // Method 1.5: Document title (LinkedIn always shows "Name | LinkedIn")
  // Handle notification count: "(3) Name | LinkedIn" or "(99+) Name | LinkedIn"
  if (!profile.name) {
    let pageTitle = document.title || '';
    // Remove notification count prefix like "(3) " or "(99+) "
    pageTitle = pageTitle.replace(/^\(\d+\+?\)\s*/, '');
    // Format: "Name | LinkedIn" or "Name - Title | LinkedIn"
    if (pageTitle.includes(' | LinkedIn')) {
      let namePart = pageTitle.split(' | LinkedIn')[0].trim();
      // If contains " - ", take only the name part (before the title)
      if (namePart.includes(' - ')) {
        namePart = namePart.split(' - ')[0].trim();
      }
      if (namePart && namePart.length > 2 && namePart.length < 50) {
        profile.name = namePart;
        console.log('[Cosmo Scraper] Found name from page title:', namePart);
      }
    }
  }

  // Method 1.6: Meta tags (fallback - doesn't have notification count)
  if (!profile.name) {
    const ogTitle = getMetaContent(['og:title']);
    if (ogTitle) {
      let namePart = ogTitle.split(' | ')[0].trim();
      if (namePart.includes(' - ')) {
        namePart = namePart.split(' - ')[0].trim();
      }
      if (namePart && namePart.length > 2 && namePart.length < 50) {
        profile.name = namePart;
        console.log('[Cosmo Scraper] Found name from og:title:', namePart);
      }
    }
  }
  if (!profile.headline) {
    const ogDesc = getMetaContent(['og:description', 'description']);
    if (ogDesc) {
      profile.headline = ogDesc.replace(/ · LinkedIn/i, '').trim();
    }
  }

  // Method 2: DOM selectors (best effort, LinkedIn changes often)
  // LinkedIn 2024-2025 layout selectors
  if (!profile.name) {
    profile.name = getTextBySelectors([
      // New LinkedIn layouts (2024-2025)
      'h1.inline.t-24.v-align-middle.break-words',
      'h1.text-heading-xlarge',
      'h1[data-anonymize="person-name"]',
      '.pv-text-details__left-panel h1',
      '.top-card-layout__title',
      // Profile card layouts
      'section.artdeco-card h1',
      '.scaffold-layout__main h1',
      // New profile page structure
      '.ph5 h1',
      '.mt2 h1',
      'div[data-view-name="profile-component-entity"] h1',
      // Generic main content h1
      'main h1',
      '#main h1',
      '[role="main"] h1',
    ]);
  }

  // Method 2.5: Fallback - find the first H1 that looks like a name
  if (!profile.name) {
    const h1Elements = document.querySelectorAll('h1');
    for (const h1 of h1Elements) {
      const text = normalizeText(h1.textContent);
      // Skip if empty or too long
      if (!text || text.length > 50 || text.length < 2) continue;
      // Skip if contains obvious UI words
      if (text.match(/(linkedin|profile|experience|education|skills|activity|about|home|jobs|messaging|notifications)/i)) continue;
      // Accept if it looks like a name (2+ words, no special patterns)
      const words = text.split(' ').filter(w => w.length > 0);
      if (words.length >= 2 && words.length <= 5) {
        profile.name = text;
        console.log('[Cosmo Scraper] Found name from H1:', text);
        break;
      }
    }
  }

  // Method 2.6: Try aria-label and data attributes
  if (!profile.name) {
    // Look for elements with aria-label containing profile name
    const ariaElements = document.querySelectorAll('[aria-label*="profile"]');
    for (const el of ariaElements) {
      const ariaLabel = el.getAttribute('aria-label') || '';
      // Pattern: "X's profile" or "View X's profile"
      const match = ariaLabel.match(/(?:View\s+)?(.+?)(?:'s\s+profile|'s profile)/i);
      if (match && match[1]) {
        const name = normalizeText(match[1]);
        if (name && name.length > 2 && name.length < 50) {
          profile.name = name;
          console.log('[Cosmo Scraper] Found name from aria-label:', name);
          break;
        }
      }
    }
  }

  // Method 2.7: Try image alt text (profile photos often have name)
  if (!profile.name) {
    const profileImgs = document.querySelectorAll('img[alt]:not([alt=""])');
    for (const img of profileImgs) {
      const alt = normalizeText(img.getAttribute('alt') || '');
      // Skip common alt texts
      if (alt.match(/^(LinkedIn|Profile photo|Photo|Image|Logo|Icon|Loading)/i)) continue;
      // Check if it looks like a name
      const words = alt.split(' ').filter(w => w.length > 0);
      if (words.length >= 2 && words.length <= 5 && alt.length < 50) {
        // Verify it's not UI text
        if (!alt.match(/(linkedin|profile|experience|education|skills|company|logo|icon|background)/i)) {
          profile.name = alt;
          console.log('[Cosmo Scraper] Found name from img alt:', alt);
          break;
        }
      }
    }
  }

  // Method 2.8: Extract from URL path (last resort)
  if (!profile.name) {
    const urlMatch = window.location.pathname.match(/\/in\/([^\/\?]+)/);
    if (urlMatch && urlMatch[1]) {
      // Convert URL slug to name: "john-doe-123abc" -> "John Doe"
      let slug = urlMatch[1].replace(/-\w{5,}$/, ''); // Remove trailing ID
      slug = slug.replace(/-/g, ' '); // Replace hyphens with spaces
      // Capitalize each word
      const name = slug.split(' ')
        .filter(w => w.length > 0)
        .map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
        .join(' ');
      if (name && name.length > 2 && name.length < 50) {
        profile.name = name;
        console.log('[Cosmo Scraper] Found name from URL:', name);
      }
    }
  }
  if (!profile.headline) {
    profile.headline = getTextBySelectors([
      // New LinkedIn layouts (2024-2025)
      '.text-body-medium.break-words',
      '.pv-text-details__left-panel .text-body-medium',
      '.top-card-layout__headline',
      // Additional headline selectors
      'div.text-body-medium',
      '.ph5 .text-body-medium',
      '.mt2 .text-body-medium',
      'div[data-view-name="profile-component-entity"] .text-body-medium',
      // Fallback: look for div after h1
      'main h1 + div',
    ]);
  }
  if (!profile.location) {
    profile.location = getTextBySelectors([
      // New LinkedIn layouts (2024-2025)
      '.pv-text-details__left-panel .text-body-small.inline',
      '.top-card-layout__first-subline',
      '.top-card__subline-item',
      // Additional location selectors
      'span.text-body-small.inline',
      '.ph5 .text-body-small',
      '.mt2 .text-body-small',
      // Look for location patterns
      'span[aria-label*="location"]',
      'span[data-anonymize="location"]',
    ])
      .replace(/\s+Metropolitan\s+Area$/i, '')
      .replace(/\s+Metro\s+Area$/i, '')
      .replace(/\s+Metroplex$/i, '')
      .trim();
  }
  if (!profile.company) {
    profile.company = getTextBySelectors([
      // New LinkedIn layouts (2024-2025)
      '.pv-text-details__right-panel .pv-text-details__right-panel-item',
      '.pv-top-card--experience-list .pv-entity__secondary-title',
      '.experience-item__subtitle',
      // Additional company selectors
      'button[aria-label*="current company"]',
      'a[data-field="experience_company_logo"]',
      '.ph5 button span',
      'div[data-view-name="profile-component-entity"] button span',
    ]);
  }

  // Method 2.9: Profile card extraction (2024-2025 layout)
  const profileCard = extractFromProfileCard();
  profile.name = profile.name || profileCard.name;
  profile.headline = profile.headline || profileCard.headline;
  profile.location = profile.location || profileCard.location;
  profile.company = profile.company || profileCard.company;
  profile.about = profile.about || profileCard.about;

  // Method 3: Parse from innerText (ONLY for about/experience - NOT for name)
  // Name extraction from innerText is too unreliable due to LinkedIn UI text
  if (!profile.about) {
    console.log('[Cosmo Scraper] Parsing from innerText for about/experience...');
    const text = document.body?.innerText || '';
    const parsed = extractFromInnerText(text);

    // Only fill in about and experience from innerText - NOT name
    // Name should come from DOM selectors or meta tags which are more reliable
    profile.headline = profile.headline || parsed.headline;
    profile.title = profile.title || parsed.title;
    profile.company = profile.company || parsed.company;
    profile.location = profile.location || parsed.location;
    profile.about = profile.about || parsed.about;
    if (parsed.experience && parsed.experience.length > 0) {
      profile.experience = parsed.experience;
    }
  }

  // Derive company/title from headline if still missing
  if (profile.headline) {
    if (!profile.title) {
      const separators = [' • ', ' | ', ' @ ', ' at ', ' - '];
      for (const sep of separators) {
        if (profile.headline.includes(sep)) {
          profile.title = profile.headline.split(sep)[0].trim();
          break;
        }
      }
      if (!profile.title) {
        profile.title = profile.headline;
      }
    }
    if (!profile.company) {
      const match = profile.headline.match(/\bat\s+([^|•-]+)/i);
      if (match && match[1]) {
        profile.company = normalizeText(match[1]);
      }
    }
  }

  // Log extraction success/failure for debugging
  console.log('[Cosmo Scraper] ========== EXTRACTION SUMMARY ==========');
  console.log('[Cosmo Scraper] Name:', profile.name || '❌ NOT FOUND');
  console.log('[Cosmo Scraper] Headline:', profile.headline ? profile.headline.slice(0, 50) + '...' : '❌ NOT FOUND');
  console.log('[Cosmo Scraper] Location:', profile.location || '❌ NOT FOUND');
  console.log('[Cosmo Scraper] Company:', profile.company || '❌ NOT FOUND');
  console.log('[Cosmo Scraper] About:', profile.about ? profile.about.slice(0, 50) + '...' : '❌ NOT FOUND');
  console.log('[Cosmo Scraper] Experience:', profile.experience ? `${profile.experience.length} items` : '❌ NOT FOUND');
  console.log('[Cosmo Scraper] ==========================================');
  console.log('[Cosmo Scraper] Final profile data:', profile);

  const rawText = normalizeText(document.body?.innerText || '').slice(0, 5000);

  return { profile, rawText };
}

// Wait for page to fully load before scraping
async function waitForPageLoad(maxWaitMs = 5000) {
  const startTime = Date.now();

  while (Date.now() - startTime < maxWaitMs) {
    // Check if main profile content has loaded
    const h1 = document.querySelector('h1');
    const mainContent = document.querySelector('main') || document.querySelector('#main');
    const bodyText = document.body?.innerText || '';
    const lineCount = bodyText.split('\n').filter(l => l.trim()).length;

    console.log(`[Cosmo Scraper] Waiting for load... H1: ${!!h1}, Main: ${!!mainContent}, Lines: ${lineCount}`);

    // Consider loaded if we have H1 and reasonable content
    if (h1 && mainContent && lineCount > 10) {
      console.log('[Cosmo Scraper] Page appears loaded');
      return true;
    }

    // Wait 500ms before checking again
    await new Promise(resolve => setTimeout(resolve, 500));
  }

  console.log('[Cosmo Scraper] Timeout waiting for page load');
  return false;
}

// Retry extraction with delays
async function extractWithRetry(maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    console.log(`[Cosmo Scraper] Extraction attempt ${i + 1}/${maxRetries}`);

    const payload = extractProfileData();

    // Check if we got meaningful data
    if (payload.profile.name && (payload.profile.headline || payload.profile.about || payload.profile.location)) {
      console.log('[Cosmo Scraper] Got meaningful data, returning');
      return payload;
    }

    // If only got name from title, wait and retry
    if (i < maxRetries - 1) {
      console.log('[Cosmo Scraper] Insufficient data, waiting 1s before retry...');
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }

  // Return whatever we have after all retries
  console.log('[Cosmo Scraper] Max retries reached, returning best effort data');
  return extractProfileData();
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === 'SCRAPE_PROFILE') {
    // Use async extraction with retry
    (async () => {
      try {
        // First wait for page to load
        await waitForPageLoad(5000);

        // Then extract with retry
        const payload = await extractWithRetry(3);
        sendResponse({ ok: true, data: payload.profile, raw_text: payload.rawText });
      } catch (err) {
        console.error('[Cosmo Scraper] Error:', err);
        sendResponse({ ok: false, error: err.message });
      }
    })();
    return true; // Keep message channel open for async response
  }
  return false;
});

console.log('[Cosmo Scraper] Content script loaded on:', window.location.href);
