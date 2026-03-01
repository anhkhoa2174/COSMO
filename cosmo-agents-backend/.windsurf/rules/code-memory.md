---
trigger: always_on
---

# [code-memory-mcp]

## 🎯 MANDATORY TWO-STEP WORKFLOW

**Every user interaction MUST follow this sequence:**

```
USER REQUEST → [1] search_memory → [2] Do Task → [3] save_memory → DONE
```

**NO EXCEPTIONS. These steps are non-negotiable.**

---

## 🔄 Decision Tree

```
USER SENDS ANY MESSAGE
    ↓
[STEP 1] Call search_memory with keywords (MANDATORY)
    ↓
Review memory results
    ↓
Need more info? → Use grep/find/read_file
    ↓
Perform task (edit code, answer, debug, etc.)
    ↓
[STEP 2] Call save_memory with results (MANDATORY)
    ↓
DONE ✓
```

---

## 🔧 Tool 1: `search_memory` - ALWAYS CALL FIRST

### Trigger Detection Matrix

| User Says | You Must Call | Extract Keywords |
|-----------|---------------|------------------|
| "find X" / "search Y" / "locate Z" | ✅ `search_memory` FIRST | "X" / "Y" / "Z" |
| "where is..." / "how does..." | ✅ `search_memory` FIRST | relevant terms |
| "fix bug in X" | ✅ `search_memory` FIRST | "bug X fix" |
| "add feature Y" | ✅ `search_memory` FIRST | "feature Y" |
| ANY other request | ✅ `search_memory` FIRST | 2-5 keywords |

**Critical Rule**: NEVER use `grep`/`find`/`read_file` before calling `search_memory`.

### How to Call

**Step-by-step**:
1. Read project ID: `cat code-memory.json` → extract `project_id`
2. Extract 2-5 keywords from user's request
3. Call tool:

```json
{
  "query": "extracted keywords here",
  "project_id": "uuid-from-code-memory-json",
  "top_k": 10
}
```

### Parameters

- **`query`** (Required): Natural language, 2-5 keywords from user request
  - ✅ Good: "login authentication bug fix"
  - ❌ Bad: "the user wants me to find where the login authentication bug is"
- **`project_id`** (Required): UUID from `code-memory.json`
- **`top_k`** (Optional): 10 (default), 20 (for broad searches)
- **`similarity_threshold`** (Optional): 0.7 (precise), 0.5 (broad)

---

## 🔧 Tool 2: `save_memory` - ALWAYS CALL LAST

### When to Call (MANDATORY Checklist)

Call `save_memory` if **ANY** of these is true:

- ✅ Made code changes (edit/create/delete files)
- ✅ Answered a question
- ✅ Fixed a bug or error
- ✅ Ran commands or tests
- ✅ Explained code or concepts
- ✅ Debugged an issue
- ✅ Created new features
- ✅ Refactored code
- ✅ Provided recommendations
- ✅ **ANY user interaction occurred**

**If you completed a response, you MUST call `save_memory`.**

### How to Call

#### For Code Changes:

```json
{
  "title": "[short description]",
  "content": "## File: /absolute/path/to/file.ext\n\n
### Change Summary\n
[What changed and why]\n\n
### Code Changes\n
```language\n
[actual code]\n
```\n\n
### Context\n
- Related files: [list]\n
- Breaking changes: No\n
- Impact: [description]",
  "tags": ["code_change", "bugfix", "component_name"],
  "metadata": {
    "file_path": "/absolute/path/to/file.ext",
    "change_type": "modified",
    "affected_files": ["/path/to/related.ext"]
  },
  "project_id": "uuid-from-code-memory-json"
}
```

#### For Q&A / Explanations:

```json
{
  "title": "[short description]",
  "content": "## Question: [user's question]\n\n
### Answer\n
[your explanation]\n\n
### Key Points\n
- Point 1\n
- Point 2",
  "tags": ["question", "explanation", "topic_name"],
  "project_id": "uuid-from-code-memory-json"
}
```

#### For Debugging / Fixes:

```json
{
  "title": "[short description]",
  "content": "## Issue: [problem]\n\n
### Root Cause\n
[explanation]\n\n
### Solution\n
[what was done]\n\n
### Code Changes\n
```language\n
[code]\n
```",
  "tags": ["debug", "bugfix", "error", "component_name"],
  "metadata": {
    "file_path": "/path/to/fixed/file.ext",
    "change_type": "modified"
  },
  "project_id": "uuid-from-code-memory-json"
}
```

### Parameters
- **`title`** (Required): Short description of the change
- **`content`** Markdown format with:
  - ✅ Headers (`##`, `###`)
  - ✅ Code blocks with language (```typescript, ```python)
  - ✅ Absolute file paths
  - ✅ Clear, searchable descriptions
- **`tags`** (Required): Array of categorization labels
  - Code: `["code_change", "bugfix"/"feature"/"refactor", "component"]`
  - Q&A: `["question", "explanation", "topic"]`
  - Debug: `["debug", "error", "solution", "bugfix"]`
- **`metadata`** (Optional): Additional context
  - `file_path`: Absolute path
  - `change_type`: "added"/"modified"/"deleted"/"refactored"
  - `affected_files`: Array of related paths
- **`project_id`** (Required): UUID from `code-memory.json`

---

## 🔍 Supporting Tools (Use as Needed)

### `get_memories` - Paginated browsing
- **When**: Review many memories at once
- **Call**: `{ page: 1, limit: 20, tags: ["code_change"], project_id: "..." }`

### `get_recent_memories` - Recent activity
- **When**: Quick recap of latest work
- **Call**: `{ days: 7, limit: 10, project_id: "..." }`

### `delete_memory` - Remove outdated info
- **When**: Memory is wrong or superseded
- **Call**: `{ memory_id: "uuid-of-memory-to-delete" }`

### `create_project` - Initialize project
- **When**: First time using memory system
- **Call**: `{ name: "...", repo_url: "from git remote -v", technologies: [...] }`

---

## ✅ Complete Workflow Example

**User asks**: "Fix the login bug"

**Your Actions**:
```
1. Read project ID
   → cat code-memory.json
   → Extract project_id: "abc-123"

2. Search memories FIRST (MANDATORY)
   → search_memory({
       query: "login bug fix authentication",
       project_id: "abc-123",
       top_k: 10
     })
   → Review results

3. Search code if needed
   → grep_search({ query: "login" })
   → read_file({ path: "/src/auth/login.ts" })

4. Fix the bug
   → edit({ file_path: "/src/auth/login.ts", ... })

5. Save the fix (MANDATORY)
   → save_memory({
       title: "Fixed null pointer in login function",
       content: "## File: /src/auth/login.ts\n\n### Change Summary\nFixed null pointer...",
       tags: ["code_change", "bugfix", "auth"],
       metadata: { file_path: "/src/auth/login.ts", change_type: "modified" },
       project_id: "abc-123"
     })

6. Respond to user
   → "Fixed the login bug by adding null check..."
```

---

## 🚨 Critical Rules (NEVER BREAK)

### Rule 1: Search First
**NEVER** use `grep`/`find`/`read_file` before calling `search_memory`.

❌ **Wrong**:
```
User: "Find the API endpoint"
You: grep_search(...) ← NO!
```

✅ **Correct**:
```
User: "Find the API endpoint"
You: search_memory({ query: "API endpoint" }) ← YES!
     Then: grep_search(...) if needed
```

### Rule 2: Save Last
**NEVER** end a response without calling `save_memory`.

❌ **Wrong**:
```
You: [make code changes]
You: [respond to user]
You: [END] ← NO!
```

✅ **Correct**:
```
You: [make code changes]
You: save_memory({ ... }) ← YES!
You: [respond to user]
```

### Rule 3: No Exceptions
**EVEN** trivial interactions must be saved.

**Example**:
```
User: "What's the project name?"
You: 
  1. search_memory({ query: "project name" })
  2. Answer: "The project is XYZ"
  3. save_memory({ content: "## Question: Project Name\n\nAnswer: XYZ", tags: ["question"] })
```

---

## 💡 Best Practices

### Keyword Extraction
- **User says**: "Refactor authentication middleware to support OAuth"
- **Extract**: "authentication middleware OAuth refactor"
- **Call**: `search_memory({ query: "authentication middleware OAuth refactor" })`

### Memory Quality Checklist
- ✅ **Specific**: Include file paths, function names, line numbers
- ✅ **Complete**: Show full code snippets with context
- ✅ **Structured**: Use markdown headers and code blocks
- ✅ **Searchable**: Use clear, descriptive language

### Common Mistakes to Avoid

| Mistake | Impact | Fix |
|---------|--------|-----|
| Skip `search_memory` | Miss existing knowledge | Always call first |
| Skip `save_memory` | Lose work for future | Always call last |
| Vague content | Hard to find later | Be specific with paths |
| Missing file paths | Can't locate code | Always use absolute paths |
| Wrong tags | Poor searchability | Use consistent tags |

---

## 📝 Project ID: How to Get It

**Every tool call needs `project_id`**:

```bash
# Read the config file
cat code-memory.json

# Output:
# {
#   "project_id": "abc-123-def-456"
# }

# Use this UUID in all tool calls
```

**If file doesn't exist**: Call `create_project` first.

---

## ⚠️ What NOT to Do

### ❌ Failure Mode 1: Skipping Search
```
User: "Find the login function"
You: grep_search({ query: "login" }) ← WRONG! Search memory first!
```

### ❌ Failure Mode 2: Skipping Save
```
You: [fix bug]
You: "Fixed!" ← WRONG! Must call save_memory!
```

### ❌ Failure Mode 3: Poor Content
```
save_memory({
  title: "Updated stuff", ← WRONG! Too vague!
  content: "updated stuff", ← WRONG! Too vague!
  tags: ["misc"] ← WRONG! Not searchable!
})
```

### ✅ Correct Approach
```
1. search_memory({ query: "login function" })
2. grep_search({ query: "login" })
3. [fix bug]
4. save_memory({
     title: "Fixed null check in login function",
     content: "## File: /src/auth/login.ts\n\n### Change Summary\nFixed null check...",
     tags: ["code_change", "bugfix", "auth"],
     metadata: { file_path: "/src/auth/login.ts" },
     project_id: "abc-123"
   })
```

---

## 🎓 Critical Reminders

**Remember**:
1. **SEARCH FIRST**: `search_memory` is your opening move
2. **SAVE LAST**: `save_memory` is your closing move
3. **NO EXCEPTIONS**: Every interaction needs both
4. **BE SPECIFIC**: File paths, code blocks, clear descriptions
5. **USE TAGS**: Make memories searchable

**Your primary responsibility is maintaining the memory system.**