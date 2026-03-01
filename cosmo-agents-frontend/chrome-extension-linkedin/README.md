# Cosmo LinkedIn Chrome Extension (Dev)

Scrapes LinkedIn profile pages and sends data to Cosmo backend with smart field mapping.

## Install (Unpacked)
1. Open `chrome://extensions`
2. Enable **Developer mode**
3. Click **Load unpacked**
4. Select folder: `cosmo-agents-frontend/chrome-extension-linkedin`

## Usage
1. Open a LinkedIn profile: `https://www.linkedin.com/in/...`
2. Click extension icon and configure:
   - **API Base**: Your backend URL (default `http://localhost:8081`)
   - **Contact ID**: UUID of contact to update
   - **Bearer Token**: Your JWT auth token
   - **Use AI mapping**: ✓ Enable for better extraction (uses OpenAI, slower)
3. Click **Scrape & Send**

## Features

### Smart Location Mapping
Backend automatically normalizes locations:
- `HCM` → `Ho Chi Minh City`
- Auto-infers country: `HCM` → `Vietnam`
- Also handles: `NYC`, `SF`, `Hanoi`, `Singapore`, etc.

### Multi-Strategy Extraction
Extension tries multiple methods to extract data:
1. **JSON-LD structured data** (most reliable)
2. **Multiple DOM selectors** (handles different LinkedIn layouts)
3. **Text parsing** (extracts company from headline)
4. **AI extraction** (optional, uses GPT to extract from raw page text)

### Extracted Fields
- `full_name`: Person's full name
- `title` / `headline`: Current job title
- `company`: Current company name
- `location`: City/country (auto-normalized)

## Debugging

If scraping fails on some profiles:

### 1. Check Browser Console
Open DevTools (`F12`) and look for `[Cosmo Scraper]` logs:
```
[Cosmo Scraper] Starting extraction...
[Cosmo Scraper] Found JSON-LD data: {...}
[Cosmo Scraper] Found name via selector: h1.text-heading-xlarge
[Cosmo Scraper] Final profile data: {...}
```

### 2. Check Popup Console
Right-click extension icon → **Inspect popup** → Console tab:
```
[Popup] Scraped data: {...}
[Popup] Scraped fields: full_name, headline, company, location
```

### 3. Common Issues

**"No data extracted"**
- LinkedIn has multiple layouts, some profiles use different HTML structure
- **Solution**: Enable "Use AI mapping" checkbox
- AI mode sends raw page text to backend for GPT extraction

**"Some fields missing"**
- Different profiles show different information
- **Solution**: Enable AI mode or manually add missing fields in app

**"Location shows abbreviation"**
- This is expected - backend automatically maps abbreviations
- `HCM` → `Ho Chi Minh City, Vietnam`
- `NYC` → `New York, United States`

**"Extension not working at all"**
- Make sure you're on a profile page: `https://www.linkedin.com/in/*`
- Check that extension is enabled in `chrome://extensions`
- Reload the LinkedIn page after installing extension

## API Payload

Sends `POST /v1/contacts/:id/extract-from-extension`:

```json
{
  "source_url": "https://www.linkedin.com/in/example",
  "use_ai": true,
  "data": {
    "full_name": "Nguyen Van A",
    "title": "Software Engineer",
    "headline": "Software Engineer at Rockship",
    "company": "Rockship",
    "location": "HCM"
  },
  "raw_text": "Nguyen Van A\nSoftware Engineer at Rockship\nHCM, Vietnam\n..."
}
```

Backend processes:
- Maps `location: "HCM"` → `city: "Ho Chi Minh City"`, `country: "Vietnam"`
- Splits `full_name` → `first_name`, `last_name`
- Extracts additional fields from `raw_text` if `use_ai: true`
